package service_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/teexue/nexa/core/service"
	"github.com/teexue/nexa/core/store"
)

func newAuthService(t *testing.T) *service.Service {
	t.Helper()
	db, err := store.Open(t.TempDir())
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	return &service.Service{StateDB: db}
}

func TestAuthProbeAndRegister(t *testing.T) {
	svc := newAuthService(t)
	st := svc.AuthProbe()
	assert.False(t, st.HasUsers)
	assert.False(t, st.AllowRegistration)

	first, err := svc.RegisterUser("alice", "secret1", "Alice")
	require.NoError(t, err)
	assert.Equal(t, store.RoleAdmin, first.Role)

	_, err = svc.RegisterUser("bob", "secret2", "Bob")
	require.ErrorIs(t, err, service.ErrRegistrationDisabled)

	allow, err := svc.SetAllowRegistration(true)
	require.NoError(t, err)
	assert.True(t, allow)

	second, err := svc.RegisterUser("bob", "secret2", "Bob")
	require.NoError(t, err)
	assert.Equal(t, store.RoleMember, second.Role)

	st = svc.AuthProbe()
	assert.True(t, st.HasUsers)
	assert.True(t, st.AllowRegistration)
}

func TestAuthenticateUser(t *testing.T) {
	svc := newAuthService(t)
	_, err := svc.RegisterUser("alice", "secret1", "")
	require.NoError(t, err)

	u, err := svc.AuthenticateUser("alice", "secret1")
	require.NoError(t, err)
	assert.Equal(t, "alice", u.Username)

	_, err = svc.AuthenticateUser("alice", "nope")
	require.Error(t, err)
}

func TestAuthOpsRequireState(t *testing.T) {
	svc := &service.Service{}
	_, err := svc.RegisterUser("a", "secret1", "")
	require.ErrorIs(t, err, service.ErrStateNotConfigured)
	_, err = svc.AuthenticateUser("a", "secret1")
	require.ErrorIs(t, err, service.ErrStateNotConfigured)
	_, ok := svc.LookupUser("usr_x")
	assert.False(t, ok)
}
