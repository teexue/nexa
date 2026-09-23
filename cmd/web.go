package main

import (
	"context"
	"flag"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/teexue/nexakit/embedding"
	kitcatalog "github.com/teexue/nexakit/provider/catalog"
	"github.com/teexue/nexakit/registry"
	"github.com/teexue/nexakit/session"
	"google.golang.org/grpc"

	"github.com/teexue/nexa/core/audit"
	"github.com/teexue/nexa/core/config"
	"github.com/teexue/nexa/core/i18n"
	"github.com/teexue/nexa/core/kanban"
	"github.com/teexue/nexa/core/knowledge"
	"github.com/teexue/nexa/core/store"
	"github.com/teexue/nexa/core/telemetry"
	grpcapi "github.com/teexue/nexa/server/grpc"
	httpapi "github.com/teexue/nexa/server/http"
)

type webFlags struct {
	addr, grpcAddr, home, locale string
	mock                         bool
	apiKeys                      []string
}

func parseWebFlags(args []string) webFlags {
	fs := flag.NewFlagSet("web", flag.ExitOnError)
	addr := fs.String("addr", ":8080", i18n.T("cli.flag.addr"))
	grpcAddr := fs.String("grpc-addr", "", i18n.T("cli.flag.grpc_addr"))
	homeFlag := fs.String("home", "", i18n.T("cli.flag.home"))
	localeFlag := fs.String("locale", "", i18n.T("cli.flag.locale"))
	mock := fs.Bool("mock", false, i18n.T("cli.flag.mock"))
	var apiKeys stringList
	fs.Var(&apiKeys, "api-key", i18n.T("cli.flag.api_key"))
	_ = fs.Parse(args)
	return webFlags{
		addr: *addr, grpcAddr: *grpcAddr, home: *homeFlag,
		locale: *localeFlag, mock: *mock, apiKeys: []string(apiKeys),
	}
}

type knowledgeRuntime struct {
	mgr *knowledge.Manager
	rt  *knowledge.Runtime
	emb embedding.Embedder
}

type knowledgeInit struct {
	paths    runtimePaths
	settings config.Settings
	creds    *config.CredentialStore
	mock     bool
	logger   *slog.Logger
	reg      *registry.Registry
}

func initKnowledge(cfg knowledgeInit) knowledgeRuntime {
	if cfg.mock {
		return knowledgeRuntime{}
	}
	kbMgr, err := knowledge.NewManager(config.KnowledgeDir(cfg.paths.home))
	if err != nil {
		cfg.logger.Error("log.knowledge.open", "error", err)
		os.Exit(1)
	}
	var emb embedding.Embedder
	if cfg.settings.Embedding != nil {
		lookup := func(k string) string { return os.Getenv(k) }
		if cfg.creds != nil {
			lookup = cfg.creds.Lookup
		}
		emb, err = embedding.New(*cfg.settings.Embedding, lookup)
		if err != nil {
			cfg.logger.Warn("log.embedding.init_failed", "error", err)
		}
	}
	kbRT := knowledge.NewRuntime(kbMgr, emb)
	knowledge.RegisterKnowledge(cfg.reg, kbRT)
	return knowledgeRuntime{mgr: kbMgr, rt: kbRT, emb: emb}
}

type webHTTPConfig struct {
	paths     runtimePaths
	reg       *registry.Registry
	catalog   *kitcatalog.Catalog
	creds     *config.CredentialStore
	stateDB   *store.DB
	sessStore session.Store
	kb        knowledgeRuntime
	logger    *slog.Logger
	mock      bool
	apiKeys   []string
}

func wireWebHTTP(cfg webHTTPConfig) *httpapi.Server {
	srv := httpapi.NewServer(httpapi.ServerConfig{
		AgentsDir:        cfg.paths.agentsDir,
		HomeDir:          cfg.paths.home,
		Registry:         cfg.reg,
		NewProvider:      resolveProvider(cfg.catalog, cfg.mock),
		StaticFS:         distFS(),
		Logger:           cfg.logger,
		Store:            cfg.sessStore,
		Knowledge:        cfg.kb.mgr,
		Embedder:         cfg.kb.emb,
		KnowledgeRuntime: cfg.kb.rt,
	})
	if cfg.kb.rt != nil {
		srv.Service().Ingester = cfg.kb.rt.CurrentIngester()
		srv.Service().Retriever = cfg.kb.rt.CurrentRetriever()
	}
	srv.SetRequestLogger(audit.NewRequestLogger(filepath.Join(cfg.paths.home, "audit", "requests")))
	if cfg.catalog != nil {
		srv.SetCatalog(cfg.catalog)
	}
	if cfg.creds != nil {
		srv.SetCredentialStore(cfg.creds)
	}
	if cfg.stateDB != nil {
		if err := srv.SetStateDB(cfg.stateDB); err != nil {
			cfg.logger.Error("log.config.load_api_keys", "error", err)
			os.Exit(1)
		}
	}
	if len(cfg.apiKeys) > 0 {
		srv.SetAPIKeys(cfg.apiKeys)
	}
	srv.StartWatcher()
	return srv
}

