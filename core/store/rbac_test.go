package store_test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/teexue/nexa/core/store"
)

func TestListUsers(t *testing.T) {
	db := openTestDB(t)

	alice, err := db.CreateUser("alice", "secret1", "Alice", store.RoleMember)
	require.NoError(t, err)
	bob, err := db.CreateUser("bob", "secret2", "", store.RoleAdmin)
	require.NoError(t, err)

	users, err := db.ListUsers()
	require.NoError(t, err)
	require.Len(t, users, 2)

	byID := map[string]store.User{}
	for _, u := range users {
		byID[u.ID] = u
	}
	assert.Equal(t, store.RoleMember, byID[alice.ID].Role)
	assert.Equal(t, store.RoleAdmin, byID[bob.ID].Role)
	assert.Equal(t, "bob", byID[bob.ID].Name)
}

func TestUpdateUserRole_LastAdminProtection(t *testing.T) {
	db := openTestDB(t)
	alice, err := db.CreateUser("alice", "secret1", "", store.RoleAdmin)
	require.NoError(t, err)

	require.Error(t, db.UpdateUserRole(alice.ID, store.RoleMember))

	role, err := db.GetUserRole(alice.ID)
	require.NoError(t, err)
	assert.Equal(t, store.RoleAdmin, role)

	bob, err := db.CreateUser("bob", "secret2", "", store.RoleAdmin)
	require.NoError(t, err)
	require.NoError(t, db.UpdateUserRole(alice.ID, store.RoleMember))

	role, err = db.GetUserRole(alice.ID)
	require.NoError(t, err)
	assert.Equal(t, store.RoleMember, role)

	require.NoError(t, db.UpdateUserRole(alice.ID, store.RoleAdmin))
	require.NoError(t, db.UpdateUserRole(bob.ID, store.RoleMember))

	require.Error(t, db.UpdateUserRole(alice.ID, "root"))
	require.ErrorIs(t, db.UpdateUserRole("usr_missing", store.RoleAdmin), os.ErrNotExist)
}

func TestDeleteUser_LastAdminProtection(t *testing.T) {
	db := openTestDB(t)
	alice, err := db.CreateUser("alice", "secret1", "", store.RoleAdmin)
	require.NoError(t, err)

	require.Error(t, db.DeleteUser(alice.ID))

	bob, err := db.CreateUser("bob", "secret2", "", store.RoleAdmin)
	require.NoError(t, err)

	require.NoError(t, db.DeleteUser(alice.ID))
	assert.False(t, db.HasUser(alice.ID))

	require.Error(t, db.DeleteUser(bob.ID))

	carol, err := db.CreateUser("carol", "secret3", "", store.RoleMember)
	require.NoError(t, err)
	_, key, err := db.AddAPIKey(carol.ID, "b1", "*", nil)
	require.NoError(t, err)
	require.NoError(t, db.DeleteUser(carol.ID))
	assert.False(t, db.HasUser(carol.ID))
	assert.False(t, db.HasAPIKeyID(key.ID))

	require.ErrorIs(t, db.DeleteUser("usr_missing"), os.ErrNotExist)
}

func TestResetUserPassword(t *testing.T) {
	db := openTestDB(t)

	alice, err := db.CreateUser("alice", "secret1", "", store.RoleMember)
	require.NoError(t, err)

	require.Error(t, db.ResetUserPassword(alice.ID, "short"))
	require.NoError(t, db.ResetUserPassword(alice.ID, "newsecret"))

	_, err = db.AuthenticateUser("alice", "secret1")
	require.Error(t, err)
	got, err := db.AuthenticateUser("alice", "newsecret")
	require.NoError(t, err)
	assert.Equal(t, alice.ID, got.ID)

	require.ErrorIs(t, db.ResetUserPassword("usr_missing", "newsecret"), os.ErrNotExist)
}

func TestGetUserRole(t *testing.T) {
	db := openTestDB(t)
	alice, err := db.CreateUser("alice", "secret1", "", store.RoleAdmin)
	require.NoError(t, err)

	role, err := db.GetUserRole(alice.ID)
	require.NoError(t, err)
	assert.Equal(t, store.RoleAdmin, role)

	_, err = db.GetUserRole("usr_missing")
	require.ErrorIs(t, err, os.ErrNotExist)
}

func TestAllowRegistration(t *testing.T) {
	db := openTestDB(t)

	assert.False(t, db.GetAllowRegistration())

	require.NoError(t, db.SetAllowRegistration(true))
	assert.True(t, db.GetAllowRegistration())

	require.NoError(t, db.SetAllowRegistration(false))
	assert.False(t, db.GetAllowRegistration())
}
