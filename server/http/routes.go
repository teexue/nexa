package httpapi

import (
	"github.com/gin-gonic/gin"

	"github.com/teexue/nexa/core/auth"
)

// mountAPIRoutes registers /v1 routes on r.
// requireScope constrains API key identities only; password sessions
// (member/admin) pass all scope checks. Admin-only routes use requireAdmin.
func (s *Server) mountAPIRoutes(r *gin.Engine) {
	v1 := r.Group("/v1", s.authMiddleware())
	s.mountAuthRoutes(v1)
	s.mountAdminRoutes(v1)
	s.mountAgentRoutes(v1)
	s.mountMCPRoutes(v1)
	s.mountKnowledgeRoutes(v1)
	s.mountProviderRoutes(v1)
	s.mountSkillRoutes(v1)
	v1.GET("/fs/list", requireScope(auth.ScopeFS), s.handleFSList)
	v1.POST("/fs/mkdir", requireScope(auth.ScopeFS), s.handleFSMkdir)
	s.mountSessionRoutes(v1)
	s.mountKanbanRoutes(v1)
}

func (s *Server) mountAuthRoutes(v1 *gin.RouterGroup) {
	v1.GET("/auth/keys", requireAdmin(), s.handleAuthKeysList)
	v1.POST("/auth/keys", requireAdmin(), s.handleAuthKeysCreate)
	v1.PATCH("/auth/keys/:id", requireAdmin(), s.handleAuthKeysPatch)
	v1.DELETE("/auth/keys/:id", requireAdmin(), s.handleAuthKeysDelete)
	v1.GET("/auth/me", s.handleAuthMe)
}

func (s *Server) mountAdminRoutes(v1 *gin.RouterGroup) {
	admin := v1.Group("", requireAdmin())
	admin.GET("/admin/users", s.handleAdminUsersList)
	admin.POST("/admin/users", s.handleAdminUserCreate)
	admin.PATCH("/admin/users/:id", s.handleAdminUserPatch)
	admin.DELETE("/admin/users/:id", s.handleAdminUserDelete)
	admin.GET("/admin/settings/registration", s.handleAdminRegistrationGet)
	admin.PUT("/admin/settings/registration", s.handleAdminRegistrationPut)
	admin.POST("/providers", s.handleProviderUpsert)
	admin.DELETE("/providers/:name", s.handleProviderDelete)
	admin.POST("/providers/models", s.handleProviderModelsTest)
	admin.POST("/providers/models/detail", s.handleProviderModelDetailTest)
	admin.PUT("/embedding", s.handleEmbeddingPut)
	admin.PUT("/subagent", s.handleSubagentPut)
	admin.PUT("/shell", s.handleShellPut)
	if s.requestLogger != nil {
		admin.GET("/audit/requests", s.handleAuditRequests)
		admin.GET("/audit/requests/detail", s.handleAuditRequestDetail)
		v1.GET("/usage/summary", s.handleUsageSummary)
	}
}

func (s *Server) mountAgentRoutes(v1 *gin.RouterGroup) {
	agents := v1.Group("", requireScope(auth.ScopeAgents))
	agents.POST("/agents/run", s.handleRun)
	agents.POST("/agents/approve", s.handleApprove)
	agents.POST("/agents/optimize", s.handleOptimizePrompt)
	agents.GET("/tools", s.handleTools)
	agents.GET("/agents", s.handleAgents)
	agents.POST("/agents", s.handleAgentCreate)
	agents.GET("/agents/:id", s.handleAgentGet)
	agents.PUT("/agents/:id", s.handleAgentPut)
	agents.DELETE("/agents/:id", s.handleAgentDelete)
	agents.POST("/agents/validate", s.handleAgentValidate)
	agents.GET("/events", s.handleEvents)
	agents.GET("/background", s.handleBackgroundGet)
	agents.HEAD("/background", s.handleBackgroundGet)
	agents.POST("/background", s.handleBackgroundUpload)
	agents.DELETE("/background", s.handleBackgroundDelete)
}

