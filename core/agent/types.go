package agent

import kitagent "github.com/teexue/nexakit/agent"

// Agent is the runtime agent config. YAML loading lives in this package.
type Agent = kitagent.Agent

// ToolExecution is the tool execution strategy on an agent.
type ToolExecution = kitagent.ToolExecution

// MCPServerConfig is an MCP server declared on an agent.
type MCPServerConfig = kitagent.MCPServerConfig

// CompactionConfig is the context-window compaction policy.
type CompactionConfig = kitagent.CompactionConfig

// KnowledgeConfig scopes RAG retrieval for an agent.
type KnowledgeConfig = kitagent.KnowledgeConfig

// OptimizeConfig enables prompt optimization before a run.
type OptimizeConfig = kitagent.OptimizeConfig

// NewID generates a stable agent id.
func NewID() string { return kitagent.NewID() }
