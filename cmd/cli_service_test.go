package main

import (
	"context"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/teexue/nexa/core/agent"
	"github.com/teexue/nexa/core/config"
	"github.com/teexue/nexakit/loop"
	"github.com/teexue/nexakit/provider"
	"github.com/teexue/nexa/core/service"
	"github.com/teexue/nexakit/session"
	"github.com/teexue/nexa/core/store"
	"github.com/teexue/nexakit/registry"
)

// testLogger silences slog output during tests.
func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// writeRunAgent persists a minimal agent YAML for run tests.
func writeRunAgent(t *testing.T, dir, id string) {
	t.Helper()
	yaml := `id: ` + id + `
name: run-demo
provider: mock
model: mock-1
system_prompt: you are a helper
tools: [get_time]
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, id+".yaml"), []byte(yaml), 0o644))
}

// newRunService wires a CLI service against a temp home without touching the
// user's real ~/.nexa.
func newRunService(t *testing.T) (*service.Service, *registry.Registry, string) {
	t.Helper()
	home := t.TempDir()
	agentsDir := config.AgentsDir(home)
	require.NoError(t, os.MkdirAll(agentsDir, 0o755))
	writeRunAgent(t, agentsDir, "agt_run")
	reg := newRegistry(agentsDir)
	svc := service.New(service.ServiceConfig{
		AgentsDir: agentsDir,
		HomeDir:   home,
		Registry:  reg,
		NewProvider: func(*agent.Agent) (provider.Provider, error) {
			return provider.EchoThenReply("hi"), nil
		},
		Logger: testLogger(),
	})
	return svc, reg, home
}

func TestCLIRunApprover(t *testing.T) {
	tests := []struct {
		name       string
		yes        bool
		jsonFormat bool
		want       string
	}{
		{name: "yes auto approves", yes: true, want: "auto"},
		{name: "json denies", jsonFormat: true, want: "deny"},
		{name: "interactive prompts", want: "cli"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ap := cliRunApprover(tt.yes, tt.jsonFormat)
			switch tt.want {
			case "auto":
				approved := ap.Approve(context.Background(), loop.ApprovalRequest{Tool: "run_command"})
				assert.True(t, approved)
			case "deny":
				approved := ap.Approve(context.Background(), loop.ApprovalRequest{Tool: "run_command"})
				assert.False(t, approved)
			case "cli":
				_, ok := ap.(CLIApprover)
				assert.True(t, ok)
			}
		})
	}
}

func TestPrepareRunViaCLIService(t *testing.T) {
	svc, reg, _ := newRunService(t)
	result, err := svc.PrepareRun(context.Background(), service.RunRequest{
		Agent: "run-demo", Prompt: "hello", Source: "cli",
	}, loop.AutoApprover{})
	require.NoError(t, err)
	defer result.Cleanup(reg)

	assert.Equal(t, "hello", result.Config.Prompt)
	assert.NotNil(t, result.Config.Session)
	assert.Equal(t, "run-demo", result.Config.Agent.Name)
}

func TestPrepareRunResumesSession(t *testing.T) {
	svc, reg, home := newRunService(t)
	stateDB, err := store.Open(home)
	require.NoError(t, err)
	defer stateDB.Close()
	config.BindDB(stateDB)
	svc.Store = store.NewSessionStore(stateDB)

	// Seed a persisted session for the agent.
	saved := session.New("run-demo")
	require.NoError(t, svc.Store.Save(saved))

	result, err := svc.PrepareRun(context.Background(), service.RunRequest{
		Agent: "run-demo", Prompt: "next", SessionID: saved.ID, Source: "cli",
	}, loop.AutoApprover{})
	require.NoError(t, err)
	defer result.Cleanup(reg)
	assert.Equal(t, saved.ID, result.Session.ID)
}

func TestLatestSessionID(t *testing.T) {
	svc, _, home := newRunService(t)
	stateDB, err := store.Open(home)
	require.NoError(t, err)
	defer stateDB.Close()
	config.BindDB(stateDB)
	svc.Store = store.NewSessionStore(stateDB)

	// Sessions persist the agent ID (as PrepareRun does via NewForUser(a.ID)).
	s1 := session.New("agt_run")
	require.NoError(t, svc.Store.Save(s1))
	s2 := session.New("agt_other")
	require.NoError(t, svc.Store.Save(s2))

	// A session saved under the display name must still be found via its ID,
	// so --continue works regardless of how the agent was referenced.
	s3 := session.New("run-demo")
	require.NoError(t, svc.Store.Save(s3))

	got, err := latestSessionID(svc, "agt_run")
	require.NoError(t, err)
	assert.Equal(t, s1.ID, got)

	got, err = latestSessionID(svc, "run-demo")
	require.NoError(t, err)
	assert.Equal(t, s3.ID, got)

	got, err = latestSessionID(svc, "agt_unknown")
	require.NoError(t, err)
	assert.Empty(t, got)
}

// TestResolveRunAgent_FallsBackToFirstWhenDefaultMissing covers the CLI
// default path that previously failed on the chat-assistant placeholder.
func TestResolveRunAgent_FallsBackToFirstWhenDefaultMissing(t *testing.T) {
	_, _, home := newRunService(t)
	agentsDir := config.AgentsDir(home)

	a, err := resolveRunAgent(agentsDir, "", "chat-assistant")
	require.NoError(t, err)
	assert.Equal(t, "agt_run", a.ID)
}

// TestResolveRunAgent_ExplicitMissingStillErrors keeps --agent strict.
func TestResolveRunAgent_ExplicitMissingStillErrors(t *testing.T) {
	_, _, home := newRunService(t)
	_, err := resolveRunAgent(config.AgentsDir(home), "missing-agent", "chat-assistant")
	require.Error(t, err)
}