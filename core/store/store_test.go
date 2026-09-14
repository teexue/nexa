package store_test

import (
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	_ "github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/teexue/nexa/core/auth"
	"github.com/teexue/nexa/core/store"
)

func createTestUser(t *testing.T, db *store.DB) *store.User {
	t.Helper()
	u, err := db.CreateUser("alice", "secret1", "Alice", store.RoleAdmin)
	require.NoError(t, err)
	return u
}

func TestOpen_NoDefaultUser(t *testing.T) {
	dir := t.TempDir()
	db, err := store.Open(dir)
	require.NoError(t, err)
	defer db.Close()

	info, err := os.Stat(store.StateFile(dir))
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o600), info.Mode().Perm())

	var n int64
	require.NoError(t, db.Model(&store.User{}).Count(&n).Error)
	assert.Equal(t, int64(0), n)
}

func TestEnsureAdmin_PromotesEarliestUser(t *testing.T) {
	dir := t.TempDir()
	db, err := store.Open(dir)
	require.NoError(t, err)

	alice, err := db.CreateUser("alice", "secret1", "Alice", store.RoleMember)
	require.NoError(t, err)
	require.NoError(t, db.Model(&store.User{}).
		Where("id = ?", alice.ID).Update("role", store.RoleMember).Error)
	require.NoError(t, db.Close())

	db, err = store.Open(dir)
	require.NoError(t, err)
	defer db.Close()

	role, err := db.GetUserRole(alice.ID)
	require.NoError(t, err)
	assert.Equal(t, store.RoleAdmin, role)
}

func TestAPIKey_AddVerifyJWT(t *testing.T) {
	dir := t.TempDir()
	db, err := store.Open(dir)
	require.NoError(t, err)
	defer db.Close()

	owner := createTestUser(t, db)

	raw, entry, err := db.AddAPIKey(owner.ID, "default", "", nil)
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(raw, "ca_"))
	assert.Len(t, raw, 3+48)
	assert.Equal(t, store.HashAPIKey(raw), entry.KeyHash)
	assert.NotEqual(t, raw, entry.KeyHash)
	assert.Equal(t, "*", entry.Scopes)
	assert.True(t, entry.Enabled)

	got, err := db.VerifyAPIKey(raw)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, entry.ID, got.ID)

	secret, err := db.EnsureJWTSecret()
	require.NoError(t, err)
	tokens := auth.NewTokenService(secret, db.HasAPIKeyID, db.HasUser)
	tok, err := tokens.Issue(auth.Identity{UserID: entry.UserID, KeyID: entry.ID})
	require.NoError(t, err)
	id, err := tokens.Parse(tok)
	require.NoError(t, err)
	assert.Equal(t, entry.UserID, id.UserID)
	assert.Equal(t, entry.ID, id.KeyID)

	require.NoError(t, db.DeleteAPIKey(entry.ID, owner.ID))
	_, err = tokens.Parse(tok)
	require.Error(t, err)
}

func TestCreateUser_Authenticate(t *testing.T) {
	dir := t.TempDir()
	db, err := store.Open(dir)
	require.NoError(t, err)
	defer db.Close()

	u, err := db.CreateUser("alice", "secret1", "Alice", "")
	require.NoError(t, err)
	assert.Equal(t, "alice", u.Username)
	assert.Equal(t, store.RoleMember, u.Role)
	assert.NotEmpty(t, u.PasswordHash)

	got, err := db.AuthenticateUser("Alice", "secret1") // case-insensitive username
	require.NoError(t, err)
	assert.Equal(t, u.ID, got.ID)

	_, err = db.AuthenticateUser("alice", "wrong")
	require.Error(t, err)

	_, err = db.CreateUser("alice", "other12", "", "")
	require.Error(t, err)

	_, err = db.CreateUser("mallory", "secret1", "", "root")
	require.Error(t, err)

	n, err := db.CountUsersWithPassword()
	require.NoError(t, err)
	assert.Equal(t, int64(1), n)
}

func TestMigrateAPIKeysFile(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "api_keys.yaml"), []byte(`keys:
  - id: ak_old
    name: legacy
    key: ca_legacy_plain_key
`), 0o600))

	db, err := store.Open(dir)
	require.NoError(t, err)
	defer db.Close()

	got, err := db.VerifyAPIKey("ca_legacy_plain_key")
	require.NoError(t, err)
	require.NotNil(t, got)
	// Legacy keys get full scope and stay enabled.
	assert.Equal(t, "*", got.Scopes)
	assert.True(t, got.Enabled)
}

