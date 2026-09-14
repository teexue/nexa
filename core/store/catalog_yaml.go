package store

import (
	"fmt"
	"os"

	"github.com/teexue/nexakit/provider"
	"gopkg.in/yaml.v3"
)

type fileThinking struct {
	Type string `yaml:"type"`
	Keep string `yaml:"keep"`
}

type fileProfileEntry struct {
	APIStyle     provider.APIStyle  `yaml:"api_style"`
	BaseURL      string             `yaml:"base_url"`
	APIKeyEnv    string             `yaml:"api_key_env"`
	APIVersion   string             `yaml:"api_version"`
	AuthStyle    provider.AuthStyle `yaml:"auth_style,omitempty"`
	DefaultModel string             `yaml:"default_model"`
	DisplayName  string             `yaml:"display_name,omitempty"`
	Models       []string           `yaml:"models,omitempty"`
	ModelsPath   string             `yaml:"models_path,omitempty"`
	Vision       bool               `yaml:"vision,omitempty"`
	Thinking     *fileThinking      `yaml:"thinking"`
	KeepAlive    string             `yaml:"keep_alive,omitempty"`
	ModelWindows map[string]int     `yaml:"model_windows,omitempty"`
}

func (e fileProfileEntry) toEntry() provider.ProfileEntry {
	out := provider.ProfileEntry{
		APIStyle: e.APIStyle, BaseURL: e.BaseURL, APIKeyEnv: e.APIKeyEnv,
		APIVersion: e.APIVersion, AuthStyle: e.AuthStyle, DefaultModel: e.DefaultModel,
		DisplayName: e.DisplayName, Models: e.Models, ModelsPath: e.ModelsPath,
		Vision: e.Vision, KeepAlive: e.KeepAlive, ModelWindows: e.ModelWindows,
	}
	if e.Thinking != nil {
		out.Thinking = &provider.ThinkingConfig{Type: e.Thinking.Type, Keep: e.Thinking.Keep}
	}
	return out
}

// CatalogFile is the legacy providers.yaml document.
type CatalogFile struct {
	Providers map[string]fileProfileEntry `yaml:"providers"`
}

// Entries converts the file document into runtime provider entries.
func (f CatalogFile) Entries() map[string]provider.ProfileEntry {
	out := make(map[string]provider.ProfileEntry, len(f.Providers))
	for name, entry := range f.Providers {
		out[name] = entry.toEntry()
	}
	return out
}

// LoadProviderCatalog reads providers.yaml into a runtime catalog.
func LoadProviderCatalog(path string, credLookup func(string) string) (*provider.Catalog, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, provider.MissingCatalog(path)
		}
		return nil, fmt.Errorf("read providers %q: %w", path, err)
	}
	var file CatalogFile
	if err := yaml.Unmarshal(data, &file); err != nil {
		return nil, fmt.Errorf("parse providers %q: %w", path, err)
	}
	return provider.NewCatalog(file.Entries(), credLookup)
}
