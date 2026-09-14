package service_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/teexue/nexa/core/config"
)

func bindConfigDB(t *testing.T, home string) {
	t.Helper()
	db, err := config.OpenAndBind(home)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = db.Close()
		config.BindDB(nil)
	})
}