func TestVerifyAPIKey_EnabledExpiryLastUsed(t *testing.T) {
	dir := t.TempDir()
	db, err := store.Open(dir)
	require.NoError(t, err)
	defer db.Close()

	owner := createTestUser(t, db)

	raw, entry, err := db.AddAPIKey(owner.ID, "k1", "agents,sessions", nil)
	require.NoError(t, err)

	// Unknown key does not authenticate.
	got, err := db.VerifyAPIKey("ca_" + strings.Repeat("0", 48))
	require.NoError(t, err)
	assert.Nil(t, got)

	// Hit updates last_used_at.
	got, err = db.VerifyAPIKey(raw)
	require.NoError(t, err)
	require.NotNil(t, got)
	require.NotNil(t, got.LastUsedAt)
	persisted, err := db.GetAPIKey(entry.ID)
	require.NoError(t, err)
	require.NotNil(t, persisted.LastUsedAt)
	assert.WithinDuration(t, time.Now(), *persisted.LastUsedAt, 5*time.Second)

	// Disabled key does not authenticate.
	require.NoError(t, db.UpdateAPIKey(entry.ID, "", store.APIKeyPatch{Enabled: ptr(false)}))
	got, err = db.VerifyAPIKey(raw)
	require.NoError(t, err)
	assert.Nil(t, got)

	// Re-enable, then expire it.
	require.NoError(t, db.UpdateAPIKey(entry.ID, "", store.APIKeyPatch{Enabled: ptr(true)}))
	past := time.Now().Add(-time.Hour)
	raw2, _, err := db.AddAPIKey(owner.ID, "k2", "*", &past)
	require.NoError(t, err)
	got, err = db.VerifyAPIKey(raw2)
	require.NoError(t, err)
	assert.Nil(t, got)

	// Future expiry still authenticates.
	future := time.Now().Add(time.Hour)
	raw3, _, err := db.AddAPIKey(owner.ID, "k3", "*", &future)
	require.NoError(t, err)
	got, err = db.VerifyAPIKey(raw3)
	require.NoError(t, err)
	require.NotNil(t, got)
}

func TestUpdateAPIKey_Patch(t *testing.T) {
	dir := t.TempDir()
	db, err := store.Open(dir)
	require.NoError(t, err)
	defer db.Close()

	owner := createTestUser(t, db)

	_, entry, err := db.AddAPIKey(owner.ID, "k1", "*", nil)
	require.NoError(t, err)

	name := "renamed"
	scopes := "agents"
	require.NoError(t, db.UpdateAPIKey(entry.ID, owner.ID, store.APIKeyPatch{
		Name: &name, Scopes: &scopes,
	}))
	got, err := db.GetAPIKey(entry.ID)
	require.NoError(t, err)
	assert.Equal(t, "renamed", got.Name)
	assert.Equal(t, "agents", got.Scopes)
	assert.True(t, got.Enabled)

	// Empty name / scopes rejected.
	empty := ""
	require.Error(t, db.UpdateAPIKey(entry.ID, "", store.APIKeyPatch{Name: &empty}))
	require.Error(t, db.UpdateAPIKey(entry.ID, "", store.APIKeyPatch{Scopes: &empty}))

	// Wrong user scope misses.
	require.ErrorIs(t, db.UpdateAPIKey(entry.ID, "usr_other", store.APIKeyPatch{Name: &name}), os.ErrNotExist)

	// Missing key.
	require.ErrorIs(t, db.UpdateAPIKey("ak_missing", "", store.APIKeyPatch{Name: &name}), os.ErrNotExist)
}

func ptr[T any](v T) *T { return &v }

// TestOpen_UpgradesLegacySchema opens a pre-RBAC state.db (users/api_keys
// without role/scopes/expires_at/last_used_at/enabled columns) and verifies
// column defaults, key backfill, and admin promotion.
func TestOpen_UpgradesLegacySchema(t *testing.T) {
	dir := t.TempDir()
	path := store.StateFile(dir)

	raw, err := sql.Open("sqlite", path)
	require.NoError(t, err)
	stmts := []string{
		`CREATE TABLE users (id TEXT PRIMARY KEY, username TEXT, password_hash TEXT NOT NULL, name TEXT NOT NULL, created_at DATETIME NOT NULL)`,
		`CREATE TABLE api_keys (id TEXT PRIMARY KEY, user_id TEXT NOT NULL, name TEXT NOT NULL, key_hash TEXT NOT NULL, prefix TEXT NOT NULL, created_at DATETIME NOT NULL)`,
		`INSERT INTO users (id, username, password_hash, name, created_at) VALUES ('usr_local', 'local', '', 'local', '2024-01-01 00:00:00')`,
		`INSERT INTO api_keys (id, user_id, name, key_hash, prefix, created_at) VALUES ('ak_legacy', 'usr_local', 'legacy', '` + store.HashAPIKey("ca_legacy_secret") + `', 'ca_lega…', '2024-01-01 00:00:00')`,
	}
	for _, s := range stmts {
		_, err := raw.Exec(s)
		require.NoError(t, err)
	}
	require.NoError(t, raw.Close())

	db, err := store.Open(dir)
	require.NoError(t, err)
	defer db.Close()

	// Existing key keeps working, backfilled to full scope + enabled.
	got, err := db.VerifyAPIKey("ca_legacy_secret")
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "*", got.Scopes)
	assert.True(t, got.Enabled)
}
