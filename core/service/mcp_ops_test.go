package service_test

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/teexue/nexakit/event"
	"github.com/teexue/nexakit/loop"
	"github.com/teexue/nexakit/mcp"
	"github.com/teexue/nexakit/provider"
	kitmock "github.com/teexue/nexakit/provider/mock"
	"github.com/teexue/nexakit/registry"
	"github.com/teexue/nexakit/session"

	"github.com/teexue/nexa/core/agent"
	"github.com/teexue/nexa/core/config"
	"github.com/teexue/nexa/core/service"
)

// mockMCPScript writes a minimal JSON-RPC MCP server that exposes one tool
// named "mcp_echo" and returns its echoed text on tools/call.
func mockMCPScript(t *testing.T) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("skipping stdio MCP test on windows")
	}
	if _, err := exec.LookPath("python3"); err != nil {
		t.Skip("python3 not available for mock MCP server")
	}
	dir := t.TempDir()
	script := filepath.Join(dir, "mock-mcp.sh")
	content := `#!/bin/sh
while IFS= read -r line; do
  method=$(echo "$line" | python3 -c "import sys,json; print(json.load(sys.stdin).get('method',''))" 2>/dev/null)
  id=$(echo "$line" | python3 -c "import sys,json; print(json.load(sys.stdin).get('id',0))" 2>/dev/null)
  case "$method" in
    initialize)
      echo "{\"jsonrpc\":\"2.0\",\"id\":$id,\"result\":{\"protocolVersion\":\"2024-11-05\",\"capabilities\":{\"tools\":{}},\"serverInfo\":{\"name\":\"mock\",\"version\":\"0.1\"}}}"
      ;;
    notifications/initialized) ;;
    tools/list)
      echo "{\"jsonrpc\":\"2.0\",\"id\":$id,\"result\":{\"tools\":[{\"name\":\"mcp_echo\",\"description\":\"Echo via MCP\",\"inputSchema\":{\"type\":\"object\",\"properties\":{\"msg\":{\"type\":\"string\"}}}}]}}"
      ;;
    tools/call)
      echo "{\"jsonrpc\":\"2.0\",\"id\":$id,\"result\":{\"content\":[{\"type\":\"text\",\"text\":\"mcp-echo-result\"}]}}"
      ;;
  esac
done
`
	require.NoError(t, os.WriteFile(script, []byte(content), 0o755))
	return script
}

