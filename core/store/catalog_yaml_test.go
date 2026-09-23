package store_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/teexue/nexakit/provider"
	kitcatalog "github.com/teexue/nexakit/provider/catalog"

	"github.com/teexue/nexa/core/store"
)

func TestLoadCatalog(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "providers.yaml")
	content := `providers:
  anthropic:
    api_style: anthropic
    api_key_env: ANTHROPIC_API_KEY
  openai:
    api_style: openai
    api_key_env: OPENAI_API_KEY
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	catalog, err := store.LoadProviderCatalog(path, nil)
	if err != nil {
		t.Fatalf("LoadCatalog: %v", err)
	}
	if len(catalog.Names()) != 2 {
		t.Fatalf("expected 2 providers, got %d", len(catalog.Names()))
	}
}

func TestResolveAnthropicProfile(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "test-key")

	dir := t.TempDir()
	path := filepath.Join(dir, "providers.yaml")
	content := `providers:
  anthropic:
    api_style: anthropic
    api_key_env: ANTHROPIC_API_KEY
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	catalog, err := store.LoadProviderCatalog(path, nil)
	if err != nil {
		t.Fatal(err)
	}

	p, err := catalog.ResolveForAgent("anthropic")
	if err != nil {
		t.Fatalf("ResolveForAgent: %v", err)
	}
	if p == nil {
		t.Fatal("expected provider")
	}
}

func TestInvalidAPIKeyEnvValue(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "providers.yaml")
	content := `providers:
  bad:
    api_style: anthropic
    api_key_env: sk-not-a-var-name
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := store.LoadProviderCatalog(path, nil); err == nil {
		t.Fatal("expected error when api_key_env looks like a key")
	}
}

func TestResolveMissingAPIKey(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "providers.yaml")
	content := `providers:
  anthropic:
    api_style: anthropic
    api_key_env: ANTHROPIC_API_KEY
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	os.Unsetenv("ANTHROPIC_API_KEY")

	catalog, err := store.LoadProviderCatalog(path, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := catalog.ResolveForAgent("anthropic"); err == nil {
		t.Fatal("expected error for missing api key")
	}
}

func TestResolveVendorDefaults(t *testing.T) {
	t.Setenv("MOONSHOT_API_KEY", "test-key")
	dir := t.TempDir()
	path := filepath.Join(dir, "providers.yaml")
	// moonshot entry omits base_url/models_path; resolve should derive them
	// from the built-in vendor registry.
	content := `providers:
  moonshot:
    api_style: openai
    api_key_env: MOONSHOT_API_KEY
    default_model: kimi-k2.6
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	catalog, err := store.LoadProviderCatalog(path, nil)
	if err != nil {
		t.Fatal(err)
	}
	prof, err := catalog.Get("moonshot")
	if err != nil {
		t.Fatal(err)
	}
	if prof.BaseURL != "https://api.moonshot.cn/v1" {
		t.Fatalf("base_url = %q, want vendor default", prof.BaseURL)
	}
	if prof.ModelsPath != "/models" {
		t.Fatalf("models_path = %q, want /models", prof.ModelsPath)
	}
	if prof.DisplayName != "Moonshot (Kimi)" {
		t.Fatalf("display_name = %q, want vendor display name", prof.DisplayName)
	}
}

func TestResolveVendorAnthropicAuth(t *testing.T) {
	t.Setenv("MOONSHOT_API_KEY", "test-key")
	dir := t.TempDir()
	path := filepath.Join(dir, "providers.yaml")
	// Moonshot's Anthropic endpoint uses Bearer auth and a distinct base URL.
	content := `providers:
  moonshot:
    api_style: anthropic
    api_key_env: MOONSHOT_API_KEY
    default_model: kimi-k2.6
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	catalog, err := store.LoadProviderCatalog(path, nil)
	if err != nil {
		t.Fatal(err)
	}
	prof, err := catalog.Get("moonshot")
	if err != nil {
		t.Fatal(err)
	}
	if prof.BaseURL != "https://api.moonshot.cn/anthropic" {
		t.Fatalf("base_url = %q, want anthropic endpoint", prof.BaseURL)
	}
	if prof.AuthStyle != provider.AuthBearer {
		t.Fatalf("auth_style = %q, want bearer", prof.AuthStyle)
	}
	if prof.ModelsPath != "/v1/models" {
		t.Fatalf("models_path = %q, want /v1/models", prof.ModelsPath)
	}
}

