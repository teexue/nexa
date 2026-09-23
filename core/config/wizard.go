package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/teexue/nexakit/provider"
	kitcatalog "github.com/teexue/nexakit/provider/catalog"

	"github.com/teexue/nexa/core/i18n"
	"github.com/teexue/nexa/core/store"
	"github.com/teexue/nexa/core/tui"
)

// ProviderSpec is CLI input for configuring a provider.
type ProviderSpec struct {
	Name         string
	APIStyle     provider.APIStyle
	BaseURL      string
	APIKeyEnv    string
	APIVersion   string
	AuthStyle    provider.AuthStyle
	DefaultModel string
	DisplayName  string
	Models       []string
	ModelsPath   string
	Vision       bool
	ThinkingType string
	ThinkingKeep string
	ModelWindows map[string]int
}

// InitInteractive runs a wizard to bootstrap ~/.nexa.
func InitInteractive(home string) error {
	if err := ensureHome(home); err != nil {
		return err
	}
	db, err := store.Open(home)
	if err != nil {
		return fmt.Errorf("open state.db: %w", err)
	}
	defer db.Close()
	BindDB(db)

	tui.PrintWelcome("setup", "config", "wizard")
	fmt.Println(tui.Muted(i18n.T("wizard.config_dir", "path", home)))

	spec, agentName, apiKeyEnv, err := RunProviderWizard()
	if err != nil {
		return err
	}

	// Local Ollama needs no API key (apiKeyEnv is empty for the preset); skip
	// the credential prompt and storage in that case.
	if apiKeyEnv != "" {
		apiKey, err := InputSecret(i18n.T("wizard.input.api_key"))
		if err != nil {
			return err
		}
		if strings.TrimSpace(apiKey) == "" {
			return fmt.Errorf("%s", i18n.T("wizard.error.api_key_required"))
		}
		creds, err := NewCredentialStore(home)
		if err != nil {
			return err
		}
		if err := creds.Set(apiKeyEnv, apiKey); err != nil {
			return err
		}
	}

	if err := UpsertProvider(home, spec); err != nil {
		return err
	}

	scPath := fmt.Sprintf("%s/%s.yaml", AgentsDir(home), agentName)
	scContent := fmt.Sprintf(`name: %s
version: 1
provider: %s
model: %s
system_prompt: |
  You are a helpful assistant. Use tools when appropriate.
tools:
  - echo
  - get_time
max_turns: 0
max_tokens: 4096
tool_execution:
  mode: parallel
  max_parallel: 4
`, agentName, spec.Name, spec.DefaultModel)
	if err := os.WriteFile(scPath, []byte(scContent), 0o644); err != nil {
		return fmt.Errorf("write agent: %w", err)
	}

	if err := SaveSettings(home, Settings{DefaultAgent: agentName}); err != nil {
		return err
	}

	fmt.Println(i18n.T("wizard.done"))
	fmt.Println(tui.Success(i18n.T("wizard.config_written", "path", home)))
	fmt.Println(tui.Muted(i18n.T("wizard.run_chat_hint")))
	return nil
}

// RunProviderWizard runs the interactive provider configuration wizard.
// Returns the provider spec, agent name, API key env name, and any error.
func RunProviderWizard() (ProviderSpec, string, string, error) {
	vendors := provider.BuiltInVendors()
	labels := make([]string, 0, len(vendors)+1)
	for _, v := range vendors {
		labels = append(labels, v.DisplayName)
	}
	customLabel := i18n.T("wizard.option.custom")
	labels = append(labels, customLabel)

	choice, err := selectOption(i18n.T("wizard.select.provider"), labels, 0)
	if err != nil {
		return ProviderSpec{}, "", "", err
	}

	var spec ProviderSpec
	var apiKeyEnv string

	if choice == customLabel {
		spec, apiKeyEnv, err = customProviderWizard()
	} else {
		var vendor provider.Vendor
		for _, v := range vendors {
			if v.DisplayName == choice {
				vendor = v
				break
			}
		}
		spec, apiKeyEnv, err = presetProviderWizard(vendor)
	}
	if err != nil {
		return ProviderSpec{}, "", "", err
	}

	agentName, err := selectOrInput(i18n.T("wizard.select.default_agent"), []string{"demo"}, 0, i18n.T("wizard.input.agent_name"))
	if err != nil {
		return ProviderSpec{}, "", "", err
	}

	return spec, agentName, apiKeyEnv, nil
}

