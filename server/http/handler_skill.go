package httpapi

import (
	"errors"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"

	"github.com/teexue/nexa/core/config"
	"github.com/teexue/nexa/core/skill"
)

// SkillInfo is the JSON DTO for skill listing.
type SkillInfo struct {
	Name        string   `json:"name"`
	Version     string   `json:"version"`
	Description string   `json:"description"`
	Format      string   `json:"format"`
	Scope       string   `json:"scope"` // "global" | "agent"
	Agent       string   `json:"agent,omitempty"`
	Author      string   `json:"author,omitempty"`
	Tools       []string `json:"tools"`
}

// SkillDetail is the JSON DTO for a single skill, including instructions.
type SkillDetail struct {
	SkillInfo
	Body          string            `json:"body"`
	License       string            `json:"license,omitempty"`
	Compatibility string            `json:"compatibility,omitempty"`
	Metadata      map[string]string `json:"metadata,omitempty"`
	AllowedTools  string            `json:"allowed_tools,omitempty"`
}

// SkillUpsertRequest is the DTO for POST/PUT /v1/skills.
type SkillUpsertRequest struct {
	Name          string            `json:"name"`
	Description   string            `json:"description"`
	Body          string            `json:"body"`
	License       string            `json:"license,omitempty"`
	Compatibility string            `json:"compatibility,omitempty"`
	Metadata      map[string]string `json:"metadata,omitempty"`
	AllowedTools  string            `json:"allowed_tools,omitempty"`
	Scope         string            `json:"scope"` // "global" (default) | "agent"
	Agent         string            `json:"agent,omitempty"`
}

// SkillInstallRequest is the DTO for POST /v1/skills/install.
type SkillInstallRequest struct {
	URL       string `json:"url"`
	Scope     string `json:"scope"` // "global" (default) | "agent"
	Agent     string `json:"agent,omitempty"`
	Overwrite bool   `json:"overwrite,omitempty"`
}

// skillInfo maps a loaded skill to its listing DTO.
func skillInfo(sk *skill.Skill) SkillInfo {
	var author string
	if sk.MDManifest != nil {
		author = sk.MDManifest.Frontmatter.Metadata["author"]
	} else if sk.LegacyManifest != nil {
		author = sk.LegacyManifest.Author
	}
	tools := sk.ToolNames()
	if tools == nil {
		tools = []string{}
	}
	scope := sk.Scope
	if scope == "" {
		scope = skill.ScopeGlobal
	}
	return SkillInfo{
		Name:        sk.Name,
		Version:     sk.Version,
		Description: sk.Description,
		Format:      sk.Format,
		Scope:       scope,
		Agent:       sk.Agent,
		Author:      author,
		Tools:       tools,
	}
}

// normalizeSkillScope validates the requested scope, defaulting to global.
func normalizeSkillScope(scope string) (string, bool) {
	switch scope {
	case "", skill.ScopeGlobal:
		return skill.ScopeGlobal, true
	case skill.ScopeAgent:
		return skill.ScopeAgent, true
	default:
		return "", false
	}
}

// skillDirFor resolves the target directory for a scoped skill name.
func (s *Server) skillDirFor(scope, agentName, name string) (string, error) {
	if scope == skill.ScopeAgent {
		if agentName == "" {
			return "", errors.New("agent is required for agent-scoped skill")
		}
		return filepath.Join(config.AgentSkillsDir(s.home, agentName), name), nil
	}
	return filepath.Join(config.SkillsDir(s.home), name), nil
}

// handleSkillsList lists skills; ?agent= narrows to that agent's effective set.
func (s *Server) handleSkillsList(c *gin.Context) {
	var skills []*skill.Skill
	var err error
	if agentName := c.Query("agent"); agentName != "" {
		skills, err = skill.LoadScoped(config.SkillsDir(s.home), config.AgentSkillsDir(s.home, agentName), agentName)
	} else {
		skills, err = skill.LoadAllScoped(config.SkillsDir(s.home), config.AgentSkillsRoot(s.home))
	}
	if err != nil {
		s.logger.Warn("log.skill.load_partial", "error", err)
	}

	result := make([]SkillInfo, 0, len(skills))
	for _, sk := range skills {
		result = append(result, skillInfo(sk))
	}
	c.JSON(http.StatusOK, result)
}

// handleSkillGet returns one skill with its full instructions.
func (s *Server) handleSkillGet(c *gin.Context) {
	scope, ok := normalizeSkillScope(c.Query("scope"))
	if !ok {
		respondError(c, http.StatusBadRequest, "invalid_request", "api.error.invalid_request")
		return
	}
	dir, err := s.skillDirFor(scope, c.Query("agent"), c.Param("name"))
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid_request", "api.error.invalid_request")
		return
	}
	sk, err := skill.Load(dir)
	if err != nil {
		respondErrorDetails(c, errorDetails{Status: http.StatusNotFound, Code: "skill_not_found", MsgKey: "api.error.skill_not_found", Details: err.Error()})
		return
	}

	detail := SkillDetail{SkillInfo: skillInfo(sk), Body: sk.Body()}
	if sk.MDManifest != nil {
		fm := sk.MDManifest.Frontmatter
		detail.License = fm.License
		detail.Compatibility = fm.Compatibility
		detail.Metadata = fm.Metadata
		detail.AllowedTools = fm.AllowedTools
	}
	c.JSON(http.StatusOK, detail)
}

