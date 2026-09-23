package service_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/teexue/nexakit/provider"
	kitmock "github.com/teexue/nexakit/provider/mock"
	"github.com/teexue/nexakit/registry"

	"github.com/teexue/nexa/core/agent"
	"github.com/teexue/nexa/core/kanban"
	"github.com/teexue/nexa/core/store"
)

// Kanban retries (reject/failure) must continue the original session, never
// start a fresh conversation.
func TestKanbanRunnerRetryContinuesSession(t *testing.T) {
	agentsDir := t.TempDir()
	writePlainAgent(t, agentsDir)
	mock := &kitmock.MockProvider{Calls: [][]kitmock.MockStep{
		{{Text: "first attempt output"}},
		{{Text: "second attempt output"}},
	}}
	sessStore := newMemStore()
	svc := newWorkdirService(t, agentsDir, sessStore)
	registry.RegisterBuiltin(svc.Registry, t.TempDir())
	svc.NewProvider = func(*agent.Agent) (provider.Provider, error) { return mock, nil }

	row := &store.KanbanRow{
		ID: kanban.NewID(), UserID: "usr_local", Title: "t", Prompt: "build it",
		Agent: "agt_wd", Status: kanban.StatusPending, Priority: kanban.PriorityMedium,
	}
	runner := svc.KanbanRunner()

	_, sid1, err := runner(context.Background(), row)
	require.NoError(t, err)
	require.NotEmpty(t, sid1)

	// Simulate a human rejection: back to pending with feedback.
	row.Status = kanban.StatusPending
	row.Feedback = "please redo the scoring part"

	_, sid2, err := runner(context.Background(), row)
	require.NoError(t, err)
	assert.Equal(t, sid1, sid2, "retry must continue the same session")

	sess, err := sessStore.Load(sid1)
	require.NoError(t, err)
	msgs := sess.GetMessages()
	// History from attempt 1 + the retry prompt with feedback appended.
	var lastUser string
	for _, m := range msgs {
		if m.Role == provider.RoleUser {
			lastUser = m.Content
		}
	}
	assert.Contains(t, lastUser, "please redo the scoring part")
	assert.Greater(t, len(msgs), 3, "session should retain earlier turns")
}
