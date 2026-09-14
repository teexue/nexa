package service_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/teexue/nexa/core/service"
	"github.com/teexue/nexa/core/store"
	"github.com/teexue/nexakit/provider"
	"github.com/teexue/nexakit/session"
)

func writeSharedModelCatalog(t *testing.T) *provider.Catalog {
	t.Helper()
	t.Setenv("OPENAI_API_KEY", "test-key")
	path := filepath.Join(t.TempDir(), "providers.yaml")
	content := `providers:
  openai:
    api_style: openai
    api_key_env: OPENAI_API_KEY
    default_model: gpt-4o
    models: [gpt-4o]
  azure:
    api_style: openai
    api_key_env: OPENAI_API_KEY
    default_model: gpt-4o
    models: [gpt-4o]
`
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
	catalog, err := store.LoadProviderCatalog(path, nil)
	require.NoError(t, err)
	return catalog
}

func writeCatalogAgent(t *testing.T, dir, providerName string) {
	t.Helper()
	yaml := `id: agt_wd
name: wd-demo
provider: ` + providerName + `
model: gpt-4o
system_prompt: you are a helper
tools: [get_time]
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "agt_wd.yaml"), []byte(yaml), 0o644))
}

func TestPrepareRun_PrefersAgentProviderForSharedModel(t *testing.T) {
	agentsDir := t.TempDir()
	writeCatalogAgent(t, agentsDir, "openai")
	svc := newWorkdirService(t, agentsDir, newMemStore())
	svc.Catalog = writeSharedModelCatalog(t)

	result, err := svc.PrepareRun(context.Background(), service.RunRequest{
		Agent:  "agt_wd",
		Prompt: "hi",
		Model:  "gpt-4o",
	}, nil)
	require.NoError(t, err)
	assert.Equal(t, "openai", result.Config.Agent.Provider)
	assert.Equal(t, "gpt-4o", result.Config.Agent.Model)
}

func TestPrepareRun_SessionProviderLockRejectsOverride(t *testing.T) {
	agentsDir := t.TempDir()
	writePlainAgent(t, agentsDir)
	store := newMemStore()
	svc := newWorkdirService(t, agentsDir, store)

	sess := session.NewForUser("agt_wd", "usr_local")
	sess.SetMetadata(session.MetadataKeyModel, "locked")
	sess.SetMetadata(session.MetadataKeyProvider, "mock")
	require.NoError(t, store.Save(sess))

	_, err := svc.PrepareRun(context.Background(), service.RunRequest{
		Agent:     "agt_wd",
		Prompt:    "hi",
		SessionID: sess.ID,
		Provider:  "other",
	}, nil)
	require.Error(t, err)
	var arg *service.ArgError
	require.ErrorAs(t, err, &arg)
	assert.Equal(t, "provider", arg.Field)
}
