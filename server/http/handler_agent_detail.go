package httpapi

import (
	"github.com/teexue/nexa/core/agent"
	"github.com/teexue/nexakit/permission"
)

// toolExecutionJSON keeps Mode and MaxParallel because the editor already reads those names.
type toolExecutionJSON struct {
	Mode        string `json:"Mode"`
	MaxParallel int    `json:"MaxParallel"`
}

type permissionsJSON struct {
	AutoApprove []string `json:"auto_approve,omitempty"`
	AlwaysDeny  []string `json:"always_deny,omitempty"`
}

type mcpServerJSON struct {
	Name    string            `json:"name"`
	Type    string            `json:"type"`
	Command string            `json:"command,omitempty"`
	Args    []string          `json:"args,omitempty"`
	Env     map[string]string `json:"env,omitempty"`
	URL     string            `json:"url,omitempty"`
}

type knowledgeJSON struct {
	Bases []string `json:"bases,omitempty"`
	TopK  int      `json:"top_k,omitempty"`
}

type optimizeJSON struct {
	UserPrompt bool `json:"user_prompt,omitempty"`
}

type compactionJSON struct {
	Strategy      string  `json:"strategy,omitempty"`
	ContextWindow int     `json:"context_window,omitempty"`
	TriggerRatio  float64 `json:"trigger_ratio,omitempty"`
	TargetRatio   float64 `json:"target_ratio,omitempty"`
	KeepRecent    int     `json:"keep_recent,omitempty"`
	KeepHead      int     `json:"keep_head,omitempty"`
	MaxMessages   int     `json:"max_messages,omitempty"`
	SummaryModel  string  `json:"summary_model,omitempty"`
}

func agentDetailFrom(a *agent.Agent) AgentDetail {
	return AgentDetail{
		ID: a.ID, Name: a.Name, Provider: a.Provider, Model: a.Model,
		SystemPrompt: a.SystemPrompt, Tools: a.Tools,
		MaxTurns: a.MaxTurns, MaxTokens: a.MaxTokens,
		ToolExecution: toolExecutionFrom(a.ToolExecution),
		Permissions:   permissionsFrom(a.Permissions),
		MCPServers:    mcpServersFrom(a.MCPServers),
		Knowledge:     knowledgeFrom(a.Knowledge),
		Optimize:      optimizeFrom(a.Optimize),
		Compaction:    compactionFrom(a.Compaction),
	}
}

func toolExecutionFrom(in *agent.ToolExecution) *toolExecutionJSON {
	if in == nil {
		return nil
	}
	return &toolExecutionJSON{Mode: in.Mode, MaxParallel: in.MaxParallel}
}

func permissionsFrom(in *permission.Permissions) *permissionsJSON {
	if in == nil {
		return nil
	}
	return &permissionsJSON{AutoApprove: in.AutoApprove, AlwaysDeny: in.AlwaysDeny}
}

func knowledgeFrom(in *agent.KnowledgeConfig) *knowledgeJSON {
	if in == nil {
		return nil
	}
	return &knowledgeJSON{Bases: in.Bases, TopK: in.TopK}
}

func optimizeFrom(in *agent.OptimizeConfig) *optimizeJSON {
	if in == nil {
		return nil
	}
	return &optimizeJSON{UserPrompt: in.UserPrompt}
}

func compactionFrom(in *agent.CompactionConfig) *compactionJSON {
	if in == nil {
		return nil
	}
	return &compactionJSON{
		Strategy: in.Strategy, ContextWindow: in.ContextWindow,
		TriggerRatio: in.TriggerRatio, TargetRatio: in.TargetRatio,
		KeepRecent: in.KeepRecent, KeepHead: in.KeepHead,
		MaxMessages: in.MaxMessages, SummaryModel: in.SummaryModel,
	}
}

func mcpServersFrom(in []agent.MCPServerConfig) []mcpServerJSON {
	if len(in) == 0 {
		return nil
	}
	out := make([]mcpServerJSON, len(in))
	for i, s := range in {
		out[i] = mcpServerJSON{
			Name: s.Name, Type: s.Type, Command: s.Command,
			Args: s.Args, Env: s.Env, URL: s.URL,
		}
	}
	return out
}
