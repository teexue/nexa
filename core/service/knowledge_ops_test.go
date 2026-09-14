package service_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/teexue/nexa/core/config"
	"github.com/teexue/nexakit/embedding"
	"github.com/teexue/nexa/core/service"
)

func TestListEmbeddingVendors(t *testing.T) {
	svc := &service.Service{}
	list := svc.ListEmbeddingVendors()
	require.Len(t, list, 2)
	names := map[string]bool{}
	for _, v := range list {
		names[v.Name] = true
	}
	assert.True(t, names["qwen"])
	assert.True(t, names["ollama"])
}

func TestSaveAndGetEmbeddingSettings(t *testing.T) {
	home := t.TempDir()
	bindConfigDB(t, home)
	creds, err := config.NewCredentialStore(home)
	require.NoError(t, err)

	svc := &service.Service{HomeDir: home, Creds: creds}
	err = svc.SaveEmbeddingSettings(service.SaveEmbeddingRequest{
		Vendor:     "qwen",
		Backend:    embedding.BackendOpenAI,
		BaseURL:    "https://dashscope.aliyuncs.com/compatible-mode/v1",
		APIKeyEnv:  "DASHSCOPE_API_KEY",
		Model:      "text-embedding-v4",
		Dimensions: 1024,
		APIKey:     "sk-test-emb",
	})
	require.NoError(t, err)
	assert.NotNil(t, svc.Embedder)
	assert.Equal(t, "sk-test-emb", creds.Get("DASHSCOPE_API_KEY"))

	view, err := svc.GetEmbeddingSettings()
	require.NoError(t, err)
	assert.Equal(t, "qwen", view.Vendor)
	assert.Equal(t, "text-embedding-v4", view.Model)
	assert.Equal(t, 1024, view.Dimensions)
	assert.True(t, view.HasAPIKey)
	assert.Equal(t, "DASHSCOPE_API_KEY", view.APIKeyEnv)

	settings, err := config.LoadSettings(home)
	require.NoError(t, err)
	require.NotNil(t, settings.Embedding)
	assert.Equal(t, "qwen", settings.Embedding.Vendor)
	assert.Equal(t, 1024, settings.Embedding.Dimensions)
	assert.Empty(t, settings.Embedding.LegacyProvider)
}

func TestLoadSettingsIgnoresLegacyEmbeddingProvider(t *testing.T) {
	home := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(home, "config.yaml"), []byte(`
default_agent: chat-assistant
embedding:
  provider: moonshot
  backend: openai
  base_url: https://example.com/v1
  api_key_env: EMB_KEY
  model: text-embedding-v3
`), 0o644))
	bindConfigDB(t, home)

	settings, err := config.LoadSettings(home)
	require.NoError(t, err)
	require.NotNil(t, settings.Embedding)
	assert.Empty(t, settings.Embedding.LegacyProvider)
	assert.Equal(t, "text-embedding-v3", settings.Embedding.Model)
}

func TestSaveEmbeddingOllamaNoKey(t *testing.T) {
	home := t.TempDir()
	bindConfigDB(t, home)
	svc := &service.Service{HomeDir: home}
	err := svc.SaveEmbeddingSettings(service.SaveEmbeddingRequest{
		Vendor:  "ollama",
		Backend: embedding.BackendOllama,
		BaseURL: "http://127.0.0.1:11434",
		Model:   "nomic-embed-text",
	})
	require.NoError(t, err)
	assert.NotNil(t, svc.Embedder)

	view, err := svc.GetEmbeddingSettings()
	require.NoError(t, err)
	assert.Equal(t, "ollama", view.Vendor)
	assert.False(t, view.HasAPIKey)
}
