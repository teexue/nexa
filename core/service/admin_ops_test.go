package service_test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/teexue/nexa/core/service"
	"github.com/teexue/nexa/core/store"
)

func TestAdminUsersAndRegistration(t *testing.T) {
	svc := newAuthService(t)
	admin, err := svc.RegisterUser("alice", "secret1", "Alice")
	require.NoError(t, err)

	_, err = svc.SetAllowRegistration(true)
	require.NoError(t, err)
	_, err = svc.RegisterUser("bob", "secret2", "Bob")
	require.NoError(t, err)

	users, err := svc.ListUsers()
	require.NoError(t, err)
	require.Len(t, users, 2) // alice + bob

	carol, err := svc.AdminCreateUser("carol", "secret3", "Carol", store.RoleMember)
	require.NoError(t, err)
	assert.Equal(t, store.RoleMember, carol.Role)

	patched, err := svc.AdminPatchUser(carol.ID, store.RoleAdmin, "")
	require.NoError(t, err)
	assert.Equal(t, store.RoleAdmin, patched.Role)

	require.NoError(t, svc.AdminDeleteUser(carol.ID))
	err = svc.AdminDeleteUser(carol.ID)
	require.ErrorIs(t, err, os.ErrNotExist)

	_, err = svc.AdminPatchUser(admin.ID, "root", "")
	require.Error(t, err)
}

func TestAdminOpsRequireState(t *testing.T) {
	svc := &service.Service{}
	_, err := svc.ListUsers()
	require.ErrorIs(t, err, service.ErrStateNotConfigured)
	_, err = svc.AllowRegistration()
	require.ErrorIs(t, err, service.ErrStateNotConfigured)
}
