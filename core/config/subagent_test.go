package config_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/teexue/nexa/core/config"
)

func TestSubagentView_Defaults(t *testing.T) {
	v := config.Settings{}.SubagentView()
	assert.True(t, v.Enabled)
	assert.Equal(t, config.DefaultSubagentMaxTurns, v.MaxTurns)
	assert.Equal(t, config.DefaultSubagentMaxDepth, v.MaxDepth)
	assert.Equal(t, 0, v.Timeout)
}

func TestSubagentView_RoundTrip(t *testing.T) {
	home := bindTestDB(t)
	enabled := false
	require.NoError(t, config.SaveSettings(home, config.Settings{
		DefaultAgent: "chat-assistant",
		Subagent: &config.SubagentSettings{
			Enabled:  &enabled,
			MaxTurns: 9,
			MaxDepth: 2,
			Timeout:  30,
		},
	}))
	s, err := config.LoadSettings(home)
	require.NoError(t, err)
	v := s.SubagentView()
	assert.False(t, v.Enabled)
	assert.Equal(t, 9, v.MaxTurns)
	assert.Equal(t, config.DefaultSubagentMaxDepth, v.MaxDepth)
	assert.Equal(t, 30, v.Timeout)
}

func TestSubagentView_IgnoresPersistedMaxDepth(t *testing.T) {
	enabled := true
	v := config.Settings{
		Subagent: &config.SubagentSettings{
			Enabled:  &enabled,
			MaxTurns: 5,
			MaxDepth: 8,
		},
	}.SubagentView()
	assert.Equal(t, config.DefaultSubagentMaxDepth, v.MaxDepth)
	assert.Equal(t, 0, v.Persistable().MaxDepth)
}
