package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/teexue/nexakit/provider"
	kitmock "github.com/teexue/nexakit/provider/mock"
	"github.com/teexue/nexakit/registry"
	"github.com/teexue/nexakit/session"
	"github.com/teexue/nexakit/tool"

	"github.com/teexue/nexa/core/agent"
)

// mockTool is a minimal tool for testing.
type mockTool struct{}

func (m *mockTool) Name() string        { return "test_tool" }
func (m *mockTool) Description() string { return "A test tool" }
func (m *mockTool) InputSchema() map[string]any {
	return map[string]any{
		"type":       "object",
		"properties": map[string]any{},
	}
}
func (m *mockTool) Execute(_ context.Context, _ json.RawMessage) (tool.Result, error) {
	return tool.Result{Output: json.RawMessage(`"ok"`)}, nil
}

func setupTestServer(t *testing.T) (*Server, string) {
	t.Helper()

	// Create temp agents directory.
	dir := t.TempDir()

	// Write a test agent.
	agentContent := `name: test
version: 1
provider: mock
model: test-model
system_prompt: |
  You are a test assistant.
tools:
  - test_tool
max_turns: 5
max_tokens: 1024
`
	if err := os.WriteFile(filepath.Join(dir, "test.yaml"), []byte(agentContent), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create registry with mock tool.
	reg := registry.New()
	reg.Register(&mockTool{})

	// Create server with mock provider factory.
	newProvider := func(a *agent.Agent) (provider.Provider, error) {
		return &kitmock.MockProvider{
			Calls: [][]kitmock.MockStep{
				{{Text: "test response"}},
			},
		}, nil
	}

	srv := NewServer(ServerConfig{AgentsDir: dir, Registry: reg, NewProvider: newProvider})
	return srv, dir
}

func setupTestServerWithProvider(t *testing.T, newProvider func(a *agent.Agent) (provider.Provider, error)) (*Server, string) {
	t.Helper()
	srv, dir := setupTestServer(t)
	srv.newProvider = newProvider
	srv.svc.NewProvider = newProvider
	return srv, dir
}

func setupTestServerWithStore(t *testing.T) (*Server, string, session.Store) {
	t.Helper()
	srv, dir := setupTestServer(t)
	store, err := session.NewFileStore(filepath.Join(dir, "sessions"))
	if err != nil {
		t.Fatal(err)
	}
	srv.SetStore(store)
	return srv, dir, store
}

func TestHandleHealth(t *testing.T) {
	srv, _ := setupTestServer(t)
	router := srv.Handler()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/healthz", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var resp map[string]string
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["status"] != "up" {
		t.Errorf("expected status 'up', got %q", resp["status"])
	}
}
