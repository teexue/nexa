package httpapi

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/teexue/nexa/core/version"
)

// handleVersion returns the build version, injected at release time via
// -ldflags -X core/version.Version=... (see Makefile / release workflow).
// Public on purpose: the settings page shows it before any auth is needed.
func (s *Server) handleVersion(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"version": version.Version,
	})
}
