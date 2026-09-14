package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/teexue/nexa/core/service"
)

func TestHandleAgents(t *testing.T) {
	srv, _ := setupTestServer(t)
	router := srv.Handler()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/v1/agents", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var items []AgentListItem
	if err := json.Unmarshal(w.Body.Bytes(), &items); err != nil {
		t.Fatalf("failed to unmarshal agents: %v", err)
	}
	if len(items) != 1 {
		t.Errorf("expected 1 agent, got %d", len(items))
	}
	if items[0].Name != "test" {
		t.Errorf("expected agent name 'test', got %q", items[0].Name)
	}
	if items[0].Provider != "mock" {
		t.Errorf("expected provider 'mock', got %q", items[0].Provider)
	}
	if items[0].Model != "test-model" {
		t.Errorf("expected model 'test-model', got %q", items[0].Model)
	}
}

func TestNormalizeAgentName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"test", "test"},
		{"test.yaml", "test"},
		{"my-agent.yaml", "my-agent"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := service.NormalizeAgentName(tt.input); got != tt.want {
				t.Errorf("NormalizeAgentName(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestHandleAgents_PartialLoadFailure(t *testing.T) {
	srv, dir := setupTestServer(t)

	// Add an invalid agent file.
	if err := os.WriteFile(filepath.Join(dir, "invalid.yaml"), []byte("name: invalid\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	router := srv.Handler()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/v1/agents", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var items []AgentListItem
	if err := json.Unmarshal(w.Body.Bytes(), &items); err != nil {
		t.Fatalf("failed to unmarshal agents: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 valid agent, got %d", len(items))
	}
	if items[0].Name != "test" {
		t.Fatalf("expected agent 'test', got %q", items[0].Name)
	}
}

func TestHandleAgentGet(t *testing.T) {
	srv, _ := setupTestServer(t)
	router := srv.Handler()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/v1/agents/test", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var detail AgentDetail
	if err := json.Unmarshal(w.Body.Bytes(), &detail); err != nil {
		t.Fatalf("failed to unmarshal agent detail: %v", err)
	}
	if detail.Name != "test" {
		t.Fatalf("expected name 'test', got %q", detail.Name)
	}
	if detail.Provider != "mock" {
		t.Fatalf("expected provider 'mock', got %q", detail.Provider)
	}
	if detail.SystemPrompt == "" {
		t.Fatal("expected non-empty system_prompt")
	}
	if detail.MaxTurns != 5 {
		t.Fatalf("expected max_turns 5, got %d", detail.MaxTurns)
	}
	if detail.MaxTokens != 1024 {
		t.Fatalf("expected max_tokens 1024, got %d", detail.MaxTokens)
	}
}

func TestHandleAgentGet_NotFound(t *testing.T) {
	srv, _ := setupTestServer(t)
	router := srv.Handler()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/v1/agents/nonexistent", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", w.Code)
	}
}

func TestHandleAgentGet_WithYamlSuffix(t *testing.T) {
	srv, _ := setupTestServer(t)
	router := srv.Handler()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/v1/agents/test.yaml", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var detail AgentDetail
	if err := json.Unmarshal(w.Body.Bytes(), &detail); err != nil {
		t.Fatalf("failed to unmarshal agent detail: %v", err)
	}
	if detail.Name != "test" {
		t.Fatalf("expected name 'test', got %q", detail.Name)
	}
}
