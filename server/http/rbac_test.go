package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/teexue/nexa/core/config"
	"github.com/teexue/nexa/core/store"
)

// setupRBACServer builds a server with both a session store and a state DB
// so scoped routes (sessions, kanban) are registered.
func setupRBACServer(t *testing.T) (*Server, *store.DB) {
	t.Helper()
	srv, dir, _ := setupTestServerWithStore(t)
	db, err := store.Open(dir)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = db.Close()
		config.BindDB(nil)
	})
	require.NoError(t, srv.SetStateDB(db))
	return srv, db
}

// doJSON performs an authenticated JSON request and returns the recorder.
func doJSON(t *testing.T, router http.Handler, method, path, token string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		require.NoError(t, json.NewEncoder(&buf).Encode(body))
	}
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	router.ServeHTTP(w, req)
	return w
}

// createKeyViaAPI creates an API key through the admin endpoint.
func createKeyViaAPI(t *testing.T, router http.Handler, adminToken, name string, scopes []string) createAPIKeyResponse {
	t.Helper()
	w := doJSON(t, router, "POST", "/v1/auth/keys", adminToken, map[string]any{
		"name": name, "scopes": scopes,
	})
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
	var created createAPIKeyResponse
	require.NoError(t, json.NewDecoder(w.Body).Decode(&created))
	require.NotEmpty(t, created.Key)
	return created
}

func TestRBAC_FirstUserAdminAndRegistrationGate(t *testing.T) {
	srv, _ := setupRBACServer(t)
	router := srv.Handler()

	// First user becomes admin.
	w := doJSON(t, router, "POST", "/v1/auth/register", "", map[string]string{
		"username": "alice", "password": "secret1",
	})
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
	var reg struct {
		Token string `json:"token"`
		User  struct {
			Role string `json:"role"`
		} `json:"user"`
	}
	require.NoError(t, json.NewDecoder(w.Body).Decode(&reg))
	assert.Equal(t, store.RoleAdmin, reg.User.Role)

	// /auth/me exposes the role.
	w = doJSON(t, router, "GET", "/v1/auth/me", reg.Token, nil)
	require.Equal(t, http.StatusOK, w.Code)
	var me struct {
		Role string `json:"role"`
	}
	require.NoError(t, json.NewDecoder(w.Body).Decode(&me))
	assert.Equal(t, store.RoleAdmin, me.Role)

	// Registration is closed by default for subsequent users.
	w = doJSON(t, router, "POST", "/v1/auth/register", "", map[string]string{
		"username": "bob", "password": "secret2",
	})
	require.Equal(t, http.StatusForbidden, w.Code)
	var errBody struct {
		Code string `json:"code"`
	}
	require.NoError(t, json.NewDecoder(w.Body).Decode(&errBody))
	assert.Equal(t, "registration_disabled", errBody.Code)

	// Status reports the flag.
	w = doJSON(t, router, "GET", "/v1/auth/status", "", nil)
	require.Equal(t, http.StatusOK, w.Code)
	var status struct {
		AllowRegistration bool `json:"allow_registration"`
	}
	require.NoError(t, json.NewDecoder(w.Body).Decode(&status))
	assert.False(t, status.AllowRegistration)

	// Admin opens registration; bob joins as member.
	w = doJSON(t, router, "PUT", "/v1/admin/settings/registration", reg.Token, map[string]bool{
		"allow_registration": true,
	})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	w = doJSON(t, router, "POST", "/v1/auth/register", "", map[string]string{
		"username": "bob", "password": "secret2",
	})
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
	require.NoError(t, json.NewDecoder(w.Body).Decode(&reg))
	assert.Equal(t, store.RoleMember, reg.User.Role)

	w = doJSON(t, router, "GET", "/v1/auth/status", "", nil)
	require.NoError(t, json.NewDecoder(w.Body).Decode(&status))
	assert.True(t, status.AllowRegistration)
}

