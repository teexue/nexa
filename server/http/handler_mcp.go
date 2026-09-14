package httpapi

import (
	"net/http"
	"path/filepath"

	"github.com/gin-gonic/gin"

	"github.com/teexue/common-agent/core/agent"
	"github.com/teexue/common-agent/core/config"
	"github.com/teexue/nexakit/mcp"
)

// MCPServerInfo is the JSON DTO for MCP server listing.
type MCPServerInfo struct {
	Name    string            `json:"name"`
	Type    string            `json:"type"`
	Command string            `json:"command,omitempty"`
	Args    []string          `json:"args,omitempty"`
	Env     map[string]string `json:"env,omitempty"`
	URL     string            `json:"url,omitempty"`
	Agent   string            `json:"agent"` // agent id/name for agent-scoped servers; "" for global
	Scope   string            `json:"scope"` // "global" | "agent"
}

func (s *Server) handleMCPList(c *gin.Context) {
	home := filepath.Dir(s.agentsDir)

	var servers []MCPServerInfo

	// Global shared servers.
	if global, err := config.LoadGlobalMCP(home); err != nil {
		s.logger.Warn("log.mcp.load_global", "error", err)
	} else {
		for _, m := range global {
			servers = append(servers, MCPServerInfo{
				Name:    m.Name,
				Type:    m.Type,
				Command: m.Command,
				Args:    m.Args,
				Env:     m.Env,
				URL:     m.URL,
				Scope:   "global",
			})
		}
	}

	// Per-agent servers.
	result, err := agent.LoadAll(s.agentsDir)
	if err != nil {
		respondErrorDetails(c, errorDetails{Status: http.StatusInternalServerError, Code: "agent_error", MsgKey: "api.error.agent_error", Details: err.Error()})
		return
	}
	for _, a := range result.Agents {
		for _, mcp := range a.MCPServers {
			servers = append(servers, MCPServerInfo{
				Name:    mcp.Name,
				Type:    mcp.Type,
				Command: mcp.Command,
				Args:    mcp.Args,
				Env:     mcp.Env,
				URL:     mcp.URL,
				Agent:   a.Name,
				Scope:   "agent",
			})
		}
	}

	if servers == nil {
		servers = []MCPServerInfo{}
	}
	c.JSON(http.StatusOK, servers)
}

// MCPServerUpsertRequest is the DTO for POST /v1/mcp/global.
type MCPServerUpsertRequest struct {
	Name    string            `json:"name"`
	Type    string            `json:"type"`
	Command string            `json:"command,omitempty"`
	Args    []string          `json:"args,omitempty"`
	Env     map[string]string `json:"env,omitempty"`
	URL     string            `json:"url,omitempty"`
}

func (s *Server) handleMCPGlobalUpsert(c *gin.Context) {
	var req MCPServerUpsertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondErrorDetails(c, errorDetails{Status: http.StatusBadRequest, Code: "invalid_json", MsgKey: "api.error.invalid_json", Details: err.Error()})
		return
	}
	if req.Name == "" {
		respondError(c, http.StatusBadRequest, "invalid_request", "api.error.invalid_request")
		return
	}

	srv := mcp.ServerConfig{
		Name:    req.Name,
		Type:    req.Type,
		Command: req.Command,
		Args:    req.Args,
		Env:     req.Env,
		URL:     req.URL,
	}
	home := filepath.Dir(s.agentsDir)
	if err := config.UpsertGlobalMCP(home, srv); err != nil {
		respondErrorDetails(c, errorDetails{Status: http.StatusBadRequest, Code: "mcp_error", MsgKey: "api.error.mcp_error", Details: err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true, "name": req.Name})
}

func (s *Server) handleMCPGlobalDelete(c *gin.Context) {
	name := c.Param("name")
	if name == "" {
		respondError(c, http.StatusBadRequest, "invalid_request", "api.error.invalid_request")
		return
	}
	home := filepath.Dir(s.agentsDir)
	if err := config.DeleteGlobalMCP(home, name); err != nil {
		respondErrorDetails(c, errorDetails{Status: http.StatusNotFound, Code: "not_found", MsgKey: "api.error.mcp_not_found", Details: err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"deleted": name})
}
