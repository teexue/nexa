//go:build integration

package integration

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/teexue/common-agent/core/agent"
	"github.com/teexue/nexakit/provider"
	"github.com/teexue/nexakit/tool"
	httpapi "github.com/teexue/common-agent/server/http"
	"github.com/teexue/nexakit/registry"
)

const runAgentYAML = `name: test
version: 1
provider: mock
model: test-model
system_prompt: |
  You are a test assistant.
tools:
  - test_tool
max_turns: 5
max_tokens: 1024
`

const gatedAgentYAML = runAgentYAML + `
permissions:
  auto_approve: []
  always_deny: []
`

type mockTool struct{}

func (m *mockTool) Name() string        { return "test_tool" }
func (m *mockTool) Description() string { return "A test tool" }
func (m *mockTool) InputSchema() map[string]any {
	return map[string]any{"type": "object", "properties": map[string]any{}}
}
func (m *mockTool) Execute(_ context.Context, _ json.RawMessage) (tool.Result, error) {
	return tool.Result{Output: json.RawMessage(`"ok"`)}, nil
}

func newRunServer(t *testing.T, yaml string, mock *provider.MockProvider) *httpapi.Server {
	t.Helper()
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "test.yaml"), []byte(yaml), 0o644))
	reg := registry.New()
	require.NoError(t, reg.Register(&mockTool{}))
	return httpapi.NewServer(httpapi.ServerConfig{
		AgentsDir: dir,
		HomeDir:   dir,
		Registry:  reg,
		NewProvider: func(_ *agent.Agent) (provider.Provider, error) {
			return mock, nil
		},
	})
}

func textMock() *provider.MockProvider {
	return &provider.MockProvider{
		Calls: [][]provider.MockStep{{{Text: "hello from loop"}}},
	}
}

func toolCallMock() *provider.MockProvider {
	return &provider.MockProvider{
		Calls: [][]provider.MockStep{
			{{ToolCalls: []provider.ToolCall{{ID: "tc-1", Name: "test_tool", Arguments: json.RawMessage(`{}`)}}}},
			{{Text: "approved"}},
		},
	}
}