type webServeConfig struct {
	addr, grpcAddr string
	srv            *httpapi.Server
	grpc           GRPCConfig
	logger         *slog.Logger
	stateDB        *store.DB
}

func serveAndWait(cfg webServeConfig) {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	cfg.srv.SetShutdownCtx(ctx)
	if cfg.stateDB != nil {
		worker := kanban.NewWorker(kanban.WorkerConfig{
			Store:     cfg.stateDB,
			Runner:    cfg.srv.Service().KanbanRunner(),
			Logger:    cfg.logger,
			TickEvery: 5 * time.Second,
		})
		worker.Start(ctx)
		defer worker.Stop()
	}
	httpServer := &http.Server{
		Addr:              cfg.addr,
		Handler:           cfg.srv.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
	}
	var grpcServer *grpc.Server
	if cfg.grpcAddr != "" {
		grpcServer = startGRPCServer(cfg.grpc)
	}
	go func() {
		cfg.logger.Info("log.http.listening", "addr", cfg.addr, "home", cfg.grpc.Paths.home)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			cfg.logger.Error("log.http.server_failed", "error", err)
			os.Exit(1)
		}
	}()
	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if grpcServer != nil {
		grpcServer.GracefulStop()
	}
	_ = httpServer.Shutdown(shutdownCtx)
}

func runWeb(args []string, logger *slog.Logger) {
	opts := parseWebFlags(args)
	paths, err := resolvePaths(opts.home)
	if err != nil {
		logger.Error("log.cmd.resolve_paths", "error", err)
		os.Exit(1)
	}
	catalog, creds, stateDB, err := bootstrapRuntime(paths, opts.mock, logger)
	if err != nil {
		logger.Error("log.cmd.bootstrap", "error", err)
		os.Exit(1)
	}
	if stateDB != nil {
		defer stateDB.Close()
	}
	settings, err := config.LoadSettings(paths.home)
	if err != nil {
		logger.Error("log.config.load_settings", "error", err)
		os.Exit(1)
	}
	logger = newLocaleLogger(opts.locale, settings.Locale)
	reg := newRegistry("")
	kb := initKnowledge(knowledgeInit{
		paths: paths, settings: settings, creds: creds,
		mock: opts.mock, logger: logger, reg: reg,
	})
	var sessStore session.Store
	if stateDB != nil {
		sessStore = store.NewSessionStore(stateDB)
	}
	srv := wireWebHTTP(webHTTPConfig{
		paths: paths, reg: reg, catalog: catalog, creds: creds,
		stateDB: stateDB, sessStore: sessStore, kb: kb, logger: logger,
		mock: opts.mock, apiKeys: opts.apiKeys,
	})
	serveAndWait(webServeConfig{
		addr: opts.addr, grpcAddr: opts.grpcAddr, srv: srv, logger: logger, stateDB: stateDB,
		grpc: GRPCConfig{
			Addr: opts.grpcAddr, Paths: paths, Reg: reg, Catalog: catalog, Mock: opts.mock,
			Logger: logger, SessStore: sessStore, Health: srv.Health(), APIKeys: opts.apiKeys,
		},
	})
}

// GRPCConfig holds configuration for starting a gRPC server.
type GRPCConfig struct {
	Addr      string
	Paths     runtimePaths
	Reg       *registry.Registry
	Catalog   *kitcatalog.Catalog
	Mock      bool
	Logger    *slog.Logger
	SessStore session.Store
	Health    *telemetry.HealthServer
	APIKeys   []string
}

// startGRPCServer creates, registers, and starts a gRPC server in a goroutine.
func startGRPCServer(cfg GRPCConfig) *grpc.Server {
	grpcSrv := grpcapi.NewGRPCServer(cfg.Paths.agentsDir, cfg.Reg, resolveProvider(cfg.Catalog, cfg.Mock), cfg.Logger, cfg.SessStore)
	grpcSrv.SetCatalog(cfg.Catalog)
	if cfg.Health != nil {
		grpcSrv.SetHealth(cfg.Health)
	}
	if len(cfg.APIKeys) > 0 {
		grpcSrv.SetAPIKeys(cfg.APIKeys)
	}
	srv := grpc.NewServer()
	grpcSrv.RegisterServer(srv)

	lis, err := net.Listen("tcp", cfg.Addr)
	if err != nil {
		cfg.Logger.Error("log.grpc.listen_failed", "error", err)
		os.Exit(1)
	}

	go func() {
		cfg.Logger.Info("log.grpc.listening", "addr", cfg.Addr, "home", cfg.Paths.home)
		if err := srv.Serve(lis); err != nil {
			cfg.Logger.Error("log.grpc.server_failed", "error", err)
			os.Exit(1)
		}
	}()
	return srv
}
