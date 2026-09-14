package service_test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/teexue/nexa/core/service"
	"github.com/teexue/nexa/core/store"
)

func TestCreateListPatchDeleteAPIKey(t *testing.T) {
	svc := newAuthService(t)
	u, err := svc.RegisterUser("alice", "secret1", "")
	require.NoError(t, err)

	created, err := svc.CreateAPIKey(service.CreateAPIKeyRequest{
		UserID: u.ID, Name: "cli", Scopes: []string{"sessions"},
	})
	require.NoError(t, err)
	assert.NotEmpty(t, created.Raw)
	assert.Equal(t, u.ID, created.Key.UserID)

	keys, err := svc.ListAPIKeys(u.ID)
	require.NoError(t, err)
	require.Len(t, keys, 1)
	assert.Equal(t, "cli", keys[0].Name)

	enabled := false
	info, err := svc.PatchAPIKey(created.Key.ID, u.ID, store.APIKeyPatch{Enabled: &enabled})
	require.NoError(t, err)
	assert.False(t, info.Enabled)

	require.NoError(t, svc.DeleteAPIKey(created.Key.ID, u.ID))
	err = svc.DeleteAPIKey(created.Key.ID, u.ID)
	require.ErrorIs(t, err, os.ErrNotExist)
}

func TestCreateAPIKeyRequiresUserID(t *testing.T) {
	svc := newAuthService(t)
	_, err := svc.CreateAPIKey(service.CreateAPIKeyRequest{Name: "x"})
	require.Error(t, err)
}

func TestListAPIKeysWithoutState(t *testing.T) {
	keys, err := (&service.Service{}).ListAPIKeys("usr_x")
	require.NoError(t, err)
	assert.Empty(t, keys)
}
