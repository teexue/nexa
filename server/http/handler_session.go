package httpapi

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/teexue/nexakit/session"

	"github.com/teexue/nexa/core/audit"
	"github.com/teexue/nexa/core/store"
)

func (s *Server) handleSessionsList(c *gin.Context) {
	if !s.requireSessionStore(c) {
		return
	}
	userID := identityFromGin(c).UserID
	metas, err := s.svc.ListSessions(userID)
	if err != nil {
		respondErrorDetails(c, errorDetails{Status: http.StatusInternalServerError, Code: "session_error", MsgKey: "api.error.session_error", Details: err.Error()})
		return
	}
	if metas == nil {
		metas = []session.Meta{}
	}
	c.JSON(http.StatusOK, metas)
}

func (s *Server) handleSessionsGet(c *gin.Context) {
	sess, ok := s.loadSessionOrRespond(c)
	if !ok {
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":         sess.ID,
		"user_id":    sess.UserID,
		"agent":      sess.Agent,
		"title":      sess.GetTitle(),
		"messages":   sess.GetMessages(),
		"metadata":   sess.GetMetadata(),
		"created_at": sess.CreatedAt,
		"updated_at": sess.UpdatedAt,
	})
}

// SessionPatchRequest is the HTTP DTO for PATCH /v1/sessions/:id. Fields left
// null are unchanged.
type SessionPatchRequest struct {
	WorkDir *string `json:"workdir"`
}

func (s *Server) handleSessionPatch(c *gin.Context) {
	sess, ok := s.loadSessionOrRespond(c)
	if !ok {
		return
	}

	var req SessionPatchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondErrorDetails(c, errorDetails{Status: http.StatusBadRequest, Code: "invalid_json", MsgKey: "api.error.invalid_json", Details: err.Error()})
		return
	}
	if req.WorkDir != nil {
		sess.SetMetadata(session.MetadataKeyWorkdir, *req.WorkDir)
	}
	if err := s.store.Save(sess); err != nil {
		respondErrorDetails(c, errorDetails{Status: http.StatusInternalServerError, Code: "session_error", MsgKey: "api.error.session_error", Details: err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": sess.ID, "metadata": sess.GetMetadata()})
}

func (s *Server) handleSessionsDelete(c *gin.Context) {
	if !s.requireSessionStore(c) {
		return
	}
	id := c.Param("id")
	userID := identityFromGin(c).UserID
	if err := s.svc.DeleteSession(id, userID); err != nil {
		if errors.Is(err, session.ErrNotFound) {
			writeSessionLoadError(c, err)
			return
		}
		respondErrorDetails(c, errorDetails{Status: http.StatusInternalServerError, Code: "delete_error", MsgKey: "api.error.delete_error", Details: err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"deleted": id})
}

// handleUsageSummary returns the aggregated token consumption report built
// from the LLM request audit logs. Query params: days (lookback window),
// all (1 = admin cross-user report, default: own usage only).
func (s *Server) handleUsageSummary(c *gin.Context) {
	days := 0
	if v := c.Query("days"); v != "" {
		_, _ = fmt.Sscanf(v, "%d", &days)
	}
	userID := identityFromGin(c).UserID
	if c.Query("all") == "1" && identityFromGin(c).Role == store.RoleAdmin {
		userID = "" // admins may aggregate across all users
	}
	summary, err := s.requestLogger.UsageSummary(audit.UsageQuery{Days: days, UserID: userID})
	if err != nil {
		respondErrorDetails(c, errorDetails{Status: http.StatusInternalServerError, Code: "usage_error", MsgKey: "api.error.audit_error", Details: err.Error()})
		return
	}
	c.JSON(http.StatusOK, summary)
}

// handleAuditRequests returns recent LLM request audit records, newest
// first. Optional filters: session_id, source, limit. Only lightweight
// summaries are returned; full request/response payloads are fetched via
// GET /v1/audit/requests/detail to keep list responses small.
func (s *Server) handleAuditRequests(c *gin.Context) {
	limit := 0
	if v := c.Query("limit"); v != "" {
		_, _ = fmt.Sscanf(v, "%d", &limit)
	}
	records, err := s.requestLogger.Query(audit.RequestFilter{
		SessionID: c.Query("session_id"),
		Source:    c.Query("source"),
		Limit:     limit,
	})
	if err != nil {
		respondErrorDetails(c, errorDetails{Status: http.StatusInternalServerError, Code: "audit_error", MsgKey: "api.error.audit_error", Details: err.Error()})
		return
	}
	out := make([]audit.RequestSummary, 0, len(records))
	for _, r := range records {
		out = append(out, r.Summary())
	}
	c.JSON(http.StatusOK, out)
}

// handleAuditRequestDetail returns the full request record (including
// request/response payloads) for a single audited LLM call, identified by
// its timestamp string (RFC3339Nano) and optional session_id.
func (s *Server) handleAuditRequestDetail(c *gin.Context) {
	ts := c.Query("ts")
	if ts == "" {
		respondError(c, http.StatusBadRequest, "invalid_request", "api.error.invalid_request")
		return
	}
	rec, err := s.requestLogger.Find(ts, c.Query("session_id"))
	if err != nil {
		respondError(c, http.StatusNotFound, "not_found", "api.error.audit_error")
		return
	}
	c.JSON(http.StatusOK, rec)
}
