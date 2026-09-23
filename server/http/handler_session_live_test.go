package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/teexue/nexakit/provider"
	kitmock "github.com/teexue/nexakit/provider/mock"
	"github.com/teexue/nexakit/session"

	"github.com/teexue/nexa/core/agent"
)

func TestHandleSessionEvents_FollowAfterDisconnect(t *testing.T) {
	srv, _, _ := setupTestServerWithStore(t)
	srv.newProvider = func(a *agent.Agent) (provider.Provider, error) {
		return &delayTextProvider{delay: 400 * time.Millisecond, text: "hello live"}, nil
	}
	srv.svc.NewProvider = srv.newProvider
	router := srv.Handler()

	sessionID := startDetachedRun(t, router)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/v1/sessions/"+sessionID+"/events", nil)
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	events := parseSSEEvents(w.Body.String())
	require.NotEmpty(t, events)
	require.Equal(t, "snapshot", events[0]["type"])
	var sawText, sawDone bool
	for _, ev := range events {
		if ev["type"] == "text_delta" && ev["content"] == "hello live" {
			sawText = true
		}
		if ev["type"] == "done" && ev["status"] == "completed" {
			sawDone = true
		}
	}
	require.True(t, sawText, "events=%v", events)
	require.True(t, sawDone, "events=%v", events)
}

func TestHandleSessionEvents_NotRunning(t *testing.T) {
	srv, _, store := setupTestServerWithStore(t)
	router := srv.Handler()
	sess := session.New("test")
	require.NoError(t, store.Save(sess))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/v1/sessions/"+sess.ID+"/events", nil)
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandleSessionAbort_CancelsRun(t *testing.T) {
	srv, _, _ := setupTestServerWithStore(t)
	srv.newProvider = func(a *agent.Agent) (provider.Provider, error) {
		return &kitmock.MockProvider{BlockOnStream: true}, nil
	}
	srv.svc.NewProvider = srv.newProvider
	router := srv.Handler()

	sessionID := startDetachedRun(t, router)
	require.True(t, srv.svc.Hub.Running(sessionID))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/v1/sessions/"+sessionID+"/abort", nil)
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if !srv.svc.Hub.Running(sessionID) {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("run still attached after abort")
}

func startDetachedRun(t *testing.T, router http.Handler) string {
	t.Helper()
	body, _ := json.Marshal(RunRequest{Agent: "test", Prompt: "hello"})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/v1/agents/run", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	started := make(chan string, 1)
	go func() {
		// Header is written as soon as the SSE stream opens.
		router.ServeHTTP(w, req)
	}()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if id := w.Header().Get("X-Session-Id"); id != "" {
			started <- id
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	select {
	case id := <-started:
		return id
	default:
		t.Fatalf("run did not start: status=%d body=%s", w.Code, w.Body.String())
		return ""
	}
}
