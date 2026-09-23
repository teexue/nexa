package grpcapi

import (
	"context"
	"log/slog"
	"strings"
	"sync"

	"github.com/teexue/nexakit/provider"
	kitcatalog "github.com/teexue/nexakit/provider/catalog"
	"github.com/teexue/nexakit/registry"
	"github.com/teexue/nexakit/session"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/teexue/nexa/core/agent"
	"github.com/teexue/nexa/core/i18n"
	"github.com/teexue/nexa/core/service"
	"github.com/teexue/nexa/core/telemetry"
	nexav1 "github.com/teexue/nexa/proto"
)

// withRequestLocale attaches an i18n bundle from gRPC metadata.
func withRequestLocale(ctx context.Context) context.Context {
	locale := ""
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		if vals := md.Get("accept-language"); len(vals) > 0 {
			locale = i18n.ParseAcceptLanguage(vals[0])
		}
		if locale == "" {
			if vals := md.Get("locale"); len(vals) > 0 {
				locale = i18n.Normalize(vals[0])
			}
		}
	}
	if locale == "" {
		locale = i18n.Global().Locale()
	}
	bundle, err := i18n.NewBundle(locale)
	if err != nil {
		bundle = i18n.Global()
	}
	return i18n.WithLocale(ctx, bundle)
}

// GRPCServer implements nexav1.AgentServiceServer.
type GRPCServer struct {
	nexav1.UnimplementedAgentServiceServer

	agentsDir   string
	registry    *registry.Registry
	newProvider func(a *agent.Agent) (provider.Provider, error)
	logger      *slog.Logger
	store       session.Store
	svc         *service.Service
	approver    *GRPCApprover
	apiKeysMu   sync.RWMutex
	apiKeys     map[string]struct{}         // when non-empty, all methods require one of these keys
	healthSrv   grpc_health_v1.HealthServer // the registered health service
	health      *telemetry.HealthServer     // optional; nil disables component checks
	healthMu    sync.RWMutex
}

// NewGRPCServer creates a GRPCServer.
func NewGRPCServer(
	agentsDir string,
	reg *registry.Registry,
	newProvider func(a *agent.Agent) (provider.Provider, error),
	logger *slog.Logger,
	store session.Store,
) *GRPCServer {
	svc := service.New(service.ServiceConfig{
		AgentsDir:   agentsDir,
		Registry:    reg,
		NewProvider: newProvider,
		Logger:      logger,
		Store:       store,
	})
	return &GRPCServer{
		agentsDir:   agentsDir,
		registry:    reg,
		newProvider: newProvider,
		logger:      logger,
		store:       store,
		svc:         svc,
		approver:    NewGRPCApprover(),
	}
}

// SetCatalog attaches the provider catalog so PrepareRun enforces enabled models.
func (s *GRPCServer) SetCatalog(c *kitcatalog.Catalog) {
	if s.svc != nil {
		s.svc.Catalog = c
	}
}

// SetHealth sets the health server for component-level readiness checks.
// When set, Check() reports SERVING only when all registered components are healthy.
func (s *GRPCServer) SetHealth(h *telemetry.HealthServer) {
	s.healthMu.Lock()
	defer s.healthMu.Unlock()
	s.health = h
}

// SetAPIKey enables API key authentication with a single key.
// Prefer SetAPIKeys when multiple keys are needed.
func (s *GRPCServer) SetAPIKey(key string) {
	if key == "" {
		s.SetAPIKeys(nil)
		return
	}
	s.SetAPIKeys([]string{key})
}

// SetAPIKeys enables API key authentication for all gRPC methods.
// When non-empty, clients must send a matching key via the
// "authorization" metadata key as "bearer <key>" or "x-api-key".
func (s *GRPCServer) SetAPIKeys(keys []string) {
	next := make(map[string]struct{}, len(keys))
	for _, k := range keys {
		if k != "" {
			next[k] = struct{}{}
		}
	}
	s.apiKeysMu.Lock()
	s.apiKeys = next
	s.apiKeysMu.Unlock()
}

// checkAuth validates the API key from gRPC metadata.
// Returns nil if auth is disabled or the key is valid.
// 遗留：gRPC 鉴权未接入 RBAC（无 role/scope 检查），仅做 API key 校验。
func (s *GRPCServer) checkAuth(ctx context.Context) error {
	s.apiKeysMu.RLock()
	enabled := len(s.apiKeys) > 0
	s.apiKeysMu.RUnlock()
	if !enabled {
		return nil
	}
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return status.Error(codes.Unauthenticated, i18n.TCtx(ctx, "api.grpc.error.missing_metadata"))
	}

	// Check authorization: bearer <key>.
	for _, val := range md.Get("authorization") {
		if len(val) > 7 && strings.EqualFold(val[:7], "bearer ") {
			if s.hasAPIKey(val[7:]) {
				return nil
			}
		}
	}

	// Check x-api-key.
	for _, val := range md.Get("x-api-key") {
		if s.hasAPIKey(val) {
			return nil
		}
	}

	return status.Error(codes.Unauthenticated, i18n.TCtx(ctx, "api.grpc.error.unauthorized"))
}

func (s *GRPCServer) hasAPIKey(key string) bool {
	s.apiKeysMu.RLock()
	defer s.apiKeysMu.RUnlock()
	_, ok := s.apiKeys[key]
	return ok
}

// RegisterServer registers the GRPCServer on the given gRPC server.
func (s *GRPCServer) RegisterServer(srv *grpc.Server) {
	nexav1.RegisterAgentServiceServer(srv, s)

	hs := &grpcHealthService{
		grpcSrv:  s,
		statuses: make(map[string]grpc_health_v1.HealthCheckResponse_ServingStatus),
	}
	hs.statuses[""] = grpc_health_v1.HealthCheckResponse_SERVING
	hs.statuses[nexav1.AgentService_ServiceDesc.ServiceName] = grpc_health_v1.HealthCheckResponse_SERVING
	s.healthSrv = hs

	grpc_health_v1.RegisterHealthServer(srv, hs)
}

// ensure compilation
var _ nexav1.AgentServiceServer = (*GRPCServer)(nil)