func (s *Server) mountMCPRoutes(v1 *gin.RouterGroup) {
	mcp := v1.Group("", requireScope(auth.ScopeMCP))
	mcp.GET("/mcp", s.handleMCPList)
	mcp.POST("/mcp/global", s.handleMCPGlobalUpsert)
	mcp.DELETE("/mcp/global/:name", s.handleMCPGlobalDelete)
}

func (s *Server) mountKnowledgeRoutes(v1 *gin.RouterGroup) {
	knowledge := v1.Group("", requireScope(auth.ScopeKnowledge))
	knowledge.GET("/knowledge", s.handleKnowledgeList)
	knowledge.POST("/knowledge", s.handleKnowledgeCreate)
	knowledge.POST("/knowledge/search", s.handleKnowledgeSearch)
	knowledge.GET("/knowledge/:id", s.handleKnowledgeGet)
	knowledge.PATCH("/knowledge/:id", s.handleKnowledgeUpdate)
	knowledge.DELETE("/knowledge/:id", s.handleKnowledgeDelete)
	knowledge.GET("/knowledge/:id/documents", s.handleKnowledgeDocsList)
	knowledge.POST("/knowledge/:id/documents", s.handleKnowledgeDocUpload)
	knowledge.DELETE("/knowledge/:id/documents/:docId", s.handleKnowledgeDocDelete)
	knowledge.POST("/knowledge/:id/reindex", s.handleKnowledgeReindex)
}

func (s *Server) mountProviderRoutes(v1 *gin.RouterGroup) {
	providers := v1.Group("", requireScope(auth.ScopeProviders))
	providers.GET("/vendors", s.handleVendors)
	providers.GET("/providers", s.handleProvidersList)
	providers.GET("/providers/:name/models", s.handleProviderModels)
	providers.GET("/providers/:name/models/:model/detail", s.handleProviderModelDetail)
	providers.GET("/embedding", s.handleEmbeddingGet)
	providers.GET("/embedding/vendors", s.handleEmbeddingVendors)
	providers.GET("/subagent", s.handleSubagentGet)
	providers.GET("/shell", s.handleShellGet)
}

func (s *Server) mountSkillRoutes(v1 *gin.RouterGroup) {
	skills := v1.Group("", requireScope(auth.ScopeSkills))
	skills.GET("/skills", s.handleSkillsList)
	skills.POST("/skills", s.handleSkillCreate)
	skills.POST("/skills/install", s.handleSkillsInstall)
	skills.GET("/skills/:name", s.handleSkillGet)
	skills.PUT("/skills/:name", s.handleSkillUpdate)
	skills.DELETE("/skills/:name", s.handleSkillDelete)
}

func (s *Server) mountSessionRoutes(v1 *gin.RouterGroup) {
	if s.store == nil {
		return
	}
	sessions := v1.Group("", requireScope(auth.ScopeSessions))
	sessions.GET("/sessions", s.handleSessionsList)
	sessions.GET("/sessions/:id/events", s.handleSessionEvents)
	sessions.POST("/sessions/:id/abort", s.handleSessionAbort)
	sessions.GET("/sessions/:id", s.handleSessionsGet)
	sessions.PATCH("/sessions/:id", s.handleSessionPatch)
	sessions.DELETE("/sessions/:id", s.handleSessionsDelete)
}

func (s *Server) mountKanbanRoutes(v1 *gin.RouterGroup) {
	if s.stateDB == nil {
		return
	}
	kanban := v1.Group("", requireScope(auth.ScopeKanban))
	kanban.GET("/kanban", s.handleKanbanList)
	kanban.POST("/kanban", s.handleKanbanCreate)
	kanban.GET("/kanban/:id", s.handleKanbanGet)
	kanban.PATCH("/kanban/:id", s.handleKanbanPatch)
	kanban.DELETE("/kanban/:id", s.handleKanbanDelete)
	kanban.POST("/kanban/:id/approve", s.handleKanbanApprove)
	kanban.POST("/kanban/:id/reject", s.handleKanbanReject)
	kanban.POST("/kanban/:id/requeue", s.handleKanbanRequeue)
}
