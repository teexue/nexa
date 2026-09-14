package config

import "github.com/teexue/nexa/core/store"

const (
	// DefaultSubagentMaxTurns is the child-run turn cap when unset.
	DefaultSubagentMaxTurns = 5
	// DefaultSubagentMaxDepth is the fixed nesting cap (not user-configurable).
	// 1 = the main agent may spawn children; those children cannot spawn.
	DefaultSubagentMaxDepth = 1
	// DefaultSubagentMaxConcurrent caps simultaneous child runs (queue the rest).
	DefaultSubagentMaxConcurrent = 2
)

// SubagentSettings is the persisted global sub-agent section.
type SubagentSettings struct {
	Enabled  *bool `yaml:"enabled,omitempty" json:"enabled,omitempty"`
	MaxTurns int   `yaml:"max_turns,omitempty" json:"max_turns,omitempty"`
	// MaxDepth is ignored; nesting is always DefaultSubagentMaxDepth.
	MaxDepth int `yaml:"max_depth,omitempty" json:"max_depth,omitempty"`
	Timeout  int `yaml:"timeout,omitempty" json:"timeout,omitempty"`
	// MaxConcurrent caps simultaneous child runs (independent of tool max_parallel).
	MaxConcurrent int `yaml:"max_concurrent,omitempty" json:"max_concurrent,omitempty"`
}

// SubagentView is the normalized public DTO (always filled with defaults).
type SubagentView struct {
	Enabled  bool `json:"enabled"`
	MaxTurns int  `json:"max_turns"`
	// MaxDepth is always DefaultSubagentMaxDepth; PUT cannot change it.
	MaxDepth      int `json:"max_depth"`
	Timeout       int `json:"timeout"`
	MaxConcurrent int `json:"max_concurrent"`
}

// SubagentView returns global sub-agent limits with defaults applied.
func (s Settings) SubagentView() SubagentView {
	v := SubagentView{
		Enabled:       true,
		MaxTurns:      DefaultSubagentMaxTurns,
		MaxDepth:      DefaultSubagentMaxDepth,
		MaxConcurrent: DefaultSubagentMaxConcurrent,
	}
	if s.Subagent == nil {
		return v
	}
	if s.Subagent.Enabled != nil {
		v.Enabled = *s.Subagent.Enabled
	}
	if s.Subagent.MaxTurns > 0 {
		v.MaxTurns = s.Subagent.MaxTurns
	}
	if s.Subagent.Timeout > 0 {
		v.Timeout = s.Subagent.Timeout
	}
	if s.Subagent.MaxConcurrent > 0 {
		v.MaxConcurrent = s.Subagent.MaxConcurrent
	}
	return v
}

// Persistable converts a view into the YAML/JSON settings blob.
func (v SubagentView) Persistable() *SubagentSettings {
	enabled := v.Enabled
	return &SubagentSettings{
		Enabled:       &enabled,
		MaxTurns:      v.MaxTurns,
		Timeout:       v.Timeout,
		MaxConcurrent: v.MaxConcurrent,
	}
}

func fromStoreSubagent(s *store.SubagentSettings) *SubagentSettings {
	if s == nil {
		return nil
	}
	return &SubagentSettings{
		Enabled: s.Enabled, MaxTurns: s.MaxTurns, MaxDepth: s.MaxDepth,
		Timeout: s.Timeout, MaxConcurrent: s.MaxConcurrent,
	}
}

func toStoreSubagent(s *SubagentSettings) *store.SubagentSettings {
	if s == nil {
		return nil
	}
	return &store.SubagentSettings{
		Enabled: s.Enabled, MaxTurns: s.MaxTurns, MaxDepth: s.MaxDepth,
		Timeout: s.Timeout, MaxConcurrent: s.MaxConcurrent,
	}
}
