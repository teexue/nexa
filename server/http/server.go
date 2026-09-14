package httpapi

import (
	"context"
	"io/fs"
	"log/slog"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"

	"github.com/teexue/nexa/core/agent"
	"github.com/teexue/nexa/core/audit"
	"github.com/teexue/nexa/core/auth"
	"github.com/teexue/nexa/core/config"
	"github.com/teexue/nexakit/embedding"
	"github.com/teexue/nexa/core/knowledge"
	"github.com/teexue/nexakit/provider"
	"github.com/teexue/nexa/core/service"
	"github.com/teexue/nexakit/session"
	"github.com/teexue/nexa/core/store"
	"github.com/teexue/nexa/core/telemetry"
	"github.com/teexue/nexakit/registry"
)

// Server exposes agent HTTP endpoints via Gin.
type Server struct {
	agentsDir     string
	home          string // ~/.nexa root; skills dirs derive from it
	registry      *registry.Registry
	newProvider   func(a *agent.Agent) (provider.Provider, error)
	staticFS      fs.FS // optional embedded frontend; nil disables static serving
	logger        *slog.Logger
	store         session.Store           // optional session persistence; nil disables session endpoints
	svc           *service.Service        // shared business logic
	approver      *HTTPApprover           // handles tool approval flow
	requestLogger *audit.RequestLogger    // optional LLM request audit; nil disables request logs
	catalog       *provider.Catalog       // optional provider catalog; nil disables provider listing
	creds         *config.CredentialStore // optional credentials for provider upsert/reload
	health        *telemetry.HealthServer
	watcher       *agent.Watcher  // watches agents dir for changes
	shutdownCtx   context.Context // cancelled on server shutdown; nil = no shutdown propagation
	stateDB       *store.DB
	tokens        *auth.TokenService
	cliAPIKeys    []string // raw keys from --api-key (ephemeral, hashed in-memory)
	cliKeyMu      sync.RWMutex
	cliKeyHash    map[string]string // hash -> synthetic key id

	// changeCh broadcasts agent file change events to SSE subscribers.
	changeCh chan agentChange
}

// ServerConfig holds configuration for creating a new HTTP server.
type ServerConfig struct {
	AgentsDir        string
	HomeDir          string
	Registry         *registry.Registry
	NewProvider      func(a *agent.Agent) (provider.Provider, error)
	StaticFS         fs.FS
	Logger           *slog.Logger
	Store            session.Store
	Knowledge        *knowledge.Manager
	Ingester         *knowledge.Ingester
	Retriever        *knowledge.Retriever
	Embedder         embedding.Embedder
	KnowledgeRuntime *knowledge.Runtime
}

// NewServer creates an HTTP server wiring.
// If staticFS is non-nil, the server also serves the embedded frontend SPA.
// If store is non-nil, session endpoints are enabled.
func NewServer(cfg ServerConfig) *Server {
	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}
	svc := service.New(service.ServiceConfig{
		AgentsDir:        cfg.AgentsDir,
		HomeDir:          cfg.HomeDir,
		Registry:         cfg.Registry,
		NewProvider:      cfg.NewProvider,
		Logger:           logger,
		Store:            cfg.Store,
		Knowledge:        cfg.Knowledge,
		Ingester:         cfg.Ingester,
		Retriever:        cfg.Retriever,
		Embedder:         cfg.Embedder,
		KnowledgeRuntime: cfg.KnowledgeRuntime,
	})
	return &Server{
		agentsDir:   cfg.AgentsDir,
		home:        filepath.Dir(cfg.AgentsDir),
		registry:    cfg.Registry,
		newProvider: cfg.NewProvider,
		staticFS:    cfg.StaticFS,
		logger:      logger,
		store:       cfg.Store,
		svc:         svc,
		approver:    NewHTTPApprover(),
		health:      telemetry.NewHealthServer(),
		changeCh:    make(chan agentChange, 16),
	}
}

// Service returns the shared business service.
func (s *Server) Service() *service.Service {
	return s.svc
}

// SetStore sets the session store and updates the shared service.
func (s *Server) SetStore(store session.Store) {
	s.store = store
	s.svc.Store = store
}

// SetRequestLogger sets the LLM request audit logger on the service.
func (s *Server) SetRequestLogger(rl *audit.RequestLogger) {
	s.requestLogger = rl
	s.svc.RequestLogger = rl
}

// SetShutdownCtx sets a context that is cancelled on server shutdown.
// Active agent runs will stop when this context is cancelled.
func (s *Server) SetShutdownCtx(ctx context.Context) {
	s.shutdownCtx = ctx
}

