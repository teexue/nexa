package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/teexue/nexa/core/agent"
	"github.com/teexue/nexakit/provider"
	"github.com/teexue/nexakit/session"
)

// parseSSEEvents parses data: lines from an SSE response body.
func parseSSEEvents(body string) []map[string]any {
	var events []map[string]any
	for _, line := range strings.Split(body, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")
		var ev map[string]any
		if err := json.Unmarshal([]byte(data), &ev); err == nil {
			events = append(events, ev)
		}
	}
	return events
}

func TestHandleRun_MissingParams(t *testing.T) {
	srv, _ := setupTestServer(t)
	router := srv.Handler()

	tests := []struct {
		name string
		body RunRequest
	}{
		{
			name: "missing agent",
			body: RunRequest{Prompt: "hello"},
		},
		{
			name: "missing prompt",
			body: RunRequest{Agent: "test"},
		},
		{
			name: "both missing",
			body: RunRequest{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.body)
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("POST", "/v1/agents/run", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(w, req)

			if w.Code != http.StatusBadRequest {
				t.Errorf("expected status 400, got %d", w.Code)
			}
		})
	}
}

func TestHandleRun_AgentNotFound(t *testing.T) {
	srv, _ := setupTestServer(t)
	router := srv.Handler()

	body, _ := json.Marshal(RunRequest{
		Agent:  "nonexistent",
		Prompt: "hello",
	})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/v1/agents/run", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

func TestHandleRun_HappyPath(t *testing.T) {
	srv, _ := setupTestServer(t)
	router := srv.Handler()

	body, _ := json.Marshal(RunRequest{
		Agent:  "test",
		Prompt: "hello",
	})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/v1/agents/run", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); ct != "text/event-stream" {
		t.Errorf("expected Content-Type text/event-stream, got %q", ct)
	}
	// Verify we got some SSE data.
	if w.Body.Len() == 0 {
		t.Error("expected non-empty response body")
	}
}

func TestHandleRun_SessionResume(t *testing.T) {
	srv, _, store := setupTestServerWithStore(t)
	router := srv.Handler()

	// Create and save a session with existing messages.
	sess := session.New("test")
	sess.SetMessages([]provider.Message{
		{Role: provider.RoleSystem, Content: "system prompt"},
		{Role: provider.RoleUser, Content: "hello"},
	})
	if err := store.Save(sess); err != nil {
		t.Fatal(err)
	}

	// Provider returns a response for the resumed conversation.
	srv.newProvider = func(a *agent.Agent) (provider.Provider, error) {
		return &provider.MockProvider{
			Calls: [][]provider.MockStep{
				{{Text: "resumed response"}},
			},
		}, nil
	}

	body, _ := json.Marshal(RunRequest{
		Agent:     "test",
		Prompt:    "follow-up",
		SessionID: sess.ID,
	})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/v1/agents/run", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}
	if ct := w.Header().Get("Content-Type"); ct != "text/event-stream" {
		t.Fatalf("expected Content-Type text/event-stream, got %q", ct)
	}
	if w.Body.Len() == 0 {
		t.Fatal("expected non-empty SSE response")
	}
}

func TestHandleRun_SessionResumeNotFound(t *testing.T) {
	srv, _, _ := setupTestServerWithStore(t)
	router := srv.Handler()

	body, _ := json.Marshal(RunRequest{
		Agent:     "test",
		Prompt:    "hello",
		SessionID: "does-not-exist",
	})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/v1/agents/run", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Code)
	}
	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if resp["code"] != "invalid_request" && resp["code"] != "run_error" {
		t.Fatalf("expected code invalid_request or run_error, got %v", resp["code"])
	}
}

func TestHandleRun_SessionResumeNoStore(t *testing.T) {
	srv, _ := setupTestServer(t)
	router := srv.Handler()

	body, _ := json.Marshal(RunRequest{
		Agent:     "test",
		Prompt:    "hello",
		SessionID: "some-id",
	})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/v1/agents/run", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Code)
	}
	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if resp["code"] != "invalid_request" && resp["code"] != "run_error" {
		t.Fatalf("expected code invalid_request or run_error, got %v", resp["code"])
	}
}

func TestHandleRun_ProviderFactoryError(t *testing.T) {
	srv, _ := setupTestServerWithProvider(t, func(a *agent.Agent) (provider.Provider, error) {
		return nil, fmt.Errorf("provider unavailable")
	})
	router := srv.Handler()

	body, _ := json.Marshal(RunRequest{
		Agent:  "test",
		Prompt: "hello",
	})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/v1/agents/run", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", w.Code)
	}
	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if resp["code"] != "provider_error" {
		t.Fatalf("expected code provider_error, got %v", resp["code"])
	}
}

type streamErrorProvider struct{}

func (p *streamErrorProvider) Stream(ctx context.Context, req provider.Request) (<-chan provider.Chunk, error) {
	return nil, fmt.Errorf("stream error: connection reset")
}

