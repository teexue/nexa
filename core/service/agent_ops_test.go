package service_test

import (
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/teexue/nexa/core/service"
)

func TestSaveAgent_RenameDisplayName(t *testing.T) {
	dir := t.TempDir()
	svc := &service.Service{AgentsDir: dir, Logger: slog.Default()}

	yaml1 := []byte(`id: agt_test01
name: old-name
provider: openai
model: gpt-4o
system_prompt: hi
tools: [echo]
`)
	require.NoError(t, svc.SaveAgent("agt_test01", yaml1))

	yaml2 := []byte(`id: agt_test01
name: new-name
provider: openai
model: gpt-4o
system_prompt: hi
tools: [echo]
`)
	require.NoError(t, svc.SaveAgent("agt_test01", yaml2))

	a, err := svc.GetAgent("agt_test01")
	require.NoError(t, err)
	assert.Equal(t, "new-name", a.Name)
	assert.Equal(t, "agt_test01", a.ID)

	// File stays under id
	_, err = os.Stat(filepath.Join(dir, "agt_test01.yaml"))
	require.NoError(t, err)

	byName, err := svc.GetAgent("new-name")
	require.NoError(t, err)
	assert.Equal(t, "agt_test01", byName.ID)
}

func TestCreateAgent_AssignsID(t *testing.T) {
	dir := t.TempDir()
	svc := &service.Service{AgentsDir: dir, Logger: slog.Default()}

	yaml := []byte(`name: fresh
provider: openai
model: gpt-4o
system_prompt: hi
tools: [echo]
`)
	a, err := svc.CreateAgent(yaml)
	require.NoError(t, err)
	assert.NotEmpty(t, a.ID)
	assert.Equal(t, "fresh", a.Name)
	assert.Equal(t, "agt_", a.ID[:4])
}

func TestListAgents_ContextWindow(t *testing.T) {
	dir := t.TempDir()
	svc := &service.Service{AgentsDir: dir, Logger: slog.Default()}

	// Model with an official spec → 1M window.
	require.NoError(t, svc.SaveAgent("spec", []byte(`id: spec
name: spec-agent
provider: openai
model: deepseek-v4-pro
system_prompt: hi
tools: [echo]
`)))
	// Unknown model + explicit compaction window → configured value wins.
	require.NoError(t, svc.SaveAgent("cfg", []byte(`id: cfg
name: cfg-agent
provider: openai
model: unknown-model
system_prompt: hi
tools: [echo]
compaction:
  context_window: 256000
`)))

	summaries := svc.ListAgents()
	require.Len(t, summaries, 2)

	byID := map[string]service.AgentSummary{}
	for _, s := range summaries {
		byID[s.ID] = s
	}
	assert.Equal(t, 1_000_000, byID["spec"].ContextWindow, "model spec window")
	assert.Equal(t, 256000, byID["cfg"].ContextWindow, "configured compaction window")

	// Unknown model with no compaction window → omit (do not advertise 128K).
	require.NoError(t, svc.SaveAgent("unk", []byte(`id: unk
name: unk-agent
provider: ollama
model: glm5_next
system_prompt: hi
tools: [echo]
`)))
	summaries = svc.ListAgents()
	byID = map[string]service.AgentSummary{}
	for _, s := range summaries {
		byID[s.ID] = s
	}
	assert.Equal(t, 0, byID["unk"].ContextWindow, "unknown model has no advertised window")

	svc.ModelWindow = func(providerName, model string) int {
		if providerName == "ollama" && model == "glm5_next" {
			return 1_000_000
		}
		return 0
	}
	summaries = svc.ListAgents()
	byID = map[string]service.AgentSummary{}
	for _, s := range summaries {
		byID[s.ID] = s
	}
	assert.Equal(t, 1_000_000, byID["unk"].ContextWindow, "provider-saved window")
}
