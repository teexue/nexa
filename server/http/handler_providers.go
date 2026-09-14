package httpapi

import (
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/gin-gonic/gin"

	"github.com/teexue/nexa/core/config"
	"github.com/teexue/nexakit/provider"
)

func (s *Server) handleProvidersList(c *gin.Context) {
	if s.catalog == nil {
		c.JSON(http.StatusOK, []provider.ProviderInfo{})
		return
	}
	c.JSON(http.StatusOK, s.catalog.Entries())
}

// handleVendors returns the built-in vendor presets (no secrets).
func (s *Server) handleVendors(c *gin.Context) {
	c.JSON(http.StatusOK, provider.VendorInfos())
}

// handleProviderModels returns the enabled model subset for a saved provider.
// It does not call the vendor listing API; unselected models stay hidden.
func (s *Server) handleProviderModels(c *gin.Context) {
	if s.catalog == nil {
		respondError(c, http.StatusServiceUnavailable, "no_catalog", "api.error.no_catalog")
		return
	}
	name := c.Param("name")
	if name == "" {
		respondError(c, http.StatusBadRequest, "invalid_request", "api.error.invalid_request")
		return
	}
	models, err := s.catalog.EnabledModelInfos(name)
	if err != nil {
		respondErrorDetails(c, errorDetails{Status: http.StatusNotFound, Code: "not_found", MsgKey: "api.error.invalid_request", Details: err.Error()})
		return
	}
	c.JSON(http.StatusOK, models)
}

// handleProviderModelDetail returns structured metadata for a single model
// from a configured provider (e.g. Ollama /api/show: context length, family,
// parameter size, quantization, capabilities).
func (s *Server) handleProviderModelDetail(c *gin.Context) {
	if s.catalog == nil {
		respondError(c, http.StatusServiceUnavailable, "no_catalog", "api.error.no_catalog")
		return
	}
	name := c.Param("name")
	model := c.Param("model")
	if name == "" || model == "" {
		respondError(c, http.StatusBadRequest, "invalid_request", "api.error.invalid_request")
		return
	}
	detail, err := s.catalog.ShowModel(c.Request.Context(), name, model)
	if err != nil {
		respondErrorDetails(c, errorDetails{Status: http.StatusBadGateway, Code: "provider_error", MsgKey: "api.error.provider_error", Details: err.Error()})
		return
	}
	s.rememberModelWindow(name, model, detail.ContextWindow)
	c.JSON(http.StatusOK, detail)
}

// ProviderModelsRequest is the DTO for POST /v1/providers/models.
// It fetches models using inline config (no saved provider required), so the
// UI can pull a model list while creating a provider before saving it.
type ProviderModelsRequest struct {
	Name       string `json:"name,omitempty"`
	APIStyle   string `json:"api_style"`
	BaseURL    string `json:"base_url,omitempty"`
	ModelsPath string `json:"models_path,omitempty"`
	APIVersion string `json:"api_version,omitempty"`
	AuthStyle  string `json:"auth_style,omitempty"`
	APIKey     string `json:"api_key,omitempty"`
}

// buildInlineProvider constructs a provider from an inline config, filling in
// vendor defaults and falling back to a saved provider's API key when one is
// not supplied. It is shared by the inline model-list and model-detail
// endpoints so the UI can introspect a provider before it is saved.
func (s *Server) buildInlineProvider(req ProviderModelsRequest) (provider.Provider, error) {
	style := provider.APIStyle(req.APIStyle)
	if style != provider.StyleOpenAI && style != provider.StyleAnthropic && style != provider.StyleOllama {
		return nil, fmt.Errorf("unsupported api_style %q", req.APIStyle)
	}

	apiKey := req.APIKey
	if apiKey == "" && req.Name != "" && s.catalog != nil {
		if prof, err := s.catalog.Get(req.Name); err == nil {
			apiKey = prof.APIKey
		}
	}
	// Local Ollama needs no API key; only require a key for the other styles.
	if apiKey == "" && style != provider.StyleOllama {
		return nil, fmt.Errorf("api_key required for %q", req.APIStyle)
	}

	baseURL, modelsPath, apiVersion, authStyle := inlineProviderFields(req, style)
	return provider.NewProvider(provider.ListingProfile(provider.Profile{
		Name:       req.Name,
		APIStyle:   style,
		BaseURL:    baseURL,
		APIKey:     apiKey,
		APIVersion: apiVersion,
		AuthStyle:  authStyle,
		ModelsPath: modelsPath,
	}))
}