// SetStateDB attaches the SQLite store and initializes JWT services.
func (s *Server) SetStateDB(db *store.DB) error {
	s.stateDB = db
	config.BindDB(db)
	if s.svc != nil {
		s.svc.StateDB = db
	}
	if db == nil {
		s.tokens = nil
		return nil
	}
	secret, err := db.EnsureJWTSecret()
	if err != nil {
		return err
	}
	s.tokens = auth.NewTokenService(secret, s.keyIDActive, s.userIDActive)
	return nil
}

func (s *Server) userIDActive(userID string) bool {
	if s.stateDB == nil || userID == "" {
		return false
	}
	return s.stateDB.HasUser(userID)
}

func (s *Server) keyIDActive(keyID string) bool {
	if s.stateDB != nil && s.stateDB.HasAPIKeyID(keyID) {
		return true
	}
	s.cliKeyMu.RLock()
	defer s.cliKeyMu.RUnlock()
	for _, id := range s.cliKeyHash {
		if id == keyID {
			return true
		}
	}
	return false
}

// SetAPIKey enables authentication with a single ephemeral CLI key.
func (s *Server) SetAPIKey(key string) {
	if key == "" {
		s.SetAPIKeys(nil)
		return
	}
	s.SetAPIKeys([]string{key})
}

// SetAPIKeys replaces ephemeral CLI-sourced API keys (not persisted).
func (s *Server) SetAPIKeys(keys []string) {
	s.cliAPIKeys = append([]string(nil), keys...)
	next := make(map[string]string, len(keys))
	for i, k := range keys {
		if k == "" {
			continue
		}
		next[store.HashAPIKey(k)] = "cli_" + strconv.Itoa(i)
	}
	s.cliKeyMu.Lock()
	s.cliKeyHash = next
	s.cliKeyMu.Unlock()
}

// authEnabled reports whether /v1 requires credentials.
// When a state DB is attached (normal serve), auth is always required so
// unauthenticated clients cannot read data. Tests without a state DB stay open.
func (s *Server) authEnabled() (bool, error) {
	s.cliKeyMu.RLock()
	cliN := len(s.cliKeyHash)
	s.cliKeyMu.RUnlock()
	if cliN > 0 {
		return true, nil
	}
	if s.stateDB != nil {
		return true, nil
	}
	return false, nil
}

// resolveIdentity validates a JWT or raw API key and returns the identity.
// Password sessions get the user's current role from the store (role changes
// take effect immediately; the JWT role claim is never trusted). API keys
// get Role="" and their scopes from the key record.
func (s *Server) resolveIdentity(token string) (auth.Identity, bool) {
	if token == "" {
		return auth.Identity{}, false
	}
	if s.tokens != nil && auth.LooksLikeJWT(token) {
		id, err := s.tokens.Parse(token)
		if err == nil {
			return s.enrichIdentity(id)
		}
	}
	if s.stateDB != nil {
		entry, err := s.stateDB.VerifyAPIKey(token)
		if err == nil && entry != nil {
			return apiKeyIdentity(entry), true
		}
	}
	return s.resolveCLIKey(token)
}

// enrichIdentity fills role/scopes for a parsed JWT identity.
func (s *Server) enrichIdentity(id auth.Identity) (auth.Identity, bool) {
	if id.KeyID == auth.PasswordKeyID {
		if s.stateDB != nil {
			if role, err := s.stateDB.GetUserRole(id.UserID); err == nil {
				id.Role = role
			}
		}
		return id, true
	}
	// Key-backed JWT: load scopes from the key record so scoped keys stay
	// constrained after exchanging via POST /v1/auth/token.
	if s.stateDB != nil {
		if key, err := s.stateDB.GetAPIKey(id.KeyID); err == nil {
			return apiKeyIdentity(&key), true
		}
	}
	return id, true
}

// apiKeyIdentity builds an identity from an API key record.
func apiKeyIdentity(key *store.APIKey) auth.Identity {
	return auth.Identity{
		UserID: key.UserID,
		KeyID:  key.ID,
		Scopes: parseScopes(key.Scopes),
	}
}

// parseScopes splits a comma-separated scope list.
func parseScopes(raw string) []string {
	var out []string
	for _, s := range strings.Split(raw, ",") {
		if s = strings.TrimSpace(s); s != "" {
			out = append(out, s)
		}
	}
	return out
}