func TestHandleRun_ProviderStreamError(t *testing.T) {
	srv, _ := setupTestServerWithProvider(t, func(a *agent.Agent) (provider.Provider, error) {
		return &streamErrorProvider{}, nil
	})
	router := srv.Handler()

	body, _ := json.Marshal(RunRequest{
		Agent:  "test",
		Prompt: "hello",
	})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/v1/agents/run", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	events := parseSSEEvents(w.Body.String())
	var foundError, foundDone bool
	for _, ev := range events {
		if ev["type"] == "error" && ev["code"] == "provider_error" {
			foundError = true
		}
		if ev["type"] == "done" && ev["status"] == "failed" {
			foundDone = true
		}
	}
	if !foundError {
		t.Fatalf("expected provider_error event, got %v", events)
	}
	if !foundDone {
		t.Fatalf("expected done failed event, got %v", events)
	}
}

func TestHandleRun_ToolApprovalRequired(t *testing.T) {
	srv, dir := setupTestServer(t)

	// Create an agent where test_tool requires confirmation.
	agentContent := `name: confirm-agent
version: 1
provider: mock
model: test-model
system_prompt: |
  You are a test assistant.
tools:
  - test_tool
permissions:
  auto_approve: []
`
	if err := os.WriteFile(filepath.Join(dir, "confirm-agent.yaml"), []byte(agentContent), 0o644); err != nil {
		t.Fatal(err)
	}

	// Provider emits a tool call that will require approval.
	args, _ := json.Marshal(map[string]string{})
	mockProviderFactory := func(a *agent.Agent) (provider.Provider, error) {
		return &provider.MockProvider{
			Calls: [][]provider.MockStep{
				{{
					ToolCalls: []provider.ToolCall{{
						ID:        "call_1",
						Name:      "test_tool",
						Arguments: args,
					}},
				}},
			},
		}, nil
	}
	srv.newProvider = mockProviderFactory
	srv.svc.NewProvider = mockProviderFactory

	router := srv.Handler()
	body, _ := json.Marshal(RunRequest{
		Agent:  "confirm-agent",
		Prompt: "call the tool",
	})

	// Resolve the approval in the background so the run can complete.
	go func() {
		time.Sleep(50 * time.Millisecond)
		srv.approver.ResolveApproval("call_1", true)
	}()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/v1/agents/run", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	events := parseSSEEvents(w.Body.String())
	var foundApproval bool
	for _, ev := range events {
		if ev["type"] == "tool_approval_required" {
			foundApproval = true
			if ev["approval_id"] == "" {
				t.Fatal("expected approval_id in approval event")
			}
			if ev["tool_call_id"] == "" {
				t.Fatal("expected tool_call_id in approval event")
			}
		}
	}
	if !foundApproval {
		t.Fatalf("expected tool_approval_required event, got %v", events)
	}
}

func TestHandleRun_ClientDisconnect(t *testing.T) {
	srv, _, store := setupTestServerWithStore(t)
	srv.newProvider = func(a *agent.Agent) (provider.Provider, error) {
		return &delayTextProvider{delay: 200 * time.Millisecond, text: "kept going"}, nil
	}
	srv.svc.NewProvider = srv.newProvider
	router := srv.Handler()

	body, _ := json.Marshal(RunRequest{Agent: "test", Prompt: "hello"})
	ctx, cancel := context.WithCancel(context.Background())
	req, _ := http.NewRequestWithContext(ctx, "POST", "/v1/agents/run", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	done := make(chan struct{})
	go func() {
		router.ServeHTTP(w, req)
		close(done)
	}()

	deadline := time.Now().Add(2 * time.Second)
	var sessionID string
	for time.Now().Before(deadline) && sessionID == "" {
		sessionID = w.Header().Get("X-Session-Id")
		time.Sleep(10 * time.Millisecond)
	}
	if sessionID == "" {
		t.Fatal("expected X-Session-Id before disconnect")
	}
	cancel()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("handler did not return after context cancellation")
	}

	waitSessionText(t, store, sessionID, "kept going")
}

type delayTextProvider struct {
	delay time.Duration
	text  string
}

func (p *delayTextProvider) Stream(ctx context.Context, req provider.Request) (<-chan provider.Chunk, error) {
	ch := make(chan provider.Chunk, 1)
	go func() {
		defer close(ch)
		select {
		case <-ctx.Done():
			return
		case <-time.After(p.delay):
		}
		ch <- provider.Chunk{TextDelta: p.text, Done: true}
	}()
	return ch, nil
}

func waitSessionText(t *testing.T, store session.Store, id, want string) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		sess, err := store.Load(id)
		if err == nil {
			for _, msg := range sess.GetMessages() {
				if msg.Role == provider.RoleAssistant && msg.Content == want {
					return
				}
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("session %s never persisted assistant %q", id, want)
}
