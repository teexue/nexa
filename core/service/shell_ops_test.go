package service_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/teexue/nexakit/tool/builtin"

	"github.com/teexue/nexa/core/service"
)

func TestSaveAndGetShellSettings(t *testing.T) {
	home := t.TempDir()
	bindConfigDB(t, home)
	svc := &service.Service{HomeDir: home}

	view, err := svc.GetShellSettings()
	require.NoError(t, err)
	assert.Equal(t, "auto", view.Shell)
	require.NotEmpty(t, view.Available)
	assert.Equal(t, view.Available[0].ID, view.Resolved.ID)

	_, err = svc.SaveShellSettings("not-a-shell")
	require.Error(t, err)

	saved, err := svc.SaveShellSettings(view.Available[0].ID)
	require.NoError(t, err)
	assert.Equal(t, view.Available[0].ID, saved.Shell)

	got, err := svc.GetShellSettings()
	require.NoError(t, err)
	assert.Equal(t, view.Available[0].ID, got.Shell)

	auto, err := svc.SaveShellSettings("auto")
	require.NoError(t, err)
	assert.Equal(t, "auto", auto.Shell)
	assert.Equal(t, builtin.HostOS(), auto.OS)
}
