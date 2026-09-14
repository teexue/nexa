package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/teexue/nexa/core/config"
	"github.com/teexue/nexa/core/store"
)

func setupAuthServer(t *testing.T) (*Server, *store.DB) {
	t.Helper()
	srv, dir := setupTestServer(t)
	db, err := store.Open(dir)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = db.Close()
		config.BindDB(nil)
	})
	require.NoError(t, srv.SetStateDB(db))
	return srv, db
}

func registerAndToken(t *testing.T, router http.Handler, username, password string) string {
	t.Helper()
	body, _ := json.Marshal(map[string]string{
		"username": username, "password": password, "name": username,
	})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/v1/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
	var session struct {
		Token string `json:"token"`
	}
	require.NoError(t, json.NewDecoder(w.Body).Decode(&session))
	require.NotEmpty(t, session.Token)
	return session.Token
}

type authReq struct {
	method string
	path   string
	token  string
	apiKey string
	body   any
}

func doAuthReq(t *testing.T, router http.Handler, r authReq) *httptest.ResponseRecorder {
	t.Helper()
	var reader *bytes.Reader
	if r.body != nil {
		b, err := json.Marshal(r.body)
		require.NoError(t, err)
		reader = bytes.NewReader(b)
	} else {
		reader = bytes.NewReader(nil)
	}
	w := httptest.NewRecorder()
	req, err := http.NewRequest(r.method, r.path, reader)
	require.NoError(t, err)
	if r.body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if r.token != "" {
		req.Header.Set("Authorization", "Bearer "+r.token)
	}
	if r.apiKey != "" {
		req.Header.Set("X-API-Key", r.apiKey)
	}
	router.ServeHTTP(w, req)
	return w
}

func createTestAPIKey(t *testing.T, router http.Handler, token, name string, scopes []string) createAPIKeyResponse {
	t.Helper()
	body := map[string]any{"name": name}
	if scopes != nil {
		body["scopes"] = scopes
	}
	w := doAuthReq(t, router, authReq{method: "POST", path: "/v1/auth/keys", token: token, body: body})
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
	var raw map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &raw))
	_, hasToken := raw["token"]
	assert.False(t, hasToken, "keys create must not return a JWT")
	var created createAPIKeyResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &created))
	require.NotEmpty(t, created.ID)
	return created
}

func patchAPIKeyEnabled(t *testing.T, router http.Handler, token, id string, enabled bool) {
	t.Helper()
	w := doAuthReq(t, router, authReq{
		method: "PATCH", path: "/v1/auth/keys/" + id, token: token,
		body: map[string]any{"enabled": enabled},
	})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
}

func registerUserJSON(t *testing.T, router http.Handler, username, password, name string) *httptest.ResponseRecorder {
	t.Helper()
	body := map[string]string{"username": username, "password": password}
	if name != "" {
		body["name"] = name
	}
	return doAuthReq(t, router, authReq{method: "POST", path: "/v1/auth/register", body: body})
}

func TestAuth_MultipleAPIKeys(t *testing.T) {
	srv, _ := setupAuthServer(t)
	srv.SetAPIKeys([]string{"key-a", "key-b"})
	router := srv.Handler()

	for _, key := range []string{"key-a", "key-b"} {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/v1/tools", nil)
		req.Header.Set("X-API-Key", key)
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code, "key %s", key)
	}

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/v1/tools", nil)
	req.Header.Set("X-API-Key", "key-c")
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuth_EventsQueryParam(t *testing.T) {
	srv, _ := setupAuthServer(t)
	srv.SetAPIKey("sse-secret")
	router := srv.Handler()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/v1/events", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)

	w = httptest.NewRecorder()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	req, _ = http.NewRequestWithContext(ctx, "GET", "/v1/events?api_key=sse-secret", nil)
	router.ServeHTTP(w, req)
	assert.NotEqual(t, http.StatusUnauthorized, w.Code)
}

