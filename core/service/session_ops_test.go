package service_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/teexue/nexa/core/service"
	"github.com/teexue/nexakit/session"
)

func TestListSessionsExcludesKanbanSessions(t *testing.T) {
	store := newMemStore()
	svc := service.New(service.ServiceConfig{Store: store})

	normal := session.NewForUser("agent", "usr_local")
	require.NoError(t, store.Save(normal))

	fromBoard := session.NewForUser("agent", "usr_local")
	fromBoard.SetMetadata(session.MetadataKeySource, session.SourceKanban)
	require.NoError(t, store.Save(fromBoard))

	fromSub := session.NewForUser("agent", "usr_local")
	fromSub.SetMetadata(session.MetadataKeySource, session.SourceSubagent)
	require.NoError(t, store.Save(fromSub))

	metas, err := svc.ListSessions("usr_local")
	require.NoError(t, err)
	require.Len(t, metas, 1)
	assert.Equal(t, normal.ID, metas[0].ID)
}
