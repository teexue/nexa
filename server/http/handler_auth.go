package httpapi

import (
	"errors"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/teexue/nexa/core/auth"
	"github.com/teexue/nexa/core/service"
	"github.com/teexue/nexa/core/store"
)

type createAPIKeyRequest struct {
	Name          string   `json:"name"`
	Scopes        []string `json:"scopes"`
	ExpiresInDays int      `json:"expires_in_days"`
}

type createAPIKeyResponse struct {
	ID        string   `json:"id"`
	Key       string   `json:"key"` // raw key, returned once
	Prefix    string   `json:"prefix"`
	Scopes    []string `json:"scopes"`
	ExpiresAt string   `json:"expires_at,omitempty"`
}

type patchAPIKeyRequest struct {
	Name    *string  `json:"name"`
	Scopes  []string `json:"scopes"`
	Enabled *bool    `json:"enabled"`
}

type tokenRequest struct {
	APIKey string `json:"api_key"`
}

type registerRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Name     string `json:"name"`
}

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (s *Server) handleAuthStatus(c *gin.Context) {
	enabled, _ := s.authEnabled()
	st := s.svc.AuthProbe()
	c.JSON(http.StatusOK, gin.H{
		"auth_required":      enabled,
		"has_users":          st.HasUsers,
		"allow_registration": st.AllowRegistration,
	})
}

func (s *Server) handleAuthRegister(c *gin.Context) {
	if s.tokens == nil {
		respondError(c, http.StatusServiceUnavailable, "not_configured", "api.error.auth_keys_not_configured")
		return
	}
	var req registerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondErrorDetails(c, errorDetails{Status: http.StatusBadRequest, Code: "invalid_json", MsgKey: "api.error.invalid_json", Details: err.Error()})
		return
	}
	u, err := s.svc.RegisterUser(req.Username, req.Password, req.Name)
	if err != nil {
		respondRegisterError(c, err)
		return
	}
	token, err := s.tokens.IssueLogin(u.ID)
	if err != nil {
		respondErrorDetails(c, errorDetails{Status: http.StatusInternalServerError, Code: "auth_error", MsgKey: "api.error.internal", Details: err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{
		"token": token, "user": u.ToUserInfo(), "user_id": u.ID,
	})
}

func respondRegisterError(c *gin.Context, err error) {
	if respondAuthConfigError(c, err) {
		return
	}
	if errors.Is(err, service.ErrRegistrationDisabled) {
		respondError(c, http.StatusForbidden, "registration_disabled", "api.error.registration_disabled")
		return
	}
	var sev *service.ServerError
	if errors.As(err, &sev) {
		respondErrorDetails(c, errorDetails{Status: http.StatusInternalServerError, Code: "auth_error", MsgKey: "api.error.internal", Details: err.Error()})
		return
	}
	respondErrorDetails(c, errorDetails{Status: http.StatusBadRequest, Code: "invalid_request", MsgKey: "api.error.invalid_request", Details: err.Error()})
}

func (s *Server) handleAuthLogin(c *gin.Context) {
	if s.tokens == nil {
		respondError(c, http.StatusServiceUnavailable, "not_configured", "api.error.auth_keys_not_configured")
		return
	}
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondErrorDetails(c, errorDetails{Status: http.StatusBadRequest, Code: "invalid_json", MsgKey: "api.error.invalid_json", Details: err.Error()})
		return
	}
	u, err := s.svc.AuthenticateUser(req.Username, req.Password)
	if err != nil {
		if respondAuthConfigError(c, err) {
			return
		}
		respondError(c, http.StatusUnauthorized, "unauthorized", "api.error.invalid_credentials")
		return
	}
	token, err := s.tokens.IssueLogin(u.ID)
	if err != nil {
		respondErrorDetails(c, errorDetails{Status: http.StatusInternalServerError, Code: "auth_error", MsgKey: "api.error.internal", Details: err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"token": token, "user": u.ToUserInfo(), "user_id": u.ID,
	})
}

func (s *Server) handleAuthKeysList(c *gin.Context) {
	id := identityFromGin(c)
	keys, err := s.svc.ListAPIKeys(id.UserID)
	if err != nil {
		respondErrorDetails(c, errorDetails{Status: http.StatusInternalServerError, Code: "auth_error", MsgKey: "api.error.internal", Details: err.Error()})
		return
	}
	enabled, _ := s.authEnabled()
	c.JSON(http.StatusOK, gin.H{"enabled": enabled, "keys": keys, "user_id": id.UserID})
}

func (s *Server) handleAuthKeysCreate(c *gin.Context) {
	var req createAPIKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondErrorDetails(c, errorDetails{Status: http.StatusBadRequest, Code: "invalid_json", MsgKey: "api.error.invalid_json", Details: err.Error()})
		return
	}
	created, err := s.svc.CreateAPIKey(service.CreateAPIKeyRequest{
		UserID: identityFromGin(c).UserID, Name: req.Name,
		Scopes: req.Scopes, ExpiresInDays: req.ExpiresInDays,
	})
	if err != nil {
		if respondAuthConfigError(c, err) {
			return
		}
		respondErrorDetails(c, errorDetails{Status: http.StatusBadRequest, Code: "invalid_request", MsgKey: "api.error.invalid_request", Details: err.Error()})
		return
	}
	resp := createAPIKeyResponse{
		ID: created.Key.ID, Key: created.Raw, Prefix: created.Key.Prefix,
		Scopes: parseScopes(created.Key.Scopes),
	}
	if created.Key.ExpiresAt != nil {
		resp.ExpiresAt = created.Key.ExpiresAt.UTC().Format("2006-01-02T15:04:05Z")
	}
	c.JSON(http.StatusCreated, resp)
}