func writeAgentWithMCP(t *testing.T, dir, script string) {
	t.Helper()
	yaml := `id: agt_mcp01
name: mcp-demo
provider: mock
model: mock-1
system_prompt: you are a helper
tools: [get_time]
mcp_servers:
  - name: mock-srv
    type: stdio
    command: /bin/sh
    args:
      - ` + script + `
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "agt_mcp01.yaml"), []byte(yaml), 0o644))
}

func TestPrepareRun_InjectsMCPTools(t *testing.T) {
	script := mockMCPScript(t)
	agentsDir := t.TempDir()
	writeAgentWithMCP(t, agentsDir, script)

	reg := registry.New()
	registry.RegisterBuiltin(reg, "")

	toolCall := provider.ToolCall{
		ID:        "call_1",
		Name:      "mcp_echo",
		Arguments: json.RawMessage(`{"msg":"hi"}`),
	}
	mockProv := &kitmock.MockProvider{
		Calls: [][]kitmock.MockStep{
			{{ToolCalls: []provider.ToolCall{toolCall}}},
			{{Text: "done after mcp"}},
		},
	}

	svc := service.New(service.ServiceConfig{
		AgentsDir:   agentsDir,
		Registry:    reg,
		NewProvider: func(*agent.Agent) (provider.Provider, error) { return mockProv, nil },
		Logger:      slog.Default(),
	})

	result, err := svc.PrepareRun(context.Background(), service.RunRequest{
		Agent:  "agt_mcp01",
		Prompt: "use the mcp echo tool",
	}, nil)
	require.NoError(t, err)
	defer result.Cleanup(reg)

	// The MCP tool must have been registered and added to the agent whitelist.
	_, ok := reg.Get("mcp_echo")
	assert.True(t, ok, "mcp_echo should be registered")
	assert.Contains(t, result.Config.Agent.Tools, "mcp_echo")

	// Run the loop and confirm the MCP tool actually executes.
	events, err := loop.Run(context.Background(), result.Config)
	require.NoError(t, err)

	var sawToolResult bool
	var finalText string
	for ev := range events {
		if ev.Type == event.TypeToolResult && ev.Tool == "mcp_echo" {
			sawToolResult = true
			assert.Contains(t, string(ev.Output), "mcp-echo-result")
		}
		if ev.Type == event.TypeTextDelta {
			finalText += ev.Content
		}
	}
	assert.True(t, sawToolResult, "loop should have executed mcp_echo")
	assert.Equal(t, "done after mcp", finalText)

	// Cleanup unregisters the MCP tool and closes the manager.
	result.Cleanup(reg)

	_, ok = reg.Get("mcp_echo")
	assert.False(t, ok, "mcp_echo should be unregistered after cleanup")
}

func TestPrepareRun_NoMCPServers(t *testing.T) {
	agentsDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(agentsDir, "agt_plain.yaml"), []byte(`id: agt_plain
name: plain
provider: mock
model: mock-1
system_prompt: hi
tools: [get_time]
`), 0o644))

	reg := registry.New()
	registry.RegisterBuiltin(reg, "")
	mockProv := &kitmock.MockProvider{Calls: [][]kitmock.MockStep{
		{{Text: "ok"}},
	}}
	svc := service.New(service.ServiceConfig{
		AgentsDir:   agentsDir,
		Registry:    reg,
		NewProvider: func(*agent.Agent) (provider.Provider, error) { return mockProv, nil },
		Logger:      slog.Default(),
	})

	result, err := svc.PrepareRun(context.Background(), service.RunRequest{
		Agent: "agt_plain", Prompt: "hi",
	}, nil)
	require.NoError(t, err)
	defer result.Cleanup(reg)

	assert.Nil(t, result.MCPManager)
	assert.Empty(t, result.MCPToolNames)
	// Ensure the session is wired so Cleanup is safe even with no MCP.
	_ = session.New("agt_plain")
}

func TestPrepareRun_GlobalMCPMerged(t *testing.T) {
	script := mockMCPScript(t)
	home := t.TempDir()
	agentsDir := filepath.Join(home, "agents")
	require.NoError(t, os.MkdirAll(agentsDir, 0o755))

	// Agent without any mcp_servers — it inherits the global server.
	require.NoError(t, os.WriteFile(filepath.Join(agentsDir, "agt_g01.yaml"), []byte(`id: agt_g01
name: g-demo
provider: mock
model: mock-1
system_prompt: hi
tools: [get_time]
`), 0o644))

	bindConfigDB(t, home)
	require.NoError(t, config.UpsertGlobalMCP(home, mcp.ServerConfig{
		Name:    "global-srv",
		Type:    "stdio",
		Command: "/bin/sh",
		Args:    []string{script},
	}))

	reg := registry.New()
	registry.RegisterBuiltin(reg, "")
	toolCall := provider.ToolCall{
		ID: "call_1", Name: "mcp_echo",
		Arguments: json.RawMessage(`{"msg":"hi"}`),
	}
	mockProv := &kitmock.MockProvider{Calls: [][]kitmock.MockStep{
		{{ToolCalls: []provider.ToolCall{toolCall}}},
		{{Text: "ok"}},
	}}
	svc := service.New(service.ServiceConfig{
		AgentsDir:   agentsDir,
		Registry:    reg,
		NewProvider: func(*agent.Agent) (provider.Provider, error) { return mockProv, nil },
		Logger:      slog.Default(),
	})

	result, err := svc.PrepareRun(context.Background(), service.RunRequest{
		Agent: "agt_g01", Prompt: "use global mcp",
	}, nil)
	require.NoError(t, err)
	defer result.Cleanup(reg)

	_, ok := reg.Get("mcp_echo")
	assert.True(t, ok, "global mcp_echo should be registered via global config")
	assert.Contains(t, result.Config.Agent.Tools, "mcp_echo")
}