// handleSkillCreate creates a SKILL.md skill (Agent Skills standard).
func (s *Server) handleSkillCreate(c *gin.Context) {
	var req SkillUpsertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondErrorDetails(c, errorDetails{Status: http.StatusBadRequest, Code: "invalid_json", MsgKey: "api.error.invalid_json", Details: err.Error()})
		return
	}
	scope, ok := normalizeSkillScope(req.Scope)
	if !ok {
		respondError(c, http.StatusBadRequest, "invalid_request", "api.error.invalid_request")
		return
	}
	dir, err := s.skillDirFor(scope, req.Agent, req.Name)
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid_request", "api.error.invalid_request")
		return
	}
	if _, err := os.Stat(filepath.Join(dir, "SKILL.md")); err == nil {
		respondError(c, http.StatusConflict, "skill_exists", "api.error.skill_exists")
		return
	}

	fm := frontmatterFromRequest(req)
	if err := skill.WriteSkill(dir, fm, req.Body); err != nil {
		respondErrorDetails(c, errorDetails{Status: http.StatusBadRequest, Code: "skill_error", MsgKey: "api.error.skill_error", Details: err.Error()})
		return
	}
	s.respondSkillWritten(c, http.StatusCreated, dir)
}

// handleSkillUpdate rewrites an existing skill's SKILL.md.
func (s *Server) handleSkillUpdate(c *gin.Context) {
	var req SkillUpsertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondErrorDetails(c, errorDetails{Status: http.StatusBadRequest, Code: "invalid_json", MsgKey: "api.error.invalid_json", Details: err.Error()})
		return
	}
	scope, ok := normalizeSkillScope(c.DefaultQuery("scope", req.Scope))
	if !ok {
		respondError(c, http.StatusBadRequest, "invalid_request", "api.error.invalid_request")
		return
	}
	agentName := c.Query("agent")
	if agentName == "" {
		agentName = req.Agent
	}
	dir, err := s.skillDirFor(scope, agentName, c.Param("name"))
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid_request", "api.error.invalid_request")
		return
	}
	if _, err := os.Stat(filepath.Join(dir, "SKILL.md")); err != nil {
		respondError(c, http.StatusNotFound, "skill_not_found", "api.error.skill_not_found")
		return
	}

	fm := frontmatterFromRequest(req)
	fm.Name = c.Param("name")
	if err := skill.WriteSkill(dir, fm, req.Body); err != nil {
		respondErrorDetails(c, errorDetails{Status: http.StatusBadRequest, Code: "skill_error", MsgKey: "api.error.skill_error", Details: err.Error()})
		return
	}
	s.respondSkillWritten(c, http.StatusOK, dir)
}

// handleSkillDelete removes a skill directory.
func (s *Server) handleSkillDelete(c *gin.Context) {
	scope, ok := normalizeSkillScope(c.Query("scope"))
	if !ok {
		respondError(c, http.StatusBadRequest, "invalid_request", "api.error.invalid_request")
		return
	}
	dir, err := s.skillDirFor(scope, c.Query("agent"), c.Param("name"))
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid_request", "api.error.invalid_request")
		return
	}
	if err := skill.RemoveSkill(dir); err != nil {
		respondErrorDetails(c, errorDetails{Status: http.StatusNotFound, Code: "skill_not_found", MsgKey: "api.error.skill_not_found", Details: err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"deleted": c.Param("name")})
}

// handleSkillsInstall installs skills from a GitHub repo or a SKILL.md URL.
func (s *Server) handleSkillsInstall(c *gin.Context) {
	var req SkillInstallRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondErrorDetails(c, errorDetails{Status: http.StatusBadRequest, Code: "invalid_json", MsgKey: "api.error.invalid_json", Details: err.Error()})
		return
	}
	if req.URL == "" {
		respondError(c, http.StatusBadRequest, "invalid_request", "api.error.invalid_request")
		return
	}
	scope, ok := normalizeSkillScope(req.Scope)
	if !ok {
		respondError(c, http.StatusBadRequest, "invalid_request", "api.error.invalid_request")
		return
	}
	destRoot := config.SkillsDir(s.home)
	if scope == skill.ScopeAgent {
		if req.Agent == "" {
			respondError(c, http.StatusBadRequest, "invalid_request", "api.error.invalid_request")
			return
		}
		destRoot = config.AgentSkillsDir(s.home, req.Agent)
	}

	installed, err := skill.Install(c.Request.Context(), req.URL, destRoot, req.Overwrite)
	if err != nil {
		respondErrorDetails(c, errorDetails{Status: http.StatusBadRequest, Code: "skill_error", MsgKey: "api.error.skill_error", Details: err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"installed": installed})
}

// frontmatterFromRequest builds validated frontmatter from an upsert request.
func frontmatterFromRequest(req SkillUpsertRequest) *skill.SkillFrontmatter {
	return &skill.SkillFrontmatter{
		Name:          req.Name,
		Description:   req.Description,
		License:       req.License,
		Compatibility: req.Compatibility,
		Metadata:      req.Metadata,
		AllowedTools:  req.AllowedTools,
	}
}

// respondSkillWritten loads the just-written skill and returns its detail.
func (s *Server) respondSkillWritten(c *gin.Context, status int, dir string) {
	sk, err := skill.Load(dir)
	if err != nil {
		respondErrorDetails(c, errorDetails{Status: http.StatusInternalServerError, Code: "skill_error", MsgKey: "api.error.skill_error", Details: err.Error()})
		return
	}
	c.JSON(status, gin.H{"ok": true, "skill": skillInfo(sk)})
}
