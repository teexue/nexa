package agent

import (
	"fmt"

	"github.com/teexue/nexakit/compaction"
	"github.com/teexue/nexakit/permission"
	"gopkg.in/yaml.v3"
)

// fileAgent is the on-disk agent document.
type fileAgent struct {
	Version       int                `yaml:"version"`
	ID            string             `yaml:"id,omitempty"`
	Name          string             `yaml:"name"`
	Provider      string             `yaml:"provider"`
	SystemPrompt  string             `yaml:"system_prompt"`
	Tools         []string           `yaml:"tools"`
	Skills        []string           `yaml:"skills,omitempty"`
	Model         string             `yaml:"model"`
	MaxTurns      int                `yaml:"max_turns"`
	MaxTokens     int                `yaml:"max_tokens"`
	ToolExecution *fileToolExecution `yaml:"tool_execution,omitempty"`
	Permissions   *filePermissions   `yaml:"permissions,omitempty"`
	MCPServers    []fileMCPServer    `yaml:"mcp_servers,omitempty"`
	Compaction    *fileCompaction    `yaml:"compaction,omitempty"`
	Knowledge     *fileKnowledge     `yaml:"knowledge,omitempty"`
	Optimize      *fileOptimize      `yaml:"optimize,omitempty"`
}

type fileToolExecution struct {
	Mode        string `yaml:"mode"`
	MaxParallel int    `yaml:"max_parallel"`
}

type fileMCPServer struct {
	Name    string            `yaml:"name"`
	Type    string            `yaml:"type"`
	Command string            `yaml:"command,omitempty"`
	Args    []string          `yaml:"args,omitempty"`
	Env     map[string]string `yaml:"env,omitempty"`
	URL     string            `yaml:"url,omitempty"`
}

type fileCompaction struct {
	Strategy      string  `yaml:"strategy"`
	ContextWindow int     `yaml:"context_window,omitempty"`
	TriggerRatio  float64 `yaml:"trigger_ratio,omitempty"`
	TargetRatio   float64 `yaml:"target_ratio,omitempty"`
	KeepRecent    int     `yaml:"keep_recent"`
	KeepHead      int     `yaml:"keep_head,omitempty"`
	MaxMessages   int     `yaml:"max_messages,omitempty"`
	SummaryModel  string  `yaml:"summary_model,omitempty"`
}

type fileKnowledge struct {
	Bases []string `yaml:"bases,omitempty"`
	TopK  int      `yaml:"top_k,omitempty"`
}

type fileOptimize struct {
	UserPrompt bool `yaml:"user_prompt,omitempty"`
}

type filePermissions struct {
	AutoApprove []string `yaml:"auto_approve,omitempty"`
	AlwaysDeny  []string `yaml:"always_deny,omitempty"`
}

func parseFile(data []byte) (*Agent, error) {
	var doc fileAgent
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("parse agent: %w", err)
	}
	a := doc.toAgent()
	if err := a.Validate(); err != nil {
		return nil, fmt.Errorf("validate agent: %w", err)
	}
	return a, nil
}

// Marshal encodes an agent as the product YAML document.
// Runtime-only fields are not written.
func Marshal(a *Agent) ([]byte, error) {
	if a == nil {
		return nil, fmt.Errorf("agent is required")
	}
	return yaml.Marshal(fromAgent(a))
}

func (doc fileAgent) toAgent() *Agent {
	return &Agent{
		Version: doc.Version, ID: doc.ID, Name: doc.Name, Provider: doc.Provider,
		SystemPrompt: doc.SystemPrompt, Tools: doc.Tools, Skills: doc.Skills,
		Model: doc.Model, MaxTurns: doc.MaxTurns, MaxTokens: doc.MaxTokens,
		ToolExecution: toolExecFromFile(doc.ToolExecution),
		Permissions:   permsFromFile(doc.Permissions),
		MCPServers:    mcpFromFile(doc.MCPServers),
		Compaction:    compactionFromFile(doc.Compaction),
		Knowledge:     knowledgeFromFile(doc.Knowledge),
		Optimize:      optimizeFromFile(doc.Optimize),
	}
}