func TestAuth_StatusPublicAndDataBlocked(t *testing.T) {
	srv, _ := setupAuthServer(t)
	router := srv.Handler()

	// Status is public.
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/v1/auth/status", nil)
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	var status struct {
		AuthRequired bool `json:"auth_required"`
		HasUsers     bool `json:"has_users"`
	}
	require.NoError(t, json.NewDecoder(w.Body).Decode(&status))
	assert.True(t, status.AuthRequired)
	assert.False(t, status.HasUsers)

	// Data endpoints blocked without credentials.
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/v1/tools", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)

	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/v1/agents", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuthKeys_CRUD_JWT(t *testing.T) {
	srv, _ := setupAuthServer(t)
	router := srv.Handler()
	token := registerAndToken(t, router, "alice", "secret1")

	w := doAuthReq(t, router, authReq{method: "GET", path: "/v1/auth/keys"})
	assert.Equal(t, http.StatusUnauthorized, w.Code)

	w = doAuthReq(t, router, authReq{method: "GET", path: "/v1/auth/keys", token: token})
	require.Equal(t, http.StatusOK, w.Code)
	var listResp struct {
		Enabled bool               `json:"enabled"`
		Keys    []store.APIKeyInfo `json:"keys"`
	}
	require.NoError(t, json.NewDecoder(w.Body).Decode(&listResp))
	assert.True(t, listResp.Enabled)
	assert.Empty(t, listResp.Keys)

	created := createTestAPIKey(t, router, token, "ui", []string{"agents"})
	assert.True(t, strings.HasPrefix(created.Key, "ca_"))
	assert.Equal(t, []string{"agents"}, created.Scopes)

	w = doAuthReq(t, router, authReq{method: "GET", path: "/v1/tools", apiKey: created.Key})
	assert.Equal(t, http.StatusOK, w.Code)
	w = doAuthReq(t, router, authReq{method: "GET", path: "/v1/fs/list", apiKey: created.Key})
	assert.Equal(t, http.StatusForbidden, w.Code)

	_ = createTestAPIKey(t, router, token, "ui2", nil)
	patchAPIKeyEnabled(t, router, token, created.ID, false)
	w = doAuthReq(t, router, authReq{method: "GET", path: "/v1/tools", apiKey: created.Key})
	assert.Equal(t, http.StatusUnauthorized, w.Code)

	patchAPIKeyEnabled(t, router, token, created.ID, true)
	w = doAuthReq(t, router, authReq{method: "DELETE", path: "/v1/auth/keys/" + created.ID, token: token})
	require.Equal(t, http.StatusOK, w.Code)
	w = doAuthReq(t, router, authReq{method: "GET", path: "/v1/tools", apiKey: created.Key})
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuth_RegisterLoginMultiUser(t *testing.T) {
	srv, db := setupAuthServer(t)
	router := srv.Handler()

	w := registerUserJSON(t, router, "alice", "secret1", "Alice")
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
	var aliceSession struct {
		Token  string `json:"token"`
		UserID string `json:"user_id"`
		User   struct {
			Username string `json:"username"`
		} `json:"user"`
	}
	require.NoError(t, json.NewDecoder(w.Body).Decode(&aliceSession))
	assert.Equal(t, "alice", aliceSession.User.Username)
	assert.NotEmpty(t, aliceSession.Token)

	w = doAuthReq(t, router, authReq{method: "GET", path: "/v1/tools"})
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	w = doAuthReq(t, router, authReq{method: "GET", path: "/v1/tools", token: aliceSession.Token})
	assert.Equal(t, http.StatusOK, w.Code)

	w = registerUserJSON(t, router, "bob", "secret2", "")
	require.Equal(t, http.StatusForbidden, w.Code)
	require.NoError(t, db.SetAllowRegistration(true))
	w = registerUserJSON(t, router, "bob", "secret2", "")
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
	var bobSession struct {
		Token  string `json:"token"`
		UserID string `json:"user_id"`
	}
	require.NoError(t, json.NewDecoder(w.Body).Decode(&bobSession))
	assert.NotEqual(t, aliceSession.UserID, bobSession.UserID)

	keyBody := map[string]any{"name": "a1"}
	w = doAuthReq(t, router, authReq{method: "POST", path: "/v1/auth/keys", token: bobSession.Token, body: keyBody})
	assert.Equal(t, http.StatusForbidden, w.Code)
	w = doAuthReq(t, router, authReq{method: "GET", path: "/v1/auth/keys", token: bobSession.Token})
	assert.Equal(t, http.StatusForbidden, w.Code)

	w = doAuthReq(t, router, authReq{method: "POST", path: "/v1/auth/keys", token: aliceSession.Token, body: keyBody})
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
	w = doAuthReq(t, router, authReq{method: "GET", path: "/v1/auth/keys", token: aliceSession.Token})
	require.Equal(t, http.StatusOK, w.Code)
	var aliceKeys struct {
		Keys []store.APIKeyInfo `json:"keys"`
	}
	require.NoError(t, json.NewDecoder(w.Body).Decode(&aliceKeys))
	assert.Len(t, aliceKeys.Keys, 1)

	w = doAuthReq(t, router, authReq{
		method: "POST", path: "/v1/auth/login",
		body: map[string]string{"username": "alice", "password": "secret1"},
	})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	n, err := db.CountUsersWithPassword()
	require.NoError(t, err)
	assert.Equal(t, int64(2), n)
}

func TestAuth_NoAPIKey_AllowsAll(t *testing.T) {
	srv, _ := setupTestServer(t)
	router := srv.Handler()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/v1/tools", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 without auth, got %d", w.Code)
	}
}

