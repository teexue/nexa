package agent_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/teexue/nexa/core/agent"
)

func TestLoad_LegacyFileUsesStemAsID(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "coder.yaml")
	content := `name: coder
provider: openai
model: gpt-4o
system_prompt: hi
tools: [echo]
`
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
	a, err := agent.Load(path)
	require.NoError(t, err)
	assert.Equal(t, "coder", a.ID)
	assert.Equal(t, "coder", a.Name)
}

func TestResolve_ByIDAndName(t *testing.T) {
	dir := t.TempDir()
	content := `id: agt_abc123
name: Daily Reporter
provider: openai
model: gpt-4o
system_prompt: hi
tools: [echo]
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "agt_abc123.yaml"), []byte(content), 0o644))

	byID, err := agent.Resolve(dir, "agt_abc123")
	require.NoError(t, err)
	assert.Equal(t, "Daily Reporter", byID.Name)

	byName, err := agent.Resolve(dir, "Daily Reporter")
	require.NoError(t, err)
	assert.Equal(t, "agt_abc123", byName.ID)
}

func TestNewID(t *testing.T) {
	id := agent.NewID()
	assert.True(t, len(id) > 4)
	assert.Equal(t, "agt_", id[:4])
}
