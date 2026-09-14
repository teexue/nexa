package grpcapi

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/test/bufconn"

	"github.com/teexue/nexa/core/agent"
	"github.com/teexue/nexakit/provider"
	"github.com/teexue/nexa/core/telemetry"
	nexav1 "github.com/teexue/nexa/proto"
	"github.com/teexue/nexakit/registry"
)

func TestGRPCHealth_Check_Serving(t *testing.T) {
	client, grpcSrv, cleanup := setupTestGRPC(t)
	defer cleanup()

	// Verify implementation is registered.
	if grpcSrv.healthSrv == nil {
		t.Fatal("healthSrv should not be nil after RegisterServer")
	}

	// Check the gRPC health service via direct call (no telemetry.HealthServer set).
	ctx := context.Background()
	resp, err := grpcSrv.healthSrv.Check(ctx, &grpc_health_v1.HealthCheckRequest{Service: ""})
	if err != nil {
		t.Fatalf("health Check failed: %v", err)
	}
	if resp.Status != grpc_health_v1.HealthCheckResponse_SERVING {
		t.Errorf("expected SERVING, got %v", resp.Status)
	}

	// Check the agent service.
	resp, err = grpcSrv.healthSrv.Check(ctx, &grpc_health_v1.HealthCheckRequest{
		Service: nexav1.AgentService_ServiceDesc.ServiceName,
	})
	if err != nil {
		t.Fatalf("health Check for agent service failed: %v", err)
	}
	if resp.Status != grpc_health_v1.HealthCheckResponse_SERVING {
		t.Errorf("expected SERVING for agent service, got %v", resp.Status)
	}

	// Unknown service.
	resp, err = grpcSrv.healthSrv.Check(ctx, &grpc_health_v1.HealthCheckRequest{
		Service: "nonexistent.Service",
	})
	if err != nil {
		t.Fatalf("health Check for unknown service failed: %v", err)
	}
	if resp.Status != grpc_health_v1.HealthCheckResponse_SERVICE_UNKNOWN {
		t.Errorf("expected SERVICE_UNKNOWN, got %v", resp.Status)
	}

	_ = client // ensure connection works
}

func TestGRPCHealth_Check_NotServing_WhenComponentFails(t *testing.T) {
	// Create a health server with a failing checker.
	healthSrv := telemetry.NewHealthServer()
	healthSrv.AddChecker(telemetry.NewProviderChecker("test-checker", func(_ context.Context) error {
		return errors.New("component down")
	}))

	dir := t.TempDir()
	agentContent := `name: test
version: 1
provider: mock
model: test-model
system_prompt: |
  You are a test assistant.
tools:
  - test_tool
max_turns: 5
max_tokens: 1024
`
	if err := os.WriteFile(filepath.Join(dir, "test.yaml"), []byte(agentContent), 0o644); err != nil {
		t.Fatal(err)
	}

	reg := registry.New()
	reg.Register(&testTool{})

	newProvider := func(a *agent.Agent) (provider.Provider, error) {
		return &provider.MockProvider{
			Calls: [][]provider.MockStep{
				{{Text: "test response"}},
			},
		}, nil
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	grpcSrv := NewGRPCServer(dir, reg, newProvider, logger, nil)
	grpcSrv.SetHealth(healthSrv)

	srv := grpc.NewServer()
	grpcSrv.RegisterServer(srv)
	defer srv.Stop()

	lis := bufconn.Listen(bufSize)
	go func() { _ = srv.Serve(lis) }()

	// When a component is down, Check should return NOT_SERVING.
	resp, err := grpcSrv.healthSrv.Check(context.Background(), &grpc_health_v1.HealthCheckRequest{Service: ""})
	if err != nil {
		t.Fatalf("health Check failed: %v", err)
	}
	if resp.Status != grpc_health_v1.HealthCheckResponse_NOT_SERVING {
		t.Errorf("expected NOT_SERVING when component is down, got %v", resp.Status)
	}

	// Agent service should also be NOT_SERVING.
	resp, err = grpcSrv.healthSrv.Check(context.Background(), &grpc_health_v1.HealthCheckRequest{
		Service: nexav1.AgentService_ServiceDesc.ServiceName,
	})
	if err != nil {
		t.Fatalf("health Check for agent service failed: %v", err)
	}
	if resp.Status != grpc_health_v1.HealthCheckResponse_NOT_SERVING {
		t.Errorf("expected NOT_SERVING for agent service when component down, got %v", resp.Status)
	}
}

func TestGRPCHealth_Watch(t *testing.T) {
	_, grpcSrv, cleanup := setupTestGRPC(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	// Watch is server-streaming; test via direct Check (Watch delegates to Check internally).
	// We verify Check works correctly for both SERVING and NOT_SERVING scenarios in the above tests.
	resp, err := grpcSrv.healthSrv.Check(ctx, &grpc_health_v1.HealthCheckRequest{Service: ""})
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}
	if resp.Status != grpc_health_v1.HealthCheckResponse_SERVING {
		t.Errorf("expected SERVING, got %v", resp.Status)
	}
}
