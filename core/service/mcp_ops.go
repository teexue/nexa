package service

import (
	"context"
	"log/slog"
	"path/filepath"

	"github.com/teexue/common-agent/core/agent"
	"github.com/teexue/common-agent/core/config"
	"github.com/teexue/nexakit/mcp"
	"github.com/teexue/common-agent/core/version"
	"github.com/teexue/nexakit/registry"
)

// injectMCP connects to the agent's MCP servers plus any global shared MCP
// servers, discovers their tools, registers them in the registry, and appends
// their names to the agent's tool whitelist so the loop exposes them to the
// LLM.
//
// Global servers (from state.db) are merged with the agent's
// own mcp_servers; on name collision the agent-level config wins.
//
// Returns the manager (so the caller can close it after the run) and the
// names of tools that were registered. Failed servers are logged and skipped;
// a missing or unreachable MCP server never aborts the run.
func injectMCP(ctx context.Context, a *agent.Agent, agentsDir string, reg *registry.Registry, log *slog.Logger) (*mcp.Manager, []string) {
	servers := resolveMCPServers(a, agentsDir, log)
	if len(servers) == 0 {
		return nil, nil
	}

	mgr := mcp.NewManager(servers, log, mcp.ClientInfo{Name: "common-agent", Version: version.Version})
	tools := mgr.ConnectAll(ctx)

	var names []string
	for _, t := range tools {
		if err := reg.Register(t); err != nil {
			// Name collision with a built-in or skill tool; skip but keep going.
			log.Warn("log.mcp.register_tool", "tool", t.Name(), "error", err)
			continue
		}
		names = append(names, t.Name())
		a.Tools = append(a.Tools, t.Name())
	}
	return mgr, names
}

// resolveMCPServer merges global MCP servers with the agent's own, applying
// per-agent override on name collision.
func resolveMCPServers(a *agent.Agent, agentsDir string, log *slog.Logger) []mcp.ServerConfig {
	byName := make(map[string]mcp.ServerConfig)

	home := filepath.Dir(agentsDir)
	if global, err := config.LoadGlobalMCP(home); err != nil {
		log.Warn("log.mcp.load_global", "error", err)
	} else {
		for _, s := range global {
			byName[s.Name] = s
		}
	}

	// Agent-level servers override globals with the same name.
	for _, s := range a.MCPServers {
		byName[s.Name] = mcp.ServerConfig{
			Name:    s.Name,
			Type:    s.Type,
			Command: s.Command,
			Args:    s.Args,
			Env:     s.Env,
			URL:     s.URL,
		}
	}

	servers := make([]mcp.ServerConfig, 0, len(byName))
	for _, s := range byName {
		servers = append(servers, s)
	}
	return servers
}
