package service_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/teexue/nexa/core/agent"
	"github.com/teexue/nexa/core/config"
	"github.com/teexue/nexakit/provider"
	"github.com/teexue/nexa/core/service"
	"github.com/teexue/nexakit/subagent"
	"github.com/teexue/nexakit/builtin"
	"github.com/teexue/nexakit/registry"
)

func TestSaveAndGetSubagentSettings(t *testing.T) {
	home := t.TempDir()
	bindConfigDB(t, home)
	svc := &service.Service{HomeDir: home}
	err := svc.SaveSubagentSettings(config.SubagentView{
		Enabled: false, MaxTurns: 7, MaxDepth: 2, Timeout: 15, MaxConcurrent: 3,
	})
	require.NoError(t, err)
	view, err := svc.GetSubagentSettings()
	require.NoError(t, err)
	assert.False(t, view.Enabled)
	assert.Equal(t, 7, view.MaxTurns)
	assert.Equal(t, config.DefaultSubagentMaxDepth, view.MaxDepth)
	assert.Equal(t, 15, view.Timeout)
	assert.Equal(t, 3, view.MaxConcurrent)
}

func TestPrepareRun_InjectsDelegateTask(t *testing.T) {
	home := t.TempDir()
	agentsDir := filepath.Join(home, "agents")
	require.NoError(t, os.MkdirAll(agentsDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(agentsDir, "agt_plain.yaml"), []byte(`id: agt_plain
name: plain
provider: mock
model: mock-1
system_prompt: hi
tools: [get_time]
`), 0o644))
	reg := registry.New()
	builtin.RegisterAll(reg, "")
	svc := service.New(service.ServiceConfig{
		AgentsDir:   agentsDir,
		HomeDir:     home,
		Registry:    reg,
		NewProvider: func(*agent.Agent) (provider.Provider, error) { return &provider.MockProvider{}, nil },
	})
	result, err := svc.PrepareRun(context.Background(), service.RunRequest{
		Agent: "agt_plain", Prompt: "hi",
	}, nil)
	require.NoError(t, err)
	assert.Contains(t, result.Config.Agent.Tools, subagent.ToolName)
	assert.True(t, result.Config.Subagent.Enabled)
	assert.Equal(t, config.DefaultSubagentMaxTurns, result.Config.Subagent.MaxTurns)
	assert.Equal(t, config.DefaultSubagentMaxDepth, result.Config.Subagent.MaxDepth)
	assert.Equal(t, config.DefaultSubagentMaxConcurrent, result.Config.Subagent.MaxConcurrent)
}