func presetProviderWizard(v provider.Vendor) (ProviderSpec, string, error) {
	model, err := selectOrInput(i18n.T("wizard.select.model"), []string{v.DefaultModel}, 0, i18n.T("wizard.input.model_name"))
	if err != nil {
		return ProviderSpec{}, "", err
	}
	if strings.TrimSpace(model) == "" {
		model = v.DefaultModel
	}

	spec := ProviderSpec{
		Name:         v.Name,
		APIStyle:     v.APIStyle,
		BaseURL:      v.BaseURLFor(v.APIStyle),
		APIKeyEnv:    v.APIKeyEnv,
		DefaultModel: model,
		APIVersion:   v.APIVersion,
		AuthStyle:    v.AuthForStyle(v.APIStyle),
		DisplayName:  v.DisplayName,
		ModelsPath:   provider.DefaultModelsPathFor(v.APIStyle),
		Vision:       v.Vision,
	}

	if v.SupportsThinking {
		if err := configureThinking(&spec); err != nil {
			return ProviderSpec{}, "", err
		}
	}

	fmt.Print(i18n.T("wizard.summary.selected", "provider", spec.Name, "type", spec.APIStyle, "model", spec.DefaultModel))
	fmt.Println()
	return spec, v.APIKeyEnv, nil
}

func configureThinking(spec *ProviderSpec) error {
	thinking, err := selectOption(i18n.T("wizard.select.thinking_mode"), []string{"disabled", "enabled"}, 0)
	if err != nil {
		return err
	}
	spec.ThinkingType = thinking
	if thinking != "enabled" {
		return nil
	}
	noKeep := i18n.T("wizard.option.no_keep")
	keep, err := selectOption(i18n.T("wizard.select.keep_reasoning"), []string{noKeep, "all"}, 0)
	if err != nil {
		return err
	}
	if keep == "all" {
		spec.ThinkingKeep = "all"
	}
	return nil
}

func customProviderWizard() (ProviderSpec, string, error) {
	name, err := inputString(i18n.T("wizard.input.provider_name"), "my-provider")
	if err != nil {
		return ProviderSpec{}, "", err
	}

	pStyle, err := selectOption(i18n.T("wizard.select.api_type"), []string{"openai", "anthropic"}, 0)
	if err != nil {
		return ProviderSpec{}, "", err
	}
	style := provider.APIStyle(pStyle)

	baseURL, err := inputString(i18n.T("wizard.input.base_url"), defaultBaseURLHint(style))
	if err != nil {
		return ProviderSpec{}, "", err
	}

	apiKeyEnv, err := inputString(i18n.T("wizard.input.api_key_env"), strings.ToUpper(name)+"_API_KEY")
	if err != nil {
		return ProviderSpec{}, "", err
	}

	model, err := inputString(i18n.T("wizard.input.default_model"), "")
	if err != nil {
		return ProviderSpec{}, "", err
	}

	spec := ProviderSpec{
		Name:         name,
		APIStyle:     style,
		BaseURL:      baseURL,
		APIKeyEnv:    apiKeyEnv,
		DefaultModel: model,
		ModelsPath:   provider.DefaultModelsPathFor(style),
	}

	if style == provider.StyleAnthropic {
		version, err := selectOption(i18n.T("wizard.select.api_version"), []string{"2023-06-01"}, 0)
		if err != nil {
			return ProviderSpec{}, "", err
		}
		spec.APIVersion = version
	}

	if style == provider.StyleOpenAI {
		notSet := i18n.T("wizard.option.not_set")
		thinking, err := selectOption(i18n.T("wizard.select.thinking_mode_custom"), []string{"disabled", "enabled", notSet}, 2)
		if err != nil {
			return ProviderSpec{}, "", err
		}
		if thinking != notSet {
			spec.ThinkingType = thinking
		}
	}

	return spec, apiKeyEnv, nil
}

func defaultBaseURLHint(style provider.APIStyle) string {
	switch style {
	case provider.StyleAnthropic:
		return "https://api.anthropic.com"
	default:
		return "https://api.example.com/v1"
	}
}

// UpsertProvider adds or updates a provider in state.db.
// For updates, empty APIKeyEnv preserves the existing value.
func UpsertProvider(home string, spec ProviderSpec) error {
	providers, err := listProviderEntries()
	if err != nil {
		return err
	}
	existing := providers[spec.Name]
	spec.applyDefaults(existing)
	if err := spec.validate(); err != nil {
		return err
	}
	_ = home
	return writeProviderEntry(spec.Name, spec.toEntry(existing))
}

func listProviderEntries() (map[string]kitcatalog.ProfileEntry, error) {
	db, err := requireDB()
	if err != nil {
		return nil, err
	}
	return db.ListProviderEntries()
}

// LookupAPIKeyEnv returns the persisted api_key_env for a named provider.
func LookupAPIKeyEnv(name string) (string, bool) {
	entries, err := listProviderEntries()
	if err != nil {
		return "", false
	}
	entry, ok := entries[name]
	if !ok || entry.APIKeyEnv == "" {
		return "", false
	}
	return entry.APIKeyEnv, true
}

func writeProviderEntry(name string, entry kitcatalog.ProfileEntry) error {
	db, err := requireDB()
	if err != nil {
		return err
	}
	return db.UpsertProviderEntry(name, entry)
}

func (spec *ProviderSpec) applyDefaults(existing kitcatalog.ProfileEntry) {
	if spec.APIKeyEnv == "" {
		spec.APIKeyEnv = existing.APIKeyEnv
	}
	if spec.APIStyle == "" {
		spec.APIStyle = existing.APIStyle
	}
	if spec.DefaultModel == "" {
		spec.DefaultModel = existing.DefaultModel
	}
	if spec.Models == nil {
		spec.Models = existing.Models
	}
	if spec.BaseURL == "" {
		spec.BaseURL = existing.BaseURL
	}
	if spec.ModelsPath == "" {
		spec.ModelsPath = existing.ModelsPath
	}
	if spec.APIVersion == "" {
		spec.APIVersion = existing.APIVersion
	}
}

func (spec ProviderSpec) toEntry(existing kitcatalog.ProfileEntry) kitcatalog.ProfileEntry {
	entry := kitcatalog.ProfileEntry{
		APIStyle:     spec.APIStyle,
		BaseURL:      spec.BaseURL,
		APIKeyEnv:    spec.APIKeyEnv,
		APIVersion:   spec.APIVersion,
		AuthStyle:    spec.AuthStyle,
		DefaultModel: spec.DefaultModel,
		DisplayName:  spec.DisplayName,
		Models:       spec.modelsOrDefault(),
		ModelsPath:   spec.ModelsPath,
		Vision:       spec.Vision,
		KeepAlive:    existing.KeepAlive,
		ModelWindows: kitcatalog.MergeModelWindows(existing.ModelWindows, spec.ModelWindows),
	}
	if spec.ThinkingType != "" {
		entry.Thinking = &provider.ThinkingConfig{Type: spec.ThinkingType, Keep: spec.ThinkingKeep}
	} else {
		entry.Thinking = existing.Thinking
	}
	return entry
}

// MergeProviderModelWindow records a discovered context window for one model
// on an existing provider. Returns changed=false when the stored value already
// matches, so callers can skip a catalog reload.
func MergeProviderModelWindow(home, name, model string, window int) (bool, error) {
	_ = home
	if name == "" || model == "" || window <= 0 {
		return false, nil
	}
	providers, err := listProviderEntries()
	if err != nil {
		return false, err
	}
	entry, ok := providers[name]
	if !ok {
		return false, fmt.Errorf("provider %q not found", name)
	}
	if entry.ModelWindows[model] == window {
		return false, nil
	}
	entry.ModelWindows = kitcatalog.MergeModelWindows(entry.ModelWindows, map[string]int{model: window})
	if err := writeProviderEntry(name, entry); err != nil {
		return false, err
	}
	return true, nil
}

// DeleteProvider removes a provider from state.db.
func DeleteProvider(home string, name string) error {
	_ = home
	if name == "" {
		return fmt.Errorf("provider name is required")
	}
	db, err := requireDB()
	if err != nil {
		return err
	}
	return db.DeleteProviderEntry(name)
}

func (s ProviderSpec) validate() error {
	if s.Name == "" {
		return fmt.Errorf("%s", i18n.T("wizard.error.provider_name_required"))
	}
	if s.APIStyle != provider.StyleAnthropic && s.APIStyle != provider.StyleOpenAI && s.APIStyle != provider.StyleOllama {
		return fmt.Errorf("%s", i18n.T("wizard.error.type_invalid"))
	}
	// Local Ollama needs no API key, so api_key_env is optional for that style.
	if s.APIKeyEnv == "" && s.APIStyle != provider.StyleOllama {
		return fmt.Errorf("%s", i18n.T("wizard.error.api_key_env_required"))
	}
	if s.DefaultModel == "" {
		return fmt.Errorf("%s", i18n.T("wizard.error.model_required"))
	}
	return s.validateModels()
}

func (s ProviderSpec) modelsOrDefault() []string {
	if len(s.Models) > 0 {
		return s.Models
	}
	if s.DefaultModel != "" {
		return []string{s.DefaultModel}
	}
	return nil
}

func (s ProviderSpec) validateModels() error {
	if len(s.Models) == 0 {
		return nil
	}
	for _, m := range s.Models {
		if m == s.DefaultModel {
			return nil
		}
	}
	return fmt.Errorf("%s", i18n.T("wizard.error.default_model_not_enabled"))
}
