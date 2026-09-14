package config_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/teexue/nexa/core/config"
	"github.com/teexue/nexakit/provider"
	"github.com/teexue/nexa/core/store"
)

func bindTestDB(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	db, err := config.OpenAndBind(home)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = db.Close()
		config.BindDB(nil)
	})
	return home
}

func TestCredentialsRoundTrip(t *testing.T) {
	dir := bindTestDB(t)
	if err := config.SetCredential(dir, "MOONSHOT_API_KEY", "sk-test"); err != nil {
		t.Fatal(err)
	}
	if err := config.LoadCredentials(dir); err != nil {
		t.Fatal(err)
	}
	if got := config.GetCredential("MOONSHOT_API_KEY"); got != "sk-test" {
		t.Fatalf("got %q", got)
	}
	info, err := os.Stat(store.StateFile(dir))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("state.db mode = %o, want 600", info.Mode().Perm())
	}
}

func TestUpsertProvider(t *testing.T) {
	dir := bindTestDB(t)
	spec := config.ProviderSpec{
		Name:         "moonshot",
		APIStyle:     provider.StyleOpenAI,
		BaseURL:      "https://api.moonshot.cn/v1",
		APIKeyEnv:    "MOONSHOT_API_KEY",
		DefaultModel: "kimi-k2.6",
		ThinkingType: "disabled",
	}
	if err := config.UpsertProvider(dir, spec); err != nil {
		t.Fatal(err)
	}
	catalog, err := config.DB().LoadCatalog(nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(catalog.Names()) != 1 {
		t.Fatalf("providers = %v", catalog.Names())
	}
	env, ok := config.LookupAPIKeyEnv("moonshot")
	if !ok || env != "MOONSHOT_API_KEY" {
		t.Fatalf("LookupAPIKeyEnv = %q ok=%v", env, ok)
	}
}

func TestMergeProviderModelWindow(t *testing.T) {
	dir := bindTestDB(t)
	spec := config.ProviderSpec{
		Name:         "ollama",
		APIStyle:     provider.StyleOllama,
		DefaultModel: "glm5",
	}
	if err := config.UpsertProvider(dir, spec); err != nil {
		t.Fatal(err)
	}
	changed, err := config.MergeProviderModelWindow(dir, "ollama", "glm5", 1_000_000)
	if err != nil {
		t.Fatal(err)
	}
	if !changed {
		t.Fatal("expected first merge to change")
	}
	changed, err = config.MergeProviderModelWindow(dir, "ollama", "glm5", 1_000_000)
	if err != nil {
		t.Fatal(err)
	}
	if changed {
		t.Fatal("same window should not change")
	}
	catalog, err := config.DB().LoadCatalog(nil)
	if err != nil {
		t.Fatal(err)
	}
	if got := catalog.ModelContextWindow("ollama", "glm5"); got != 1_000_000 {
		t.Fatalf("saved window = %d", got)
	}
}

func TestEnsureDirs(t *testing.T) {
	dir := t.TempDir()
	if err := config.EnsureDirs(dir); err != nil {
		t.Fatal(err)
	}
	for _, sub := range []string{"agents", "sessions", "knowledge"} {
		if _, err := os.Stat(filepath.Join(dir, sub)); err != nil {
			t.Fatalf("expected %s dir: %v", sub, err)
		}
	}
	// EnsureDirs must NOT pre-install any vendor provider or agent.
	if _, err := os.Stat(config.ProvidersFile(dir)); !os.IsNotExist(err) {
		t.Fatalf("providers.yaml should not be pre-installed, got err=%v", err)
	}
	if entries, err := os.ReadDir(config.AgentsDir(dir)); err == nil && len(entries) != 0 {
		t.Fatalf("agents dir should be empty, got %d entries", len(entries))
	}
}

func TestLoadSettingsRequiresDB(t *testing.T) {
	config.BindDB(nil)
	_, err := config.LoadSettings(t.TempDir())
	if !errors.Is(err, config.ErrDBNotBound) {
		t.Fatalf("got %v, want ErrDBNotBound", err)
	}
}
