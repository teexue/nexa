package main

import (
	"log/slog"
	"os"
	"path/filepath"

	"github.com/teexue/nexa/core/audit"
	"github.com/teexue/nexa/core/config"
	"github.com/teexue/nexa/core/knowledge"
	"github.com/teexue/nexa/core/service"
	"github.com/teexue/nexa/core/store"
	"github.com/teexue/nexakit/embedding"
	"github.com/teexue/nexakit/provider"
	"github.com/teexue/nexakit/registry"
	"github.com/teexue/nexakit/session"
)

// cliServiceConfig holds everything needed to assemble the shared Service for
// run / chat. It mirrors what runWeb wires for the HTTP server so CLI runs go
// through the same PrepareRun path as the UI.
type cliServiceConfig struct {
	paths    runtimePaths
	reg      *registry.Registry
	catalog  *provider.Catalog
	creds    *config.CredentialStore
	stateDB  *store.DB
	settings config.Settings
	mock     bool
	logger   *slog.Logger
}

// wireCLIService assembles the shared service.Service for CLI runs.
// The returned service owns Policy, Store, MCP, skills, model lock, prompt
// optimization and subagent wiring via PrepareRun — none of that may be
// duplicated in cmd.
func wireCLIService(cfg cliServiceConfig) *service.Service {
	var sessStore session.Store
	if cfg.stateDB != nil {
		sessStore = store.NewSessionStore(cfg.stateDB)
	}
	svc := service.New(service.ServiceConfig{
		AgentsDir:   cfg.paths.agentsDir,
		HomeDir:     cfg.paths.home,
		Registry:    cfg.reg,
		NewProvider: resolveProvider(cfg.catalog, cfg.mock),
		Logger:      cfg.logger,
		Store:       sessStore,
		StateDB:     cfg.stateDB,
		Creds:       cfg.creds,
	})
	svc.RequestLogger = audit.NewRequestLogger(auditDir(cfg.paths.home))
	if cfg.catalog != nil {
		svc.Catalog = cfg.catalog
		svc.ModelWindow = cfg.catalog.ModelContextWindow
	}
	wireCLIKnowledge(svc, cfg)
	return svc
}

// wireCLIKnowledge registers knowledge tools backed by the user's home
// directory, mirroring the server wiring so CLI runs support agents that
// reference them. Non-fatal: failures are logged and skipped.
func wireCLIKnowledge(svc *service.Service, cfg cliServiceConfig) {
	if cfg.mock {
		return
	}
	kbMgr, err := knowledge.NewManager(config.KnowledgeDir(cfg.paths.home))
	if err != nil {
		cfg.logger.Warn("log.knowledge.open", "error", err)
		return
	}
	emb := cliEmbedder(cfg)
	rt := knowledge.NewRuntime(kbMgr, emb)
	knowledge.RegisterKnowledge(cfg.reg, rt)
	svc.Knowledge = kbMgr
	svc.Embedder = emb
	svc.KnowledgeRuntime = rt
	if rt != nil {
		svc.Ingester = rt.CurrentIngester()
		svc.Retriever = rt.CurrentRetriever()
	}
}

// cliEmbedder builds the embedding client from user settings; nil when unset.
func cliEmbedder(cfg cliServiceConfig) embedding.Embedder {
	if cfg.settings.Embedding == nil {
		return nil
	}
	lookup := embedding.KeyLookup(os.Getenv)
	if cfg.creds != nil {
		lookup = cfg.creds.Lookup
	}
	emb, err := embedding.New(*cfg.settings.Embedding, lookup)
	if err != nil {
		cfg.logger.Warn("log.embedding.init_failed", "error", err)
		return nil
	}
	return emb
}

// auditDir returns the LLM request audit directory under home.
func auditDir(home string) string {
	return filepath.Join(home, "audit", "requests")
}
