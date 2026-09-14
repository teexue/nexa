package httpapi

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/teexue/nexa/core/auth"
	"github.com/teexue/nexa/core/store"
)

const ginIdentityKey = "identity"

// authMiddleware enforces authentication for /v1/ routes when auth is enabled.
func (s *Server) authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		enabled, err := s.authEnabled()
		if err != nil {
			respondErrorDetails(c, errorDetails{Status: http.StatusInternalServerError, Code: "auth_error", MsgKey: "api.error.internal", Details: err.Error()})
			c.Abort()
			return
		}
		if !enabled {
			id := auth.Identity{Role: store.RoleAdmin}
			c.Set(ginIdentityKey, id)
			c.Request = c.Request.WithContext(auth.WithIdentity(c.Request.Context(), id))
			c.Next()
			return
		}

		token := extractRequestToken(c)
		if token == "" {
			respondError(c, http.StatusUnauthorized, "unauthorized", "api.error.unauthorized")
			c.Abort()
			return
		}

		id, ok := s.resolveIdentity(token)
		if !ok {
			respondError(c, http.StatusUnauthorized, "unauthorized", "api.error.unauthorized")
			c.Abort()
			return
		}
		c.Set(ginIdentityKey, id)
		c.Request = c.Request.WithContext(auth.WithIdentity(c.Request.Context(), id))
		c.Next()
	}
}

func extractRequestToken(c *gin.Context) string {
	authz := c.GetHeader("Authorization")
	if len(authz) > 7 && strings.EqualFold(authz[:7], "bearer ") {
		return strings.TrimSpace(authz[7:])
	}
	if key := c.GetHeader("X-API-Key"); key != "" {
		return key
	}
	if c.Request.Method == http.MethodGet || c.Request.Method == http.MethodHead {
		switch c.Request.URL.Path {
		case "/v1/events", "/v1/background":
			if t := c.Query("access_token"); t != "" {
				return t
			}
			return c.Query("api_key")
		}
	}
	return ""
}

func identityFromGin(c *gin.Context) auth.Identity {
	if v, ok := c.Get(ginIdentityKey); ok {
		if id, ok := v.(auth.Identity); ok {
			return id
		}
	}
	return auth.Identity{}
}

// requireAdmin allows only identities with the admin role (403 otherwise).
func requireAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		if identityFromGin(c).Role != store.RoleAdmin {
			respondError(c, http.StatusForbidden, "forbidden", "api.error.forbidden")
			c.Abort()
			return
		}
		c.Next()
	}
}

// requireScope allows identities carrying the given scope. Password sessions
// always pass; only API key identities are constrained (see auth.HasScope).
func requireScope(scope string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !auth.HasScope(identityFromGin(c), scope) {
			respondError(c, http.StatusForbidden, "forbidden", "api.error.forbidden")
			c.Abort()
			return
		}
		c.Next()
	}
}

// backgroundUploadMaxBytes is the body size limit for POST /v1/background
// and knowledge document uploads.
const backgroundUploadMaxBytes = 50 << 20 // 50 MB

// bodySizeLimit returns middleware that limits request body size.
func (s *Server) bodySizeLimit(maxBytes int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		limit := maxBytes
		if c.Request.Method == http.MethodPost {
			path := c.Request.URL.Path
			if path == "/v1/background" ||
				(strings.HasPrefix(path, "/v1/knowledge/") && strings.HasSuffix(path, "/documents")) {
				limit = backgroundUploadMaxBytes
			}
		}
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, limit)
		c.Next()
	}
}
