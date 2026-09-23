package main

import (
	"fmt"
	"os"

	"github.com/teexue/nexakit/provider"
	kitcatalog "github.com/teexue/nexakit/provider/catalog"
	kitmock "github.com/teexue/nexakit/provider/mock"
	"github.com/teexue/nexakit/registry"

	"github.com/teexue/nexa/core/agent"
	"github.com/teexue/nexa/core/service"
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
	registry.RegisterBuiltin(reg, workDir)
	return reg
}

func resolveProvider(catalog *kitcatalog.Catalog, useMock bool) func(a *agent.Agent) (provider.Provider, error) {
	return func(a *agent.Agent) (provider.Provider, error) {
		if useMock {
			return mockProvider(), nil
		}
		if catalog == nil {
			return nil, fmt.Errorf("no provider configured; add one in the Settings UI or run: nexa config set provider")
		}
		return catalog.ResolveForAgent(a.Provider)
	}
}

func mockProvider() provider.Provider {
	return &kitmock.MockProvider{
		Calls: [][]kitmock.MockStep{
			{{
				ToolCalls: []provider.ToolCall{{
					ID: "call_1", Name: "get_time", Arguments: []byte("{}"),
				}},
			}},
			{{Text: "The current time has been retrieved."}},
		},
	}
}