func fromAgent(a *Agent) fileAgent {
	return fileAgent{
		Version: a.Version, ID: a.ID, Name: a.Name, Provider: a.Provider,
		SystemPrompt: a.SystemPrompt, Tools: a.Tools, Skills: a.Skills,
		Model: a.Model, MaxTurns: a.MaxTurns, MaxTokens: a.MaxTokens,
		ToolExecution: toolExecToFile(a.ToolExecution),
		Permissions:   permsToFile(a.Permissions),
		MCPServers:    mcpToFile(a.MCPServers),
		Compaction:    compactionToFile(a.Compaction),
		Knowledge:     knowledgeToFile(a.Knowledge),
		Optimize:      optimizeToFile(a.Optimize),
	}
}

func toolExecFromFile(in *fileToolExecution) *ToolExecution {
	if in == nil {
		return nil
	}
	return &ToolExecution{Mode: in.Mode, MaxParallel: in.MaxParallel}
}

func toolExecToFile(in *ToolExecution) *fileToolExecution {
	if in == nil {
		return nil
	}
	return &fileToolExecution{Mode: in.Mode, MaxParallel: in.MaxParallel}
}

func permsFromFile(in *filePermissions) *permission.Permissions {
	if in == nil {
		return nil
	}
	return &permission.Permissions{AutoApprove: in.AutoApprove, AlwaysDeny: in.AlwaysDeny}
}

func permsToFile(in *permission.Permissions) *filePermissions {
	if in == nil {
		return nil
	}
	return &filePermissions{AutoApprove: in.AutoApprove, AlwaysDeny: in.AlwaysDeny}
}

func knowledgeFromFile(in *fileKnowledge) *KnowledgeConfig {
	if in == nil {
		return nil
	}
	return &KnowledgeConfig{Bases: in.Bases, TopK: in.TopK}
}

func knowledgeToFile(in *KnowledgeConfig) *fileKnowledge {
	if in == nil {
		return nil
	}
	return &fileKnowledge{Bases: in.Bases, TopK: in.TopK}
}

func optimizeFromFile(in *fileOptimize) *OptimizeConfig {
	if in == nil {
		return nil
	}
	return &OptimizeConfig{UserPrompt: in.UserPrompt}
}

func optimizeToFile(in *OptimizeConfig) *fileOptimize {
	if in == nil {
		return nil
	}
	return &fileOptimize{UserPrompt: in.UserPrompt}
}

func compactionFromFile(in *fileCompaction) *CompactionConfig {
	if in == nil {
		return nil
	}
	return &CompactionConfig{
		Strategy: compaction.Strategy(in.Strategy), ContextWindow: in.ContextWindow,
		TriggerRatio: in.TriggerRatio, TargetRatio: in.TargetRatio,
		KeepRecent: in.KeepRecent, KeepHead: in.KeepHead,
		MaxMessages: in.MaxMessages, SummaryModel: in.SummaryModel,
	}
}

func compactionToFile(in *CompactionConfig) *fileCompaction {
	if in == nil {
		return nil
	}
	return &fileCompaction{
		Strategy: string(in.Strategy), ContextWindow: in.ContextWindow,
		TriggerRatio: in.TriggerRatio, TargetRatio: in.TargetRatio,
		KeepRecent: in.KeepRecent, KeepHead: in.KeepHead,
		MaxMessages: in.MaxMessages, SummaryModel: in.SummaryModel,
	}
}

func mcpFromFile(in []fileMCPServer) []MCPServerConfig {
	if in == nil {
		return nil
	}
	out := make([]MCPServerConfig, len(in))
	for i, s := range in {
		out[i] = MCPServerConfig{
			Name: s.Name, Type: s.Type, Command: s.Command,
			Args: s.Args, Env: s.Env, URL: s.URL,
		}
	}
	return out
}

func mcpToFile(in []MCPServerConfig) []fileMCPServer {
	if in == nil {
		return nil
	}
	out := make([]fileMCPServer, len(in))
	for i, s := range in {
		out[i] = fileMCPServer{
			Name: s.Name, Type: s.Type, Command: s.Command,
			Args: s.Args, Env: s.Env, URL: s.URL,
		}
	}
	return out
}