func TestAuth_WithAPIKey_RequiresKey(t *testing.T) {
	srv, _ := setupTestServer(t)
	srv.SetAPIKey("secret-key-123")
	router := srv.Handler()

	// No key → 401.
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/v1/tools", nil)
	router.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without key, got %d", w.Code)
	}

	// Wrong key → 401.
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/v1/tools", nil)
	req.Header.Set("Authorization", "Bearer wrong-key")
	router.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 with wrong key, got %d", w.Code)
	}

	// Correct key via Bearer → 200.
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/v1/tools", nil)
	req.Header.Set("Authorization", "Bearer secret-key-123")
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 with correct Bearer key, got %d", w.Code)
	}

	// Correct key via X-API-Key → 200.
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/v1/tools", nil)
	req.Header.Set("X-API-Key", "secret-key-123")
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 with correct X-API-Key, got %d", w.Code)
	}
}

func TestAuth_WithAPIKey_CaseInsensitiveBearer(t *testing.T) {
	srv, _ := setupTestServer(t)
	srv.SetAPIKey("secret-key-123")
	router := srv.Handler()

	// Lowercase "bearer" should also be accepted.
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/v1/tools", nil)
	req.Header.Set("Authorization", "bearer secret-key-123")
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 with lowercase bearer, got %d", w.Code)
	}
}

func TestAuth_HealthEndpointsAlwaysPublic(t *testing.T) {
	srv, _ := setupTestServer(t)
	srv.SetAPIKey("secret-key-123")
	router := srv.Handler()

	for _, path := range []string{"/healthz", "/readyz"} {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", path, nil)
		router.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200 for %s with auth enabled, got %d", path, w.Code)
		}
	}
}

func TestAuth_RunEndpointWithKey(t *testing.T) {
	srv, _ := setupTestServer(t)
	srv.SetAPIKey("test-api-key")
	router := srv.Handler()

	body, _ := json.Marshal(RunRequest{
		Agent:  "test",
		Prompt: "hello",
	})

	// Without key → 401.
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/v1/agents/run", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without key, got %d", w.Code)
	}

	// With correct key → 200.
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("POST", "/v1/agents/run", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer test-api-key")
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 with correct key, got %d: %s", w.Code, w.Body.String())
	}
}
