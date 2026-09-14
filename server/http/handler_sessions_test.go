package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/teexue/nexakit/session"
)

func TestHandleSessionsList(t *testing.T) {
	srv, _, store := setupTestServerWithStore(t)
	router := srv.Handler()

	sess := session.New("test")
	if err := store.Save(sess); err != nil {
		t.Fatal(err)
	}

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/v1/sessions", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var metas []session.SessionMeta
	if err := json.Unmarshal(w.Body.Bytes(), &metas); err != nil {
		t.Fatalf("failed to unmarshal sessions: %v", err)
	}
	if len(metas) != 1 {
		t.Fatalf("expected 1 session, got %d", len(metas))
	}
	if metas[0].ID != sess.ID {
		t.Fatalf("expected session id %q, got %q", sess.ID, metas[0].ID)
	}
}

func TestHandleSessionsGet(t *testing.T) {
	srv, _, store := setupTestServerWithStore(t)
	router := srv.Handler()

	sess := session.New("test")
	if err := store.Save(sess); err != nil {
		t.Fatal(err)
	}

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/v1/sessions/"+sess.ID, nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if resp["id"] != sess.ID {
		t.Fatalf("expected id %q, got %v", sess.ID, resp["id"])
	}
}

func TestHandleSessionsGet_NotFound(t *testing.T) {
	srv, _, _ := setupTestServerWithStore(t)
	router := srv.Handler()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/v1/sessions/does-not-exist", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", w.Code)
	}
}

func TestHandleSessionsDelete(t *testing.T) {
	srv, _, store := setupTestServerWithStore(t)
	router := srv.Handler()

	sess := session.New("test")
	if err := store.Save(sess); err != nil {
		t.Fatal(err)
	}

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", "/v1/sessions/"+sess.ID, nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	_, err := store.Load(sess.ID)
	if err == nil {
		t.Fatal("expected session to be deleted")
	}
}

func TestHandleSessionsDelete_NotFound(t *testing.T) {
	srv, _, _ := setupTestServerWithStore(t)
	router := srv.Handler()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", "/v1/sessions/does-not-exist", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", w.Code)
	}
}
