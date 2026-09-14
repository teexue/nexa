package main

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/teexue/nexa/core/config"
	"github.com/teexue/nexakit/provider"
	"github.com/teexue/nexakit/session"
	"github.com/teexue/nexa/core/store"
)

// openTestStateDB opens a state.db under the temp home and binds it.
func openTestStateDB(t *testing.T, home string) *store.DB {
	t.Helper()
	db, err := store.Open(home)
	require.NoError(t, err)
	config.BindDB(db)
	return db
}

// TestChatTurnRequest_FirstTurnOmitsSessionID guards against passing an
// unsaved session id to PrepareRun (which would fail on LoadSession).
func TestChatTurnRequest_FirstTurnOmitsSessionID(t *testing.T) {
	state := &chatState{agent: "run-demo", sess: nil}
	req := chatTurnRequest("hi", state)
	assert.Empty(t, req.SessionID)
	assert.Empty(t, req.Messages)
}

// TestChatTurnRequest_ResumeTurn passes the live session id when a store
// exists so PrepareRun reuses the persisted conversation.
func TestChatTurnRequest_ResumeTurn(t *testing.T) {
	svc, _, home := newRunService(t)
	stateDB := openTestStateDB(t, home)
	defer stateDB.Close()
	svc.Store = store.NewSessionStore(stateDB)

	sess := session.New("run-demo")
	state := &chatState{svc: svc, agent: "run-demo", sess: sess}
	req := chatTurnRequest("hi", state)
	assert.Equal(t, sess.ID, req.SessionID)
}

// TestChatTurnRequest_NoStoreReplaysHistory keeps multi-turn context alive
// when session persistence is unavailable (e.g. mock runs without state.db).
func TestChatTurnRequest_NoStoreReplaysHistory(t *testing.T) {
	svc, _, _ := newRunService(t)
	svc.Store = nil
	sess := session.New("run-demo")
	sess.AddMessages(provider.Message{Role: "user", Content: "first"})
	state := &chatState{svc: svc, agent: "run-demo", sess: sess}
	req := chatTurnRequest("second", state)
	assert.Empty(t, req.SessionID)
	require.Len(t, req.Messages, 1)
	assert.Equal(t, "first", req.Messages[0].Content)
}

// TestPrepareChatTurn_FreshSessionDoesNotLookup exercises the first-turn path
// end to end: no SessionID is sent, so no store lookup can fail.
func TestPrepareChatTurn_FreshSessionDoesNotLookup(t *testing.T) {
	svc, reg, _ := newRunService(t)
	state := &chatState{svc: svc, agent: "run-demo", sess: nil}
	result, err := prepareChatTurn("hello", state, context.Background())
	require.NoError(t, err)
	defer result.Cleanup(reg)
	assert.NotEmpty(t, result.Session.ID)
	assert.Equal(t, "hello", result.Config.Prompt)
}

// TestHandleClearCommand verifies /clear saves the old session, then resets
// the handle so the next turn starts a fresh session via PrepareRun.
func TestHandleClearCommand(t *testing.T) {
	svc, _, home := newRunService(t)
	stateDB := openTestStateDB(t, home)
	defer stateDB.Close()
	svc.Store = store.NewSessionStore(stateDB)

	sess := session.New("run-demo")
	require.NoError(t, svc.Store.Save(sess))
	state := &chatState{svc: svc, agent: "run-demo", sess: sess}

	handleClearCommand(state)
	assert.Nil(t, state.sess, "next turn must start a fresh session")

	metas, err := svc.Store.List()
	require.NoError(t, err)
	assert.Len(t, metas, 1)
	assert.Equal(t, sess.ID, metas[0].ID)
}

// TestHandleAgentCommand_SwitchResetsSession verifies switching agents drops
// the session so the next turn starts fresh via PrepareRun.
func TestHandleAgentCommand_SwitchResetsSession(t *testing.T) {
	svc, _, home := newRunService(t)
	stateDB := openTestStateDB(t, home)
	defer stateDB.Close()

	state := &chatState{
		svc:   svc,
		paths: runtimePaths{agentsDir: config.AgentsDir(home)},
		agent: "run-demo",
		sess:  session.New("run-demo"),
	}
	exit := handleAgentCommand([]string{"/agent", "agt_run"}, state)
	assert.False(t, exit)
	assert.Equal(t, "agt_run", state.agent)
	assert.Nil(t, state.sess)
}