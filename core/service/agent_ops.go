package service

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/teexue/nexa/core/agent"
	"github.com/teexue/nexakit/provider"
)

// AgentSummary is the lightweight representation of an agent for list endpoints.
type AgentSummary struct {
	ID            string   `json:"id"`
	Name          string   `json:"name"`
	Provider      string   `json:"provider"`
	Model         string   `json:"model"`
	Tools         []string `json:"tools"`
	MaxTurns      int      `json:"max_turns"`
	ContextWindow int      `json:"context_window,omitempty"`
}

// agentContextWindow is the window advertised on the agent list: explicit
// compaction config, then a window saved on the provider (Ollama /api/show
// at setup), then an official model spec. Unknown models stay 0.
func (s *Service) agentContextWindow(a *agent.Agent) int {
	if a.Compaction != nil && a.Compaction.ContextWindow > 0 {
		return a.Compaction.ContextWindow
	}
	if s.ModelWindow != nil {
		if n := s.ModelWindow(a.Provider, a.Model); n > 0 {
			return n
		}
	}
	if spec, ok := provider.SpecForModel(a.Model); ok {
		return spec.ContextWindow
	}
	return 0
}

func (s *Service) savedContextWindow(a *agent.Agent) int {
	if s.ModelWindow == nil || a == nil {
		return 0
	}
	return s.ModelWindow(a.Provider, a.Model)
}

// ListAgents loads all agents and returns summaries. Errors per-agent are logged
// but do not fail the entire request.
func (s *Service) ListAgents() []AgentSummary {
	result, err := agent.LoadAll(s.AgentsDir)
	if err != nil {
		s.Logger.Warn("log.agent.load_all", "error", err)
		return nil
	}
	for _, e := range result.Errors {
		s.Logger.Warn("log.agent.load_error", "error", e)
	}
	out := make([]AgentSummary, len(result.Agents))
	for i, a := range result.Agents {
		out[i] = AgentSummary{
			ID:            a.ID,
			Name:          a.Name,
			Provider:      a.Provider,
			Model:         a.Model,
			Tools:         a.Tools,
			MaxTurns:      a.MaxTurns,
			ContextWindow: s.agentContextWindow(a),
		}
	}
	return out
}

// GetAgent loads a single agent by id or display name.
func (s *Service) GetAgent(ref string) (*agent.Agent, error) {
	ref = NormalizeAgentName(ref)
	if ref == "" {
		return nil, fmt.Errorf("agent id is required")
	}
	return agent.Resolve(s.AgentsDir, ref)
}

// CreateAgent validates and persists a new agent, assigning an id when missing.
func (s *Service) CreateAgent(yamlContent []byte) (*agent.Agent, error) {
	if len(yamlContent) == 0 {
		return nil, fmt.Errorf("agent YAML content is required")
	}
	a, err := agent.LoadFromBytes(yamlContent)
	if err != nil {
		return nil, fmt.Errorf("invalid agent YAML: %w", err)
	}
	if a.ID == "" {
		a.ID = agent.NewID()
	}
	if err := s.ensureUniqueName(a.ID, a.Name); err != nil {
		return nil, err
	}
	if err := s.writeAgent(a); err != nil {
		return nil, err
	}
	s.Logger.Info("log.agent.created", "id", a.ID, "name", a.Name)
	return a, nil
}

// SaveAgent validates and persists an agent YAML file under agents/{id}.yaml.
// The display name may differ from the id (rename-safe).
func (s *Service) SaveAgent(id string, yamlContent []byte) error {
	id = NormalizeAgentName(id)
	if id == "" {
		return fmt.Errorf("agent id is required")
	}
	if len(yamlContent) == 0 {
		return fmt.Errorf("agent YAML content is required")
	}
	a, err := agent.LoadFromBytes(yamlContent)
	if err != nil {
		return fmt.Errorf("invalid agent YAML: %w", err)
	}
	if a.ID == "" {
		a.ID = id
	}
	if a.ID != id {
		return fmt.Errorf("agent id %q does not match URL %q", a.ID, id)
	}
	if err := s.ensureUniqueName(a.ID, a.Name); err != nil {
		return err
	}
	if err := s.writeAgent(a); err != nil {
		return err
	}
	s.Logger.Info("log.agent.saved", "id", a.ID, "name", a.Name)
	return nil
}

func (s *Service) writeAgent(a *agent.Agent) error {
	if err := os.MkdirAll(s.AgentsDir, 0o755); err != nil {
		return fmt.Errorf("create agents dir: %w", err)
	}
	data, err := agent.Marshal(a)
	if err != nil {
		return fmt.Errorf("marshal agent: %w", err)
	}
	path := filepath.Join(s.AgentsDir, a.ID+".yaml")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write agent file: %w", err)
	}
	return nil
}

func (s *Service) ensureUniqueName(id, name string) error {
	result, err := agent.LoadAll(s.AgentsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	for _, other := range result.Agents {
		if other.ID == id {
			continue
		}
		if other.Name == name {
			return fmt.Errorf("agent name %q already used by id %q", name, other.ID)
		}
	}
	return nil
}

// DeleteAgent removes an agent YAML file by id. Returns os.ErrNotExist if not found.
func (s *Service) DeleteAgent(id string) error {
	id = NormalizeAgentName(id)
	if id == "" {
		return fmt.Errorf("agent id is required")
	}
	path := filepath.Join(s.AgentsDir, id+".yaml")
	if err := os.Remove(path); err != nil {
		return err
	}
	s.Logger.Info("log.agent.deleted", "id", id)
	return nil
}

// NormalizeAgentName strips .yaml suffix and normalizes an agent id or name ref.
func NormalizeAgentName(name string) string {
	return strings.TrimSuffix(strings.TrimSpace(name), ".yaml")
}
