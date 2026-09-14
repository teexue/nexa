package agent_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/teexue/nexa/core/agent"
)

func writeAgentYAML(t *testing.T, dir, id, name string) {
	t.Helper()
	content := "id: " + id + "\nname: " + name + "\nprovider: p\nmodel: m\nsystem_prompt: hi\ntools: [echo]\n"
	require.NoError(t, os.WriteFile(filepath.Join(dir, id+".yaml"), []byte(content), 0o644))
}

func TestResolveDefault_UsesPreferredWhenPresent(t *testing.T) {
	dir := t.TempDir()
	writeAgentYAML(t, dir, "agt_a", "alpha")
	writeAgentYAML(t, dir, "agt_b", "beta")

	a, err := agent.ResolveDefault(dir, "agt_b")
	require.NoError(t, err)
	assert.Equal(t, "agt_b", a.ID)
}

func TestResolveDefault_FallsBackToFirstWhenMissing(t *testing.T) {
	dir := t.TempDir()
	writeAgentYAML(t, dir, "agt_z", "zeta")
	writeAgentYAML(t, dir, "agt_a", "alpha")

	a, err := agent.ResolveDefault(dir, "chat-assistant")
	require.NoError(t, err)
	assert.Equal(t, "agt_a", a.ID, "first by sorted id")
}

func TestResolveDefault_EmptyPreferredUsesFirst(t *testing.T) {
	dir := t.TempDir()
	writeAgentYAML(t, dir, "agt_m", "mid")

	a, err := agent.ResolveDefault(dir, "")
	require.NoError(t, err)
	assert.Equal(t, "agt_m", a.ID)
}

func TestResolveDefault_NoAgents(t *testing.T) {
	_, err := agent.ResolveDefault(t.TempDir(), "chat-assistant")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no agents configured")
}
