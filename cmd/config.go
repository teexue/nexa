package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/teexue/nexa/core/config"
	"github.com/teexue/nexa/core/i18n"
	"github.com/teexue/nexakit/provider"
	"github.com/teexue/nexa/core/store"
)

// bindHomeDB opens state.db for config CLI commands and returns a closer.
func bindHomeDB(home string) (func(), error) {
	db, err := openStateDB(home, slog.Default())
	if err != nil {
		return nil, err
	}
	return func() { _ = db.Close() }, nil
}

func runConfig(args []string) {
	if len(args) == 0 {
		configUsage()
		os.Exit(1)
	}
	switch args[0] {
	case "init":
		runConfigInit(args[1:])
	case "show":
		runConfigShow(args[1:])
	case "path":
		runConfigPath(args[1:])
	case "set-key":
		runConfigSetKey(args[1:])
	case "set":
		runConfigSet(args[1:])
	default:
		configUsage()
		os.Exit(1)
	}
}

func configUsage() {
	fmt.Fprint(os.Stderr, i18n.T("cli.usage.config"))
}

func runConfigInit(args []string) {
	fs := flag.NewFlagSet("config init", flag.ExitOnError)
	homeFlag := fs.String("home", "", i18n.T("cli.flag.home"))
	_ = fs.Parse(args)

	home, err := config.Home(true)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if *homeFlag != "" {
		home = *homeFlag
		if err := config.EnsureDirs(home); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}

	if err := config.InitInteractive(home); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func runConfigShow(args []string) {
	fs := flag.NewFlagSet("config show", flag.ExitOnError)
	homeFlag := fs.String("home", "", i18n.T("cli.flag.home_short"))
	_ = fs.Parse(args)

	paths, err := resolvePaths(*homeFlag)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	closeDB, err := bindHomeDB(paths.home)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer closeDB()
	printPaths(paths)

	settings, err := config.LoadSettings(paths.home)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Print(i18n.T("cli.config.default_agent", "name", settings.DefaultAgent))
	fmt.Println()

	creds, err := config.NewCredentialStore(paths.home)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	credKeys := creds.Keys()
	if len(credKeys) > 0 {
		fmt.Println(i18n.T("cli.config.credentials_header"))
		for _, k := range credKeys {
			fmt.Printf("  - %s\n", k)
		}
	}

	catalog, err := config.DB().LoadCatalog(nil)
	if err != nil {
		fmt.Fprint(os.Stderr, i18n.T("cli.config.providers_not_configured", "error", err.Error()))
		fmt.Fprintln(os.Stderr)
		return
	}
	fmt.Println(i18n.T("cli.config.providers_header"))
	for _, name := range catalog.Names() {
		fmt.Printf("  - %s\n", name)
	}
	_ = store.StateFile(paths.home)
}

func runConfigPath(args []string) {
	fs := flag.NewFlagSet("config path", flag.ExitOnError)
	_ = fs.Parse(args)
	paths, err := defaultPaths()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println(paths.home)
}

func runConfigSetKey(args []string) {
	fs := flag.NewFlagSet("config set-key", flag.ExitOnError)
	homeFlag := fs.String("home", "", i18n.T("cli.flag.home_short"))
	_ = fs.Parse(args)

	if len(fs.Args()) != 2 {
		fmt.Fprintln(os.Stderr, i18n.T("cli.usage.config_set_key"))
		os.Exit(1)
	}
	paths, err := resolvePaths(*homeFlag)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	closeDB, err := bindHomeDB(paths.home)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer closeDB()
	creds, err := config.NewCredentialStore(paths.home)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := creds.Set(fs.Args()[0], fs.Args()[1]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println(i18n.T("cli.config.key_saved", "env", fs.Args()[0], "path", store.StateFile(paths.home)))
}

func runConfigSet(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, i18n.T("cli.usage.config_set"))
		os.Exit(1)
	}
	switch args[0] {
	case "default-agent":
		runConfigSetDefaultAgent(args[1:])
	case "provider":
		runConfigSetProvider(args[1:])
	default:
		fmt.Fprintln(os.Stderr, i18n.T("cli.config.unknown_set_target", "target", args[0]))
		os.Exit(1)
	}
}

func runConfigSetDefaultAgent(args []string) {
	fs := flag.NewFlagSet("config set default-agent", flag.ExitOnError)
	homeFlag := fs.String("home", "", i18n.T("cli.flag.home_short"))
	_ = fs.Parse(args)
	if len(fs.Args()) != 1 {
		fmt.Fprintln(os.Stderr, i18n.T("cli.usage.config_set_default_agent"))
		os.Exit(1)
	}
	paths, err := resolvePaths(*homeFlag)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	closeDB, err := bindHomeDB(paths.home)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer closeDB()
	settings, _ := config.LoadSettings(paths.home)
	settings.DefaultAgent = fs.Args()[0]
	if err := config.SaveSettings(paths.home, settings); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println(i18n.T("cli.config.default_agent_updated"))
}