func inlineProviderFields(req ProviderModelsRequest, style provider.APIStyle) (string, string, string, provider.AuthStyle) {
	baseURL := req.BaseURL
	modelsPath := req.ModelsPath
	apiVersion := req.APIVersion
	authStyle := provider.AuthStyle(req.AuthStyle)
	if v, ok := provider.LookupVendor(req.Name); ok {
		if baseURL == "" {
			baseURL = v.BaseURLFor(style)
		}
		if modelsPath == "" {
			modelsPath = provider.DefaultModelsPathFor(style)
		}
		if apiVersion == "" {
			apiVersion = v.APIVersion
		}
		if authStyle == "" {
			authStyle = v.AuthForStyle(style)
		}
	}
	if baseURL == "" {
		baseURL = provider.DefaultBaseURLFor(style)
	}
	if modelsPath == "" {
		modelsPath = provider.DefaultModelsPathFor(style)
	}
	if authStyle == "" {
		if style == provider.StyleAnthropic {
			authStyle = provider.AuthXAPIKey
		} else {
			authStyle = provider.AuthBearer
		}
	}
	return baseURL, modelsPath, apiVersion, authStyle
}

// handleProviderModelsTest fetches models from an inline provider config.
// If api_key is empty and name matches a saved provider, the stored key is used.
func (s *Server) handleProviderModelsTest(c *gin.Context) {
	var req ProviderModelsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondErrorDetails(c, errorDetails{Status: http.StatusBadRequest, Code: "invalid_json", MsgKey: "api.error.invalid_json", Details: err.Error()})
		return
	}
	p, err := s.buildInlineProvider(req)
	if err != nil {
		respondErrorDetails(c, errorDetails{Status: http.StatusBadRequest, Code: "provider_error", MsgKey: "api.error.provider_error", Details: err.Error()})
		return
	}
	lister, ok := p.(provider.ModelLister)
	if !ok {
		respondErrorDetails(c, errorDetails{Status: http.StatusBadGateway, Code: "provider_error", MsgKey: "api.error.provider_error", Details: "provider does not support model listing"})
		return
	}
	models, err := lister.ListModels(c.Request.Context())
	if err != nil {
		respondErrorDetails(c, errorDetails{Status: http.StatusBadGateway, Code: "provider_error", MsgKey: "api.error.provider_error", Details: err.Error()})
		return
	}
	c.JSON(http.StatusOK, models)
}

// ProviderModelDetailRequest is the DTO for POST /v1/providers/models/detail.
// It introspects a single model using inline config, mirroring the inline
// model-list endpoint so the UI can show model details before saving.
type ProviderModelDetailRequest struct {
	ProviderModelsRequest
	Model string `json:"model"`
}

// handleProviderModelDetailTest returns structured metadata for a single model
// using inline provider config. Only providers implementing ModelDetailer
// (e.g. Ollama) are supported.
func (s *Server) handleProviderModelDetailTest(c *gin.Context) {
	var req ProviderModelDetailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondErrorDetails(c, errorDetails{Status: http.StatusBadRequest, Code: "invalid_json", MsgKey: "api.error.invalid_json", Details: err.Error()})
		return
	}
	if req.Model == "" {
		respondError(c, http.StatusBadRequest, "invalid_request", "api.error.invalid_request")
		return
	}
	p, err := s.buildInlineProvider(req.ProviderModelsRequest)
	if err != nil {
		respondErrorDetails(c, errorDetails{Status: http.StatusBadRequest, Code: "provider_error", MsgKey: "api.error.provider_error", Details: err.Error()})
		return
	}
	detailer, ok := p.(provider.ModelDetailer)
	if !ok {
		respondErrorDetails(c, errorDetails{Status: http.StatusBadGateway, Code: "provider_error", MsgKey: "api.error.provider_error", Details: "provider does not support model detail"})
		return
	}
	detail, err := detailer.ShowModel(c.Request.Context(), req.Model)
	if err != nil {
		respondErrorDetails(c, errorDetails{Status: http.StatusBadGateway, Code: "provider_error", MsgKey: "api.error.provider_error", Details: err.Error()})
		return
	}
	c.JSON(http.StatusOK, detail)
}

// ProviderUpsertRequest is the DTO for POST/PUT /v1/providers.
type ProviderUpsertRequest struct {
	Name          string   `json:"name"`
	APIStyle      string   `json:"api_style"`
	BaseURL       string   `json:"base_url,omitempty"`
	APIKey        string   `json:"api_key,omitempty"`
	APIKeyEnv     string   `json:"api_key_env,omitempty"`
	APIVersion    string   `json:"api_version,omitempty"`
	AuthStyle     string   `json:"auth_style,omitempty"`
	DefaultModel  string   `json:"default_model,omitempty"`
	DisplayName   string   `json:"display_name,omitempty"`
	Models        []string `json:"models,omitempty"`
	ModelsPath    string   `json:"models_path,omitempty"`
	Vision        bool     `json:"vision,omitempty"`
	ContextWindow int      `json:"context_window,omitempty"`
}

