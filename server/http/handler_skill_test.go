package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/teexue/nexa/core/agent"
	"github.com/teexue/nexa/core/config"
	"github.com/teexue/nexakit/provider"
	"github.com/teexue/nexakit/registry"
)

// setupSkillTestServer builds a server whose home layout mirrors production:
// <home>/agents for agent YAMLs, skills dirs derived from <home>.
func setupSkillTestServer(t *testing.T) (*Server, string) {
	t.Helper()
	home := t.TempDir()
	agentsDir := config.AgentsDir(home)
	if err := os.MkdirAll(agentsDir, 0o755); err != nil {
		t.Fatal(err)
	}

	reg := registry.New()
	reg.Register(&mockTool{})
	newProvider := func(a *agent.Agent) (provider.Provider, error) {
		return &provider.MockProvider{}, nil
	}
	srv := NewServer(ServerConfig{AgentsDir: agentsDir, Registry: reg, NewProvider: newProvider})
	return srv, home
}

func doSkillRequest(t *testing.T, srv *Server, method, path string, body any) (*httptest.ResponseRecorder, map[string]any) {
	t.Helper()
	var reader *bytes.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		reader = bytes.NewReader(data)
	} else {
		reader = bytes.NewReader(nil)
	}
	req, _ := http.NewRequestWithContext(t.Context(), method, path, reader)
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)

	var parsed map[string]any
	if len(w.Body.Bytes()) > 0 {
		_ = json.Unmarshal(w.Body.Bytes(), &parsed)
	}
	return w, parsed
}

func TestHandleSkillCreateAndDelete(t *testing.T) {
	srv, home := setupSkillTestServer(t)

	w, _ := doSkillRequest(t, srv, http.MethodPost, "/v1/skills", map[string]any{
		"name": "pdf-tools", "description": "Extract PDF text.", "body": "# PDF",
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("create: status = %d, body = %s", w.Code, w.Body.String())
	}
	if _, err := os.Stat(filepath.Join(config.SkillsDir(home), "pdf-tools", "SKILL.md")); err != nil {
		t.Fatalf("SKILL.md not written: %v", err)
	}

	// Duplicate create → 409.
	w, _ = doSkillRequest(t, srv, http.MethodPost, "/v1/skills", map[string]any{
		"name": "pdf-tools", "description": "Extract PDF text.", "body": "# PDF",
	})
	if w.Code != http.StatusConflict {
		t.Fatalf("duplicate create: status = %d, want 409", w.Code)
	}

	// Delete → 200, file gone.
	w, _ = doSkillRequest(t, srv, http.MethodDelete, "/v1/skills/pdf-tools", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("delete: status = %d", w.Code)
	}
	if _, err := os.Stat(filepath.Join(config.SkillsDir(home), "pdf-tools")); !os.IsNotExist(err) {
		t.Fatalf("skill dir should be removed, stat err = %v", err)
	}
}

func TestHandleSkillCreateValidation(t *testing.T) {
	srv, _ := setupSkillTestServer(t)

	tests := []struct {
		name string
		body map[string]any
		want int
	}{
		{"invalid name", map[string]any{"name": "PDF--Tools", "description": "d", "body": "b"}, http.StatusBadRequest},
		{"missing description", map[string]any{"name": "pdf-tools", "body": "b"}, http.StatusBadRequest},
		{"agent scope without agent", map[string]any{"name": "pdf-tools", "description": "d", "body": "b", "scope": "agent"}, http.StatusBadRequest},
		{"bad scope", map[string]any{"name": "pdf-tools", "description": "d", "body": "b", "scope": "weird"}, http.StatusBadRequest},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w, _ := doSkillRequest(t, srv, http.MethodPost, "/v1/skills", tt.body)
			if w.Code != tt.want {
				t.Fatalf("status = %d, want %d (body: %s)", w.Code, tt.want, w.Body.String())
			}
		})
	}
}

func TestHandleSkillAgentScope(t *testing.T) {
	srv, home := setupSkillTestServer(t)

	w, _ := doSkillRequest(t, srv, http.MethodPost, "/v1/skills", map[string]any{
		"name": "private-skill", "description": "Agent only.", "body": "# P",
		"scope": "agent", "agent": "demo",
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("create: status = %d, body = %s", w.Code, w.Body.String())
	}
	if _, err := os.Stat(filepath.Join(config.AgentSkillsDir(home, "demo"), "private-skill", "SKILL.md")); err != nil {
		t.Fatalf("agent-scoped SKILL.md not written: %v", err)
	}

	// List without filter shows the agent scope tag.
	req, _ := http.NewRequestWithContext(t.Context(), http.MethodGet, "/v1/skills", nil)
	w = httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)
	var list []map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &list); err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0]["scope"] != "agent" || list[0]["agent"] != "demo" {
		t.Fatalf("list = %v", list)
	}
}

func TestHandleSkillGetAndUpdate(t *testing.T) {
	srv, _ := setupSkillTestServer(t)

	doSkillRequest(t, srv, http.MethodPost, "/v1/skills", map[string]any{
		"name": "pdf-tools", "description": "v1 desc", "body": "# v1",
	})

	// Get returns the body for editing.
	req, _ := http.NewRequestWithContext(t.Context(), http.MethodGet, "/v1/skills/pdf-tools", nil)
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)
	var detail map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &detail); err != nil {
		t.Fatal(err)
	}
	if detail["body"] != "# v1" || detail["description"] != "v1 desc" {
		t.Fatalf("detail = %v", detail)
	}

	// Update rewrites content.
	w, _ = doSkillRequest(t, srv, http.MethodPut, "/v1/skills/pdf-tools", map[string]any{
		"description": "v2 desc", "body": "# v2",
	})
	if w.Code != http.StatusOK {
		t.Fatalf("update: status = %d, body = %s", w.Code, w.Body.String())
	}
	req, _ = http.NewRequestWithContext(t.Context(), http.MethodGet, "/v1/skills/pdf-tools", nil)
	w = httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)
	if err := json.Unmarshal(w.Body.Bytes(), &detail); err != nil {
		t.Fatal(err)
	}
	if detail["body"] != "# v2" || detail["description"] != "v2 desc" {
		t.Fatalf("after update, detail = %v", detail)
	}
}

func TestHandleSkillsInstallDirectURL(t *testing.T) {
	fileSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("---\nname: web-skill\ndescription: From the web.\n---\n\n# Web\n"))
	}))
	defer fileSrv.Close()

	srv, home := setupSkillTestServer(t)
	w, resp := doSkillRequest(t, srv, http.MethodPost, "/v1/skills/install", map[string]any{
		"url": fileSrv.URL + "/SKILL.md",
	})
	if w.Code != http.StatusOK {
		t.Fatalf("install: status = %d, body = %s", w.Code, w.Body.String())
	}
	installed, _ := resp["installed"].([]any)
	if len(installed) != 1 || installed[0] != "web-skill" {
		t.Fatalf("installed = %v", resp)
	}
	if _, err := os.Stat(filepath.Join(config.SkillsDir(home), "web-skill", "SKILL.md")); err != nil {
		t.Fatalf("installed SKILL.md missing: %v", err)
	}
}

func TestHandleSkillsInstallBadURL(t *testing.T) {
	srv, _ := setupSkillTestServer(t)
	w, _ := doSkillRequest(t, srv, http.MethodPost, "/v1/skills/install", map[string]any{
		"url": "https://example.com/not-a-skill.zip",
	})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}
