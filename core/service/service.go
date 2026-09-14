// Package service provides shared business logic for HTTP and gRPC transports.
// It eliminates duplication between server/http and server/grpc handlers
// by extracting common agent, session, and run operations.
package service

import (
	"log/slog"

	"github.com/teexue/nexa/core/agent"
	"github.com/teexue/nexa/core/audit"
	"github.com/teexue/nexa/core/config"
	"github.com/teexue/nexakit/embedding"
	"github.com/teexue/nexa/core/knowledge"
	"github.com/teexue/nexakit/provider"
	"github.com/teexue/nexakit/session"
	"github.com/teexue/nexa/core/store"
	"github.com/teexue/nexakit/registry"
)

// Service provides shared operations used by both HTTP and gRPC handlers.
type Service struct {
	AgentsDir   string
	HomeDir     string
	Registry    *registry.Registry
	NewProvider func(a *agent.Agent) (provider.Provider, error)
	Logger      *slog.Logger
	Store       session.Store
	StateDB     *store.DB
	Creds       *config.CredentialStore
	// Catalog, when set, validates run-time model selection against each
	// provider's enabled model list.
	Catalog *provider.Catalog
	// RequestLogger, when set, audits every LLM request/response. Optional.
	RequestLogger *audit.RequestLogger

	// ModelWindow looks up a context window saved on the provider for a model.
	// Optional; nil means no saved windows (fall back to spec / 0).
	ModelWindow func(providerName, model string) int

	Knowledge        *knowledge.Manager
	Ingester         *knowledge.Ingester
	Retriever        *knowledge.Retriever
	Embedder         embedding.Embedder
	KnowledgeRuntime *knowledge.Runtime

	// Hub fans out in-flight HTTP run events so a client can disconnect
	// (navigate away or refresh) without cancelling the loop.
	Hub *RunHub
}

// New creates a Service instance.
func New(cfg ServiceConfig) *Service {
	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}
	return &Service{
		AgentsDir:        cfg.AgentsDir,
		HomeDir:          cfg.HomeDir,
		Registry:         cfg.Registry,
		NewProvider:      cfg.NewProvider,
		Logger:           logger,
		Store:            cfg.Store,
		StateDB:          cfg.StateDB,
		Creds:            cfg.Creds,
		Knowledge:        cfg.Knowledge,
		Ingester:         cfg.Ingester,
		Retriever:        cfg.Retriever,
		Embedder:         cfg.Embedder,
		KnowledgeRuntime: cfg.KnowledgeRuntime,
		Hub:              NewRunHub(),
	}
}

// ServiceConfig holds configuration for creating a Service.
type ServiceConfig struct {
	AgentsDir        string
	HomeDir          string
	Registry         *registry.Registry
	NewProvider      func(a *agent.Agent) (provider.Provider, error)
	Logger           *slog.Logger
	Store            session.Store
	StateDB          *store.DB
	Creds            *config.CredentialStore
	Knowledge        *knowledge.Manager
	Ingester         *knowledge.Ingester
	Retriever        *knowledge.Retriever
	Embedder         embedding.Embedder
	KnowledgeRuntime *knowledge.Runtime
}
