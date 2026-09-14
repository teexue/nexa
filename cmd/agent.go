package main

import (
	"fmt"
	"os"

	"github.com/teexue/common-agent/core/agent"
	"github.com/teexue/nexakit/provider"
	"github.com/teexue/common-agent/core/service"
	"github.com/teexue/nexakit/builtin"
	"github.com/teexue/nexakit/registry"
)

// resolveRunAgent picks the agent for CLI run/chat: an explicit --agent must
// exist; otherwise try settings default then the first agent under agentsDir.
func resolveRunAgent(agentsDir, flagAgent, defaultAgent string) (*agent.Agent, error) {
	if flagAgent != "" {
		return agent.LoadByName(agentsDir, service.NormalizeAgentName(flagAgent))
	}
	return agent.ResolveDefault(agentsDir, service.NormalizeAgentName(defaultAgent))
}

func newRegistry(workDir string) *registry.Registry {
	// If no workDir specified, use current working directory.
	if workDir == "" {
		if cwd, err := os.Getwd(); err == nil {
			workDir = cwd
		}
	}
	reg := registry.New()
	builtin.RegisterAll(reg, workDir)
	return reg
}

func resolveProvider(catalog *provider.Catalog, useMock bool) func(a *agent.Agent) (provider.Provider, error) {
	return func(a *agent.Agent) (provider.Provider, error) {
		if useMock {
			return mockProvider(), nil
		}
		if catalog == nil {
			return nil, fmt.Errorf("no provider configured; add one in the Settings UI or run: common-agent config set provider")
		}
		return catalog.ResolveForAgent(a.Provider)
	}
}

func mockProvider() provider.Provider {
	return &provider.MockProvider{
		Calls: [][]provider.MockStep{
			{{
				ToolCalls: []provider.ToolCall{{
					ID: "call_1", Name: "get_time", Arguments: []byte("{}"),
				}},
			}},
			{{Text: "The current time has been retrieved."}},
		},
	}
}
