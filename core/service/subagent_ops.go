package service

import (
	"fmt"

	"github.com/teexue/common-agent/core/agent"
	"github.com/teexue/common-agent/core/config"
	"github.com/teexue/nexakit/loop"
	"github.com/teexue/nexakit/subagent"
)

const (
	maxSubagentTurns        = 100
	maxSubagentTimeout      = 3600
	maxSubagentConcurrent   = 32
)

// GetSubagentSettings returns normalized global sub-agent limits.
func (s *Service) GetSubagentSettings() (config.SubagentView, error) {
	if s.HomeDir == "" {
		return config.Settings{}.SubagentView(), nil
	}
	settings, err := s.loadSettings()
	if err != nil {
		return config.SubagentView{}, err
	}
	return settings.SubagentView(), nil
}

// SaveSubagentSettings persists global sub-agent limits.
func (s *Service) SaveSubagentSettings(view config.SubagentView) error {
	if err := validateSubagentView(&view); err != nil {
		return err
	}
	settings, err := s.loadSettings()
	if err != nil {
		return err
	}
	settings.Subagent = view.Persistable()
	return config.SaveSettings(s.HomeDir, settings)
}

func (s *Service) loadSettings() (config.Settings, error) {
	if s.HomeDir == "" {
		return config.Settings{}, fmt.Errorf("home dir not set")
	}
	return config.LoadSettings(s.HomeDir)
}

func (s *Service) applySubagent(a *agent.Agent) loop.SubagentLimits {
	view := config.Settings{}.SubagentView()
	if s.HomeDir != "" {
		if settings, err := config.LoadSettings(s.HomeDir); err == nil {
			view = settings.SubagentView()
		}
	}
	limits := loop.SubagentLimits{
		Enabled: view.Enabled, MaxTurns: view.MaxTurns,
		MaxDepth: config.DefaultSubagentMaxDepth, Timeout: view.Timeout,
		MaxConcurrent: view.MaxConcurrent,
	}
	if s.Registry != nil {
		if _, ok := s.Registry.Get(subagent.ToolName); ok {
			subagent.ApplyToAgent(a, limits.Enabled)
		}
	}
	return limits
}

func validateSubagentView(v *config.SubagentView) error {
	if v.MaxTurns <= 0 {
		v.MaxTurns = config.DefaultSubagentMaxTurns
	}
	if v.MaxTurns > maxSubagentTurns {
		return &ArgError{Field: "max_turns", Message: fmt.Sprintf("must be <= %d", maxSubagentTurns)}
	}
	v.MaxDepth = config.DefaultSubagentMaxDepth
	if v.Timeout < 0 {
		v.Timeout = 0
	}
	if v.Timeout > maxSubagentTimeout {
		return &ArgError{Field: "timeout", Message: fmt.Sprintf("must be <= %d", maxSubagentTimeout)}
	}
	if v.MaxConcurrent <= 0 {
		v.MaxConcurrent = config.DefaultSubagentMaxConcurrent
	}
	if v.MaxConcurrent > maxSubagentConcurrent {
		return &ArgError{Field: "max_concurrent", Message: fmt.Sprintf("must be <= %d", maxSubagentConcurrent)}
	}
	return nil
}