func runConfigSetProvider(args []string) {
	// Check if flags were passed (for scripted use).
	hasFlags := len(args) > 0 && strings.HasPrefix(args[0], "-")
	// Check if a provider name was passed positionally with additional flags.
	hasPositionalWithFlags := len(args) >= 2 && !strings.HasPrefix(args[0], "-") && !strings.HasPrefix(args[1], "-")

	if hasFlags || hasPositionalWithFlags {
		runConfigSetProviderFlags(args)
		return
	}

	// Interactive mode: no flags or only --home.
	homeFlag := ""
	if len(args) >= 2 && args[0] == "--home" {
		homeFlag = args[1]
	} else if len(args) == 1 && strings.HasPrefix(args[0], "--home=") {
		homeFlag = strings.TrimPrefix(args[0], "--home=")
	}

	paths, err := resolvePaths(homeFlag)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	closeDB, err := bindHomeDB(paths.home)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer closeDB()

	spec, _, apiKeyEnv, err := config.RunProviderWizard()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if err := config.UpsertProvider(paths.home, spec); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println(i18n.T("cli.config.provider_updated", "name", spec.Name, "path", store.StateFile(paths.home)))
	saveWizardAPIKey(paths.home, apiKeyEnv)
}

func saveWizardAPIKey(home, apiKeyEnv string) {
	creds, err := config.NewCredentialStore(home)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if creds.Lookup(apiKeyEnv) != "" {
		return
	}
	apiKey, err := config.InputSecret(i18n.T("wizard.input.api_key"))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if strings.TrimSpace(apiKey) == "" {
		return
	}
	if err := creds.Set(apiKeyEnv, apiKey); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println(i18n.T("cli.config.key_saved", "env", apiKeyEnv, "path", store.StateFile(home)))
}

func runConfigSetProviderFlags(args []string) {
	fs := flag.NewFlagSet("config set provider", flag.ExitOnError)
	homeFlag := fs.String("home", "", i18n.T("cli.flag.home_short"))
	apiStyle := fs.String("api-style", "openai", i18n.T("cli.flag.api_style"))
	baseURL := fs.String("base-url", "", i18n.T("cli.flag.base_url"))
	apiKeyEnv := fs.String("api-key-env", "", i18n.T("cli.flag.api_key_env"))
	apiVersion := fs.String("api-version", "", i18n.T("cli.flag.api_version"))
	model := fs.String("model", "", i18n.T("cli.flag.model"))
	displayName := fs.String("display-name", "", i18n.T("cli.flag.display_name"))
	modelsPath := fs.String("models-path", "", i18n.T("cli.flag.models_path"))
	authStyle := fs.String("auth-style", "", i18n.T("cli.flag.auth_style"))
	thinking := fs.String("thinking", "", i18n.T("cli.flag.thinking"))
	thinkingKeep := fs.String("thinking-keep", "", i18n.T("cli.flag.thinking_keep"))
	_ = fs.Parse(args)

	if len(fs.Args()) != 1 {
		fmt.Fprintln(os.Stderr, i18n.T("cli.usage.config_set_provider"))
		os.Exit(1)
	}
	paths, err := resolvePaths(*homeFlag)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	closeDB, err := bindHomeDB(paths.home)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer closeDB()

	spec := config.ProviderSpec{
		Name:         fs.Args()[0],
		APIStyle:     provider.APIStyle(*apiStyle),
		BaseURL:      *baseURL,
		APIKeyEnv:    *apiKeyEnv,
		APIVersion:   *apiVersion,
		AuthStyle:    provider.AuthStyle(*authStyle),
		DefaultModel: *model,
		DisplayName:  *displayName,
		ModelsPath:   *modelsPath,
		ThinkingType: *thinking,
		ThinkingKeep: *thinkingKeep,
	}
	if spec.APIKeyEnv == "" {
		spec.APIKeyEnv = strings.ToUpper(spec.Name) + "_API_KEY"
	}
	if spec.ModelsPath == "" {
		spec.ModelsPath = provider.DefaultModelsPathFor(spec.APIStyle)
	}
	if err := config.UpsertProvider(paths.home, spec); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println(i18n.T("cli.config.provider_updated", "name", spec.Name, "path", store.StateFile(paths.home)))
}
