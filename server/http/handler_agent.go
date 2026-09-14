package httpapi

import (
	"errors"
	"io"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"

	"github.com/teexue/nexa/core/agent"
	"github.com/teexue/nexa/core/service"
)

// AgentListItem is the HTTP DTO for GET /v1/agents.
type AgentListItem struct {
	ID            string   `json:"id"`
	Name          string   `json:"name"`
	Provider      string   `json:"provider"`
	Model         string   `json:"model"`
	Tools         []string `json:"tools"`
	MaxTurns      int      `json:"max_turns"`
	ContextWindow int      `json:"context_window,omitempty"`
}

// AgentDetail is the HTTP DTO for GET /v1/agents/:id.
type AgentDetail struct {
	ID            string             `json:"id"`
	Name          string             `json:"name"`
	Provider      string             `json:"provider"`
	Model         string             `json:"model"`
	SystemPrompt  string             `json:"system_prompt"`
	Tools         []string           `json:"tools"`
	MaxTurns      int                `json:"max_turns"`
	MaxTokens     int                `json:"max_tokens"`
	ToolExecution *toolExecutionJSON `json:"tool_execution,omitempty"`
	Permissions   *permissionsJSON   `json:"permissions,omitempty"`
	MCPServers    []mcpServerJSON    `json:"mcp_servers,omitempty"`
	Knowledge     *knowledgeJSON     `json:"knowledge,omitempty"`
	Optimize      *optimizeJSON      `json:"optimize,omitempty"`
	Compaction    *compactionJSON    `json:"compaction,omitempty"`
}

func (s *Server) handleAgents(c *gin.Context) {
	summaries := s.svc.ListAgents()
	items := make([]AgentListItem, len(summaries))
	for i, a := range summaries {
		items[i] = AgentListItem{
			ID:            a.ID,
			Name:          a.Name,
			Provider:      a.Provider,
			Model:         a.Model,
			Tools:         a.Tools,
			MaxTurns:      a.MaxTurns,
			ContextWindow: a.ContextWindow,
		}
	}
	c.JSON(http.StatusOK, items)
}

func (s *Server) handleAgentGet(c *gin.Context) {
	id := c.Param("id")
	a, err := s.svc.GetAgent(id)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			respondError(c, http.StatusNotFound, "not_found", "api.error.agent_not_found")
			return
		}
		respondErrorDetails(c, errorDetails{Status: http.StatusBadRequest, Code: "agent_error", MsgKey: "api.error.agent_error", Details: err.Error()})
		return
	}

	c.JSON(http.StatusOK, agentDetailFrom(a))
}

func (s *Server) handleAgentCreate(c *gin.Context) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		respondErrorDetails(c, errorDetails{Status: http.StatusBadRequest, Code: "invalid_request", MsgKey: "api.error.invalid_request", Details: err.Error()})
		return
	}
	a, err := s.svc.CreateAgent(body)
	if err != nil {
		respondErrorDetails(c, errorDetails{Status: http.StatusBadRequest, Code: "save_error", MsgKey: "api.error.save_error", Details: err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"status": "ok", "id": a.ID, "name": a.Name})
}

func (s *Server) handleAgentPut(c *gin.Context) {
	id := c.Param("id")
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		respondErrorDetails(c, errorDetails{Status: http.StatusBadRequest, Code: "invalid_request", MsgKey: "api.error.invalid_request", Details: err.Error()})
		return
	}

	if err := s.svc.SaveAgent(id, body); err != nil {
		respondErrorDetails(c, errorDetails{Status: http.StatusBadRequest, Code: "save_error", MsgKey: "api.error.save_error", Details: err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok", "id": service.NormalizeAgentName(id)})
}

func (s *Server) handleAgentDelete(c *gin.Context) {
	id := c.Param("id")
	if err := s.svc.DeleteAgent(id); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			respondError(c, http.StatusNotFound, "not_found", "api.error.agent_not_found")
			return
		}
		respondErrorDetails(c, errorDetails{Status: http.StatusInternalServerError, Code: "delete_error", MsgKey: "api.error.delete_error", Details: err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok", "id": service.NormalizeAgentName(id)})
}

func (s *Server) handleAgentValidate(c *gin.Context) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid_request", "api.error.cannot_read_body")
		return
	}

	a, err := agent.LoadFromBytes(body)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"valid":   false,
			"message": err.Error(),
		})
		return
	}

	if err := s.registry.ValidateTools(a.Tools); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"valid":   false,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"valid": true,
		"id":    a.ID,
		"name":  a.Name,
	})
}
