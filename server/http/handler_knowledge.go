package httpapi

import (
	"errors"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/teexue/nexa/core/knowledge"
	"github.com/teexue/nexa/core/service"
)

func (s *Server) handleKnowledgeList(c *gin.Context) {
	metas, err := s.svc.ListKnowledge()
	if err != nil {
		respondKnowledgeErr(c, err)
		return
	}
	if metas == nil {
		metas = []knowledge.Meta{}
	}
	c.JSON(http.StatusOK, gin.H{"bases": metas})
}

func (s *Server) handleKnowledgeCreate(c *gin.Context) {
	var req struct {
		ID          string `json:"id"`
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondErrorDetails(c, errorDetails{Status: http.StatusBadRequest, Code: "invalid_json", MsgKey: "api.error.invalid_json", Details: err.Error()})
		return
	}
	meta, err := s.svc.CreateKnowledge(req.ID, req.Name, req.Description)
	if err != nil {
		respondKnowledgeErr(c, err)
		return
	}
	c.JSON(http.StatusCreated, meta)
}

func (s *Server) handleKnowledgeGet(c *gin.Context) {
	meta, err := s.svc.GetKnowledge(c.Param("id"))
	if err != nil {
		respondKnowledgeErr(c, err)
		return
	}
	c.JSON(http.StatusOK, meta)
}

func (s *Server) handleKnowledgeUpdate(c *gin.Context) {
	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondErrorDetails(c, errorDetails{Status: http.StatusBadRequest, Code: "invalid_json", MsgKey: "api.error.invalid_json", Details: err.Error()})
		return
	}
	meta, err := s.svc.UpdateKnowledge(c.Param("id"), req.Name, req.Description)
	if err != nil {
		respondKnowledgeErr(c, err)
		return
	}
	c.JSON(http.StatusOK, meta)
}

func (s *Server) handleKnowledgeDelete(c *gin.Context) {
	if err := s.svc.DeleteKnowledge(c.Param("id")); err != nil {
		respondKnowledgeErr(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (s *Server) handleKnowledgeDocsList(c *gin.Context) {
	docs, err := s.svc.ListKnowledgeDocuments(c.Param("id"))
	if err != nil {
		respondKnowledgeErr(c, err)
		return
	}
	if docs == nil {
		docs = []knowledge.Document{}
	}
	c.JSON(http.StatusOK, gin.H{"documents": docs})
}

func (s *Server) handleKnowledgeDocUpload(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		respondErrorDetails(c, errorDetails{Status: http.StatusBadRequest, Code: "invalid_json", MsgKey: "api.error.invalid_json", Details: err.Error()})
		return
	}
	f, err := file.Open()
	if err != nil {
		respondErrorDetails(c, errorDetails{Status: http.StatusBadRequest, Code: "invalid_request", MsgKey: "api.error.invalid_request", Details: err.Error()})
		return
	}
	defer f.Close()
	content, err := io.ReadAll(f)
	if err != nil {
		respondErrorDetails(c, errorDetails{Status: http.StatusBadRequest, Code: "invalid_request", MsgKey: "api.error.invalid_request", Details: err.Error()})
		return
	}
	doc, err := s.svc.AddKnowledgeDocument(c.Request.Context(), c.Param("id"), file.Filename, content)
	if err != nil {
		respondKnowledgeErr(c, err)
		return
	}
	c.JSON(http.StatusCreated, doc)
}

func (s *Server) handleKnowledgeDocDelete(c *gin.Context) {
	if err := s.svc.DeleteKnowledgeDocument(c.Param("id"), c.Param("docId")); err != nil {
		respondKnowledgeErr(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (s *Server) handleKnowledgeReindex(c *gin.Context) {
	if err := s.svc.ReindexKnowledge(c.Request.Context(), c.Param("id")); err != nil {
		respondKnowledgeErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (s *Server) handleKnowledgeSearch(c *gin.Context) {
	var req struct {
		Query string   `json:"query"`
		KBIDs []string `json:"kb_ids"`
		TopK  int      `json:"top_k"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondErrorDetails(c, errorDetails{Status: http.StatusBadRequest, Code: "invalid_json", MsgKey: "api.error.invalid_json", Details: err.Error()})
		return
	}
	hits, err := s.svc.SearchKnowledge(c.Request.Context(), req.Query, req.KBIDs, req.TopK)
	if err != nil {
		respondKnowledgeErr(c, err)
		return
	}
	if hits == nil {
		hits = []knowledge.Hit{}
	}
	c.JSON(http.StatusOK, gin.H{"hits": hits, "count": len(hits)})
}

func (s *Server) handleEmbeddingGet(c *gin.Context) {
	view, err := s.svc.GetEmbeddingSettings()
	if err != nil {
		respondErrorDetails(c, errorDetails{Status: http.StatusInternalServerError, Code: "config_error", MsgKey: "api.error.config_error", Details: err.Error()})
		return
	}
	c.JSON(http.StatusOK, view)
}

func (s *Server) handleEmbeddingVendors(c *gin.Context) {
	c.JSON(http.StatusOK, s.svc.ListEmbeddingVendors())
}

func (s *Server) handleEmbeddingPut(c *gin.Context) {
	var req service.SaveEmbeddingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondErrorDetails(c, errorDetails{Status: http.StatusBadRequest, Code: "invalid_json", MsgKey: "api.error.invalid_json", Details: err.Error()})
		return
	}
	if err := s.svc.SaveEmbeddingSettings(req); err != nil {
		respondKnowledgeErr(c, err)
		return
	}
	view, err := s.svc.GetEmbeddingSettings()
	if err != nil {
		respondErrorDetails(c, errorDetails{Status: http.StatusInternalServerError, Code: "config_error", MsgKey: "api.error.config_error", Details: err.Error()})
		return
	}
	c.JSON(http.StatusOK, view)
}

func respondKnowledgeErr(c *gin.Context, err error) {
	var arg *service.ArgError
	if errors.As(err, &arg) {
		respondErrorDetails(c, errorDetails{Status: http.StatusBadRequest, Code: "invalid_request", MsgKey: "api.error.invalid_request", Details: arg.Error()})
		return
	}
	var srv *service.ServerError
	if errors.As(err, &srv) {
		respondErrorDetails(c, errorDetails{Status: http.StatusServiceUnavailable, Code: "knowledge_error", MsgKey: "api.error.knowledge_error", Details: srv.Error()})
		return
	}
	if errors.Is(err, os.ErrNotExist) || strings.Contains(err.Error(), "not exist") {
		respondErrorDetails(c, errorDetails{Status: http.StatusNotFound, Code: "not_found", MsgKey: "api.error.not_found", Details: err.Error()})
		return
	}
	if strings.Contains(err.Error(), "already exists") || strings.Contains(err.Error(), "unsupported") || strings.Contains(err.Error(), "invalid id") {
		respondErrorDetails(c, errorDetails{Status: http.StatusBadRequest, Code: "invalid_request", MsgKey: "api.error.invalid_request", Details: err.Error()})
		return
	}
	respondErrorDetails(c, errorDetails{Status: http.StatusInternalServerError, Code: "knowledge_error", MsgKey: "api.error.knowledge_error", Details: err.Error()})
}