func (s *Server) handleProviderUpsert(c *gin.Context) {
	var req ProviderUpsertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondErrorDetails(c, errorDetails{Status: http.StatusBadRequest, Code: "invalid_json", MsgKey: "api.error.invalid_json", Details: err.Error()})
		return
	}
	if req.Name == "" {
		respondError(c, http.StatusBadRequest, "invalid_request", "api.error.invalid_request")
		return
	}

	home := filepath.Dir(s.agentsDir)
	existingEnv, _ := config.LookupAPIKeyEnv(req.Name)
	vendorEnv := ""
	hasVendor := false
	if v, ok := provider.LookupVendor(req.Name); ok {
		vendorEnv = v.APIKeyEnv
		hasVendor = true
	}
	req.APIKeyEnv = resolveProviderAPIKeyEnv(req.Name, req.APIKeyEnv, existingEnv, vendorEnv, hasVendor)

	spec := config.ProviderSpec{
		Name:         req.Name,
		APIStyle:     provider.APIStyle(req.APIStyle),
		BaseURL:      req.BaseURL,
		APIKeyEnv:    req.APIKeyEnv,
		APIVersion:   req.APIVersion,
		AuthStyle:    provider.AuthStyle(req.AuthStyle),
		DefaultModel: req.DefaultModel,
		DisplayName:  req.DisplayName,
		Models:       req.Models,
		ModelsPath:   req.ModelsPath,
		Vision:       req.Vision,
		ModelWindows: s.modelWindowsForUpsert(c.Request.Context(), req),
	}
	if err := config.UpsertProvider(home, spec); err != nil {
		respondErrorDetails(c, errorDetails{Status: http.StatusBadRequest, Code: "provider_error", MsgKey: "api.error.provider_error", Details: err.Error()})
		return
	}

	if req.APIKey != "" && req.APIKeyEnv != "" {
		if s.creds == nil {
			cs, err := config.NewCredentialStore(home)
			if err != nil {
				respondErrorDetails(c, errorDetails{Status: http.StatusInternalServerError, Code: "provider_error", MsgKey: "api.error.provider_error", Details: err.Error()})
				return
			}
			s.creds = cs
		}
		if err := s.creds.Set(req.APIKeyEnv, req.APIKey); err != nil {
			respondErrorDetails(c, errorDetails{Status: http.StatusBadRequest, Code: "provider_error", MsgKey: "api.error.provider_error", Details: err.Error()})
			return
		}
	}

	// Reload catalog.
	if err := s.reloadCatalog(); err != nil {
		s.logger.Warn("reload catalog after upsert", "error", err)
	}

	c.JSON(http.StatusOK, gin.H{"ok": true, "name": req.Name, "api_key_env": req.APIKeyEnv})
}

func defaultAPIKeyEnv(name string) string {
	var b strings.Builder
	for _, r := range strings.ToUpper(name) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		} else {
			b.WriteByte('_')
		}
	}
	s := strings.Trim(b.String(), "_")
	if s == "" {
		s = "PROVIDER"
	}
	return s + "_API_KEY"
}

// resolveProviderAPIKeyEnv picks the credential env name for a provider upsert.
// An existing provider keeps its env unless the request sets one explicitly, so
// a re-save without an API key cannot retarget a newly invented name.
func resolveProviderAPIKeyEnv(name, requested, existing, vendorEnv string, hasVendor bool) string {
	if requested != "" {
		return requested
	}
	if existing != "" {
		return existing
	}
	if hasVendor {
		return vendorEnv
	}
	return defaultAPIKeyEnv(name)
}

func (s *Server) handleProviderDelete(c *gin.Context) {
	name := c.Param("name")
	if name == "" {
		respondError(c, http.StatusBadRequest, "invalid_request", "api.error.invalid_request")
		return
	}

	home := filepath.Dir(s.agentsDir)
	if err := config.DeleteProvider(home, name); err != nil {
		respondErrorDetails(c, errorDetails{Status: http.StatusBadRequest, Code: "provider_error", MsgKey: "api.error.provider_error", Details: err.Error()})
		return
	}

	// Reload catalog.
	if err := s.reloadCatalog(); err != nil {
		s.logger.Warn("reload catalog after delete", "error", err)
	}

	c.JSON(http.StatusOK, gin.H{"deleted": name})
}

func (s *Server) reloadCatalog() error {
	home := filepath.Dir(s.agentsDir)
	var lookup func(string) string
	if s.creds != nil {
		lookup = s.creds.Lookup
	} else {
		cs, err := config.NewCredentialStore(home)
		if err == nil {
			s.creds = cs
			lookup = cs.Lookup
		}
	}
	var cat *provider.Catalog
	var err error
	if s.stateDB != nil {
		cat, err = s.stateDB.LoadCatalog(lookup)
	} else {
		return fmt.Errorf("state.db is not open")
	}
	if err != nil {
		return err
	}
	s.SetCatalog(cat)
	return nil
}