func (s *Server) handleAuthKeysPatch(c *gin.Context) {
	var req patchAPIKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondErrorDetails(c, errorDetails{Status: http.StatusBadRequest, Code: "invalid_json", MsgKey: "api.error.invalid_json", Details: err.Error()})
		return
	}
	patch := store.APIKeyPatch{Name: req.Name, Enabled: req.Enabled}
	if req.Scopes != nil {
		scopes := strings.Join(req.Scopes, ",")
		patch.Scopes = &scopes
	}
	info, err := s.svc.PatchAPIKey(c.Param("id"), identityFromGin(c).UserID, patch)
	if err != nil {
		respondAuthKeyMutateError(c, err)
		return
	}
	c.JSON(http.StatusOK, info)
}

func (s *Server) handleAuthKeysDelete(c *gin.Context) {
	id := c.Param("id")
	if err := s.svc.DeleteAPIKey(id, identityFromGin(c).UserID); err != nil {
		if respondAuthConfigError(c, err) {
			return
		}
		if errors.Is(err, os.ErrNotExist) {
			respondError(c, http.StatusNotFound, "not_found", "api.error.auth_key_not_found")
			return
		}
		respondErrorDetails(c, errorDetails{Status: http.StatusInternalServerError, Code: "delete_error", MsgKey: "api.error.delete_error", Details: err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true, "id": id})
}

func respondAuthKeyMutateError(c *gin.Context, err error) {
	if respondAuthConfigError(c, err) {
		return
	}
	if errors.Is(err, os.ErrNotExist) {
		respondError(c, http.StatusNotFound, "not_found", "api.error.auth_key_not_found")
		return
	}
	respondErrorDetails(c, errorDetails{Status: http.StatusBadRequest, Code: "invalid_request", MsgKey: "api.error.invalid_request", Details: err.Error()})
}

func (s *Server) handleAuthToken(c *gin.Context) {
	if s.tokens == nil {
		respondError(c, http.StatusServiceUnavailable, "not_configured", "api.error.auth_keys_not_configured")
		return
	}
	var req tokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondErrorDetails(c, errorDetails{Status: http.StatusBadRequest, Code: "invalid_json", MsgKey: "api.error.invalid_json", Details: err.Error()})
		return
	}
	entry, err := s.svc.VerifyStoredAPIKey(req.APIKey)
	if err != nil {
		if respondAuthConfigError(c, err) {
			return
		}
		respondErrorDetails(c, errorDetails{Status: http.StatusInternalServerError, Code: "auth_error", MsgKey: "api.error.internal", Details: err.Error()})
		return
	}
	if entry == nil {
		s.issueCLIToken(c, req.APIKey)
		return
	}
	token, err := s.tokens.Issue(auth.Identity{UserID: entry.UserID, KeyID: entry.ID})
	if err != nil {
		respondErrorDetails(c, errorDetails{Status: http.StatusInternalServerError, Code: "auth_error", MsgKey: "api.error.internal", Details: err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"token": token, "user_id": entry.UserID, "key_id": entry.ID})
}

func (s *Server) issueCLIToken(c *gin.Context, rawKey string) {
	if id, ok := s.resolveCLIKey(rawKey); ok {
		token, err := s.tokens.Issue(id)
		if err != nil {
			respondErrorDetails(c, errorDetails{Status: http.StatusInternalServerError, Code: "auth_error", MsgKey: "api.error.internal", Details: err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"token": token, "user_id": id.UserID, "key_id": id.KeyID})
		return
	}
	respondError(c, http.StatusUnauthorized, "unauthorized", "api.error.unauthorized")
}

func (s *Server) handleAuthMe(c *gin.Context) {
	id := identityFromGin(c)
	resp := gin.H{
		"user_id":          id.UserID,
		"key_id":           id.KeyID,
		"role":             id.Role,
		"password_session": id.IsPasswordSession(),
		"auth_enabled":     false,
	}
	if len(id.Scopes) > 0 {
		resp["scopes"] = id.Scopes
	}
	if enabled, err := s.authEnabled(); err == nil {
		resp["auth_enabled"] = enabled
	}
	if u, ok := s.svc.LookupUser(id.UserID); ok {
		resp["user"] = u
	}
	c.JSON(http.StatusOK, resp)
}