func (s *Server) resolveCLIKey(raw string) (auth.Identity, bool) {
	h := store.HashAPIKey(raw)
	s.cliKeyMu.RLock()
	kid, ok := s.cliKeyHash[h]
	s.cliKeyMu.RUnlock()
	if !ok {
		return auth.Identity{}, false
	}
	// Ephemeral CLI keys are operator credentials: full access.
	return auth.Identity{
		UserID: kid,
		KeyID:  kid,
		Role:   store.RoleAdmin,
		Scopes: []string{auth.ScopeAll},
	}, true
}

// SetCatalog sets the provider catalog for listing available providers.
// Also rewires Service.NewProvider so subsequent runs use the latest catalog.
func (s *Server) SetCatalog(c *provider.Catalog) {
	s.catalog = c
	if s.svc != nil {
		if c != nil {
			s.svc.NewProvider = func(a *agent.Agent) (provider.Provider, error) {
				return c.ResolveForAgent(a.Provider)
			}
			s.svc.ModelWindow = c.ModelContextWindow
			s.svc.Catalog = c
		} else {
			s.svc.ModelWindow = nil
			s.svc.Catalog = nil
		}
	}
}

// SetCredentialStore sets the credential store used when saving provider API keys.
func (s *Server) SetCredentialStore(cs *config.CredentialStore) {
	s.creds = cs
	if s.svc != nil {
		s.svc.Creds = cs
	}
}

// StartWatcher begins watching the agents directory for file changes.
// Agent change events are broadcast via the /v1/events SSE endpoint.
func (s *Server) StartWatcher() {
	s.watcher = agent.NewWatcher(s.agentsDir, s.logger, func(change agent.AgentChange) {
		var eventType string
		switch change.Type {
		case agent.ChangeCreated:
			eventType = "agent_created"
		case agent.ChangeUpdated:
			eventType = "agent_updated"
		case agent.ChangeDeleted:
			eventType = "agent_deleted"
		default:
			return
		}
		// Non-blocking send.
		select {
		case s.changeCh <- agentChange{Type: eventType, Name: change.Name}:
		default:
		}
	})
	if err := s.watcher.Start(); err != nil {
		s.logger.Error("log.agent_watcher.start_failed", "error", err)
	}
}

// StopWatcher stops the file watcher.
func (s *Server) StopWatcher() {
	if s.watcher != nil {
		s.watcher.Stop()
	}
}

// Health returns the health server for adding custom checkers.
func (s *Server) Health() *telemetry.HealthServer {
	return s.health
}

// Handler returns the root Gin engine.
func (s *Server) Handler() *gin.Engine {
	if enabled, _ := s.authEnabled(); !enabled {
		s.logger.Warn("log.http.api_key_not_set")
	}

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(LocaleMiddleware())
	r.Use(s.bodySizeLimit(10 << 20)) // 10 MB

	// Health endpoints — always public, no auth required.
	r.GET("/healthz", gin.WrapF(s.health.HandleHealth))
	r.GET("/readyz", gin.WrapF(s.health.HandleReady))
	r.GET("/metrics", gin.WrapF(s.health.HandleMetrics))

	// Build version — public so the UI can render it pre-auth.
	r.GET("/v1/system/version", s.handleVersion)

	// Public auth endpoints (status / register / login / raw-key → JWT).
	r.GET("/v1/auth/status", s.handleAuthStatus)
	r.POST("/v1/auth/register", s.handleAuthRegister)
	r.POST("/v1/auth/login", s.handleAuthLogin)
	r.POST("/v1/auth/token", s.handleAuthToken)

	// API routes — protected when password users or API keys exist.
	s.mountAPIRoutes(r)

	// Static frontend serving (only when embedded).
	if s.staticFS != nil {
		fileServer := http.FileServerFS(s.staticFS)

		r.NoRoute(func(c *gin.Context) {
			path := c.Request.URL.Path
			// API routes — 404.
			if strings.HasPrefix(path, "/v1") || path == "/healthz" {
				c.AbortWithStatus(http.StatusNotFound)
				return
			}
			// Try to serve the file from embedded FS.
			// Strip leading slash for fs lookup.
			name := strings.TrimPrefix(path, "/")
			if name == "" {
				name = "index.html"
			}
			if f, err := s.staticFS.Open(name); err == nil {
				f.Close()
				fileServer.ServeHTTP(c.Writer, c.Request)
				return
			}
			// SPA fallback — serve index.html.
			c.Request.URL.Path = "/"
			fileServer.ServeHTTP(c.Writer, c.Request)
		})
	}

	return r
}
