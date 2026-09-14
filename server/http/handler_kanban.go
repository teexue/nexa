package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/teexue/nexa/core/service"
	"github.com/teexue/nexa/core/store"
)

// kanbanItemResponse renders a kanban row for API responses, decoding tags.
func kanbanItemResponse(row *store.KanbanRow) gin.H {
	tags := []string{}
	if row.TagsJSON != "" {
		_ = json.Unmarshal([]byte(row.TagsJSON), &tags)
	}
	return gin.H{
		"id":          row.ID,
		"user_id":     row.UserID,
		"title":       row.Title,
		"prompt":      row.Prompt,
		"agent":       row.Agent,
		"workdir":     row.WorkDir,
		"status":      row.Status,
		"priority":    row.Priority,
		"tags":        tags,
		"due_at":      row.DueAt,
		"feedback":    row.Feedback,
		"result":      row.Result,
		"session_id":  row.SessionID,
		"attempts":    row.Attempts,
		"created_at":  row.CreatedAt,
		"updated_at":  row.UpdatedAt,
		"finished_at": row.FinishedAt,
	}
}

// respondKanbanError maps service errors to HTTP responses.
func respondKanbanError(c *gin.Context, err error) {
	var argErr *service.ArgError
	switch {
	case errors.Is(err, store.ErrKanbanNotFound):
		respondError(c, http.StatusNotFound, "not_found", "api.error.kanban_not_found")
	case errors.As(err, &argErr):
		respondErrorDetails(c, errorDetails{Status: http.StatusBadRequest, Code: "invalid_request", MsgKey: "api.error.kanban_error", Details: err.Error()})
	default:
		respondErrorDetails(c, errorDetails{Status: http.StatusInternalServerError, Code: "kanban_error", MsgKey: "api.error.kanban_error", Details: err.Error()})
	}
}

func (s *Server) handleKanbanList(c *gin.Context) {
	userID := identityFromGin(c).UserID
	rows, err := s.svc.ListKanbanItems(userID)
	if err != nil {
		respondKanbanError(c, err)
		return
	}
	items := make([]gin.H, 0, len(rows))
	for i := range rows {
		items = append(items, kanbanItemResponse(&rows[i]))
	}
	c.JSON(http.StatusOK, items)
}

func (s *Server) handleKanbanGet(c *gin.Context) {
	userID := identityFromGin(c).UserID
	row, err := s.svc.GetKanbanItem(c.Param("id"), userID)
	if err != nil {
		respondKanbanError(c, err)
		return
	}
	c.JSON(http.StatusOK, kanbanItemResponse(row))
}

func (s *Server) handleKanbanCreate(c *gin.Context) {
	var req service.CreateKanbanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondErrorDetails(c, errorDetails{Status: http.StatusBadRequest, Code: "invalid_json", MsgKey: "api.error.invalid_json", Details: err.Error()})
		return
	}
	userID := identityFromGin(c).UserID
	row, err := s.svc.CreateKanbanItem(userID, req)
	if err != nil {
		respondKanbanError(c, err)
		return
	}
	c.JSON(http.StatusCreated, kanbanItemResponse(row))
}

func (s *Server) handleKanbanPatch(c *gin.Context) {
	var req service.UpdateKanbanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondErrorDetails(c, errorDetails{Status: http.StatusBadRequest, Code: "invalid_json", MsgKey: "api.error.invalid_json", Details: err.Error()})
		return
	}
	userID := identityFromGin(c).UserID
	row, err := s.svc.UpdateKanbanItem(c.Param("id"), userID, req)
	if err != nil {
		respondKanbanError(c, err)
		return
	}
	c.JSON(http.StatusOK, kanbanItemResponse(row))
}

func (s *Server) handleKanbanDelete(c *gin.Context) {
	userID := identityFromGin(c).UserID
	id := c.Param("id")
	if err := s.svc.DeleteKanbanItem(id, userID); err != nil {
		respondKanbanError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"deleted": id})
}

func (s *Server) handleKanbanApprove(c *gin.Context) {
	userID := identityFromGin(c).UserID
	row, err := s.svc.ApproveKanbanItem(c.Param("id"), userID)
	if err != nil {
		respondKanbanError(c, err)
		return
	}
	c.JSON(http.StatusOK, kanbanItemResponse(row))
}

// KanbanRejectRequest is the HTTP DTO for POST /v1/kanban/:id/reject.
type KanbanRejectRequest struct {
	Feedback string `json:"feedback"`
}

func (s *Server) handleKanbanReject(c *gin.Context) {
	var req KanbanRejectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondErrorDetails(c, errorDetails{Status: http.StatusBadRequest, Code: "invalid_json", MsgKey: "api.error.invalid_json", Details: err.Error()})
		return
	}
	userID := identityFromGin(c).UserID
	row, err := s.svc.RejectKanbanItem(c.Param("id"), userID, req.Feedback)
	if err != nil {
		respondKanbanError(c, err)
		return
	}
	c.JSON(http.StatusOK, kanbanItemResponse(row))
}

func (s *Server) handleKanbanRequeue(c *gin.Context) {
	userID := identityFromGin(c).UserID
	row, err := s.svc.RequeueKanbanItem(c.Param("id"), userID)
	if err != nil {
		respondKanbanError(c, err)
		return
	}
	c.JSON(http.StatusOK, kanbanItemResponse(row))
}
