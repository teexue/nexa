//go:build integration

package integration

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/teexue/nexa/core/config"
	"github.com/teexue/nexa/core/store"
)

func TestAuthRequired(t *testing.T) {
	srv := newRunServer(t, runAgentYAML, textMock())
	db, err := store.Open(t.TempDir())
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = db.Close()
		config.BindDB(nil)
	})
	require.NoError(t, srv.SetStateDB(db))

	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)

	body, err := json.Marshal(map[string]string{"agent": "test", "prompt": "hello"})
	require.NoError(t, err)
	resp, err := http.Post(ts.URL+"/v1/agents/run", "application/json", bytes.NewReader(body))
	require.NoError(t, err)
	t.Cleanup(func() { _ = resp.Body.Close() })
	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}
