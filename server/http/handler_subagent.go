package httpapi

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/teexue/nexa/core/config"
)

func (s *Server) handleSubagentGet(c *gin.Context) {
	view, err := s.svc.GetSubagentSettings()
	if err != nil {
		respondErrorDetails(c, errorDetails{
			Status: http.StatusInternalServerError, Code: "config_error",
			MsgKey: "api.error.config_error", Details: err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, view)
}

func (s *Server) handleSubagentPut(c *gin.Context) {
	var view config.SubagentView
	if err := c.ShouldBindJSON(&view); err != nil {
		respondErrorDetails(c, errorDetails{
			Status: http.StatusBadRequest, Code: "invalid_json",
			MsgKey: "api.error.invalid_json", Details: err.Error(),
		})
		return
	}
	if err := s.svc.SaveSubagentSettings(view); err != nil {
		respondServiceError(c, err, errorDetails{
			Status: http.StatusInternalServerError, Code: "config_error",
			MsgKey: "api.error.config_error",
		})
		return
	}
	saved, err := s.svc.GetSubagentSettings()
	if err != nil {
		respondErrorDetails(c, errorDetails{
			Status: http.StatusInternalServerError, Code: "config_error",
			MsgKey: "api.error.config_error", Details: err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, saved)
}