func TestListingProfileDualVendor(t *testing.T) {
	// Moonshot configured as Anthropic: listing should switch to the OpenAI endpoint.
	anthropic := kitcatalog.Profile{
		Name:      "moonshot",
		APIStyle:  provider.StyleAnthropic,
		BaseURL:   "https://api.moonshot.cn/anthropic",
		APIKey:    "k",
		AuthStyle: provider.AuthBearer,
	}
	got := kitcatalog.ListingProfile(anthropic)
	if got.APIStyle != provider.StyleOpenAI {
		t.Fatalf("listing api_style = %q, want openai", got.APIStyle)
	}
	if got.BaseURL != "https://api.moonshot.cn/v1" {
		t.Fatalf("listing base_url = %q, want openai endpoint", got.BaseURL)
	}
	if got.AuthStyle != provider.AuthBearer {
		t.Fatalf("listing auth_style = %q, want bearer", got.AuthStyle)
	}
	if got.ModelsPath != "/models" {
		t.Fatalf("listing models_path = %q, want /models", got.ModelsPath)
	}

	// Pure-Anthropic vendor keeps its native style.
	native := kitcatalog.Profile{
		Name:      "anthropic",
		APIStyle:  provider.StyleAnthropic,
		BaseURL:   "https://api.anthropic.com",
		APIKey:    "k",
		AuthStyle: provider.AuthXAPIKey,
	}
	if got := kitcatalog.ListingProfile(native); got.APIStyle != provider.StyleAnthropic {
		t.Fatalf("anthropic listing api_style = %q, want anthropic", got.APIStyle)
	}

	// Custom Anthropic base URL on a dual vendor is respected (user override).
	custom := anthropic
	custom.BaseURL = "https://custom.example.com"
	if got := kitcatalog.ListingProfile(custom); got.BaseURL != "https://custom.example.com" || got.APIStyle != provider.StyleAnthropic {
		t.Fatalf("custom base_url listing should be respected, got %+v", got)
	}
}

func TestModelContextWindow(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "providers.yaml")
	content := `providers:
  ollama:
    api_style: ollama
    default_model: glm5
    model_windows:
      glm5: 1000000
      other: 8192
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	catalog, err := store.LoadProviderCatalog(path, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got := catalog.ModelContextWindow("ollama", "glm5"); got != 1_000_000 {
		t.Fatalf("glm5 window = %d", got)
	}
	if got := catalog.ModelContextWindow("ollama", "missing"); got != 0 {
		t.Fatalf("missing model = %d", got)
	}
	infos := catalog.Entries()
	if len(infos) != 1 || infos[0].ContextWindow != 1_000_000 {
		t.Fatalf("provider info window = %+v", infos)
	}
}

func TestMergeModelWindows(t *testing.T) {
	got := kitcatalog.MergeModelWindows(map[string]int{"a": 1000}, map[string]int{"b": 2000, "a": 0})
	if got["a"] != 1000 || got["b"] != 2000 {
		t.Fatalf("merge = %v", got)
	}
	if kitcatalog.MergeModelWindows(nil, nil) != nil {
		t.Fatal("empty merge should be nil")
	}
}

func TestMissingCatalog(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "providers.yaml")
	if _, err := store.LoadProviderCatalog(path, nil); err == nil {
		t.Fatal("expected error for missing providers file")
	} else if !kitcatalog.IsMissingCatalogError(err) {
		t.Fatalf("expected missingCatalogError, got %v", err)
	}

	// Empty providers map should also be treated as missing.
	if err := os.WriteFile(path, []byte("providers: {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := store.LoadProviderCatalog(path, nil); err == nil {
		t.Fatal("expected error for empty providers")
	} else if !kitcatalog.IsMissingCatalogError(err) {
		t.Fatalf("expected missingCatalogError for empty, got %v", err)
	}
}