func TestRBAC_MemberForbiddenAdminRoutes(t *testing.T) {
	srv, db := setupRBACServer(t)
	router := srv.Handler()
	adminToken := registerAndToken(t, router, "alice", "secret1")
	require.NoError(t, db.SetAllowRegistration(true))
	memberToken := registerAndToken(t, router, "bob", "secret2")

	// Member cannot touch admin routes.
	for _, tc := range []struct{ method, path string }{
		{"GET", "/v1/admin/users"},
		{"POST", "/v1/admin/users"},
		{"PUT", "/v1/admin/settings/registration"},
		{"POST", "/v1/providers"},
		{"DELETE", "/v1/providers/openai"},
		{"PUT", "/v1/embedding"},
		{"PUT", "/v1/subagent"},
		{"PUT", "/v1/shell"},
		{"GET", "/v1/auth/keys"},
	} {
		w := doJSON(t, router, tc.method, tc.path, memberToken, map[string]any{})
		assert.Equal(t, http.StatusForbidden, w.Code, "%s %s", tc.method, tc.path)
	}

	// Member still passes scope checks on data routes.
	w := doJSON(t, router, "GET", "/v1/sessions", memberToken, nil)
	assert.Equal(t, http.StatusOK, w.Code)

	// Admin can list users.
	w = doJSON(t, router, "GET", "/v1/admin/users", adminToken, nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var list struct {
		Users []store.UserInfo `json:"users"`
	}
	require.NoError(t, json.NewDecoder(w.Body).Decode(&list))
	assert.Len(t, list.Users, 2) // alice + bob
}

func TestRBAC_ScopedAPIKey(t *testing.T) {
	srv, _ := setupRBACServer(t)
	router := srv.Handler()
	adminToken := registerAndToken(t, router, "alice", "secret1")

	// Key limited to the sessions scope.
	key := createKeyViaAPI(t, router, adminToken, "sessions-only", []string{"sessions"})
	assert.Equal(t, []string{"sessions"}, key.Scopes)

	req, _ := http.NewRequest("GET", "/v1/sessions", nil)
	req.Header.Set("X-API-Key", key.Key)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code, w.Body.String())

	for _, path := range []string{"/v1/tools", "/v1/kanban", "/v1/fs/list", "/v1/skills"} {
		req, _ := http.NewRequest("GET", path, nil)
		req.Header.Set("X-API-Key", key.Key)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusForbidden, w.Code, "GET %s", path)
	}
	req, _ = http.NewRequest("POST", "/v1/fs/mkdir", nil)
	req.Header.Set("X-API-Key", key.Key)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusForbidden, w.Code, "POST /v1/fs/mkdir")

	// Key-exchanged JWT keeps the same scope restriction.
	w = doJSON(t, router, "POST", "/v1/auth/token", "", map[string]string{"api_key": key.Key})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var tok struct {
		Token string `json:"token"`
	}
	require.NoError(t, json.NewDecoder(w.Body).Decode(&tok))
	require.NotEmpty(t, tok.Token)

	w = doJSON(t, router, "GET", "/v1/sessions", tok.Token, nil)
	assert.Equal(t, http.StatusOK, w.Code)
	w = doJSON(t, router, "GET", "/v1/tools", tok.Token, nil)
	assert.Equal(t, http.StatusForbidden, w.Code)

	// Default key (no scopes) gets "*" and passes everywhere.
	full := createKeyViaAPI(t, router, adminToken, "full", nil)
	assert.Equal(t, []string{"*"}, full.Scopes)
	req, _ = http.NewRequest("GET", "/v1/tools", nil)
	req.Header.Set("X-API-Key", full.Key)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRBAC_AdminUserManagement(t *testing.T) {
	srv, db := setupRBACServer(t)
	router := srv.Handler()
	adminToken := registerAndToken(t, router, "alice", "secret1")

	// Create a member.
	w := doJSON(t, router, "POST", "/v1/admin/users", adminToken, map[string]string{
		"username": "carol", "password": "secret3", "name": "Carol", "role": "member",
	})
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
	var created store.UserInfo
	require.NoError(t, json.NewDecoder(w.Body).Decode(&created))
	assert.Equal(t, "carol", created.Username)
	assert.Equal(t, store.RoleMember, created.Role)

	// Promote to admin and reset password.
	w = doJSON(t, router, "PATCH", "/v1/admin/users/"+created.ID, adminToken, map[string]string{
		"role": "admin", "password": "newsecret1",
	})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.NoError(t, json.NewDecoder(w.Body).Decode(&created))
	assert.Equal(t, store.RoleAdmin, created.Role)

	// New password works for login.
	w = doJSON(t, router, "POST", "/v1/auth/login", "", map[string]string{
		"username": "carol", "password": "newsecret1",
	})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	// Delete the user.
	w = doJSON(t, router, "DELETE", "/v1/admin/users/"+created.ID, adminToken, nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.False(t, db.HasUser(created.ID))

	// Deleting a missing user is 404.
	w = doJSON(t, router, "DELETE", "/v1/admin/users/"+created.ID, adminToken, nil)
	assert.Equal(t, http.StatusNotFound, w.Code)
}
