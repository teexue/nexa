package grpcapi

import (
	"context"
	"io"
	"log/slog"
	"net"
	"os"
	"path/filepath"
	"testing"

	"github.com/teexue/nexakit/provider"
	kitmock "github.com/teexue/nexakit/provider/mock"
	"github.com/teexue/nexakit/registry"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"

	"github.com/teexue/nexa/core/agent"
	nexav1 "github.com/teexue/nexa/proto"
)

func setupTestGRPCWithAuth(t *testing.T, apiKey string) (nexav1.AgentServiceClient, func()) {
	t.Helper()

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
		return &kitmock.MockProvider{
			Calls: [][]kitmock.MockStep{
				{{Text: "test response"}},
			},
		}, nil
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	grpcSrv := NewGRPCServer(dir, reg, newProvider, logger, nil)
	grpcSrv.SetAPIKey(apiKey)
	srv := grpc.NewServer()
	grpcSrv.RegisterServer(srv)

	lis := bufconn.Listen(bufSize)
	go func() {
		_ = srv.Serve(lis)
	}()

	conn, err := grpc.NewClient("passthrough:///bufnet",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
			return lis.DialContext(ctx)
		}),
	)
	if err != nil {
		t.Fatal(err)
	}

	client := nexav1.NewAgentServiceClient(conn)
	cleanup := func() {
		conn.Close()
		srv.Stop()
	}

	return client, cleanup
}

func TestGRPCAuth_NoKey_Unauthenticated(t *testing.T) {
	client, cleanup := setupTestGRPCWithAuth(t, "grpc-secret-key")
	defer cleanup()

	_, err := client.ListTools(context.Background(), &nexav1.ListToolsRequest{})
	if err == nil {
		t.Fatal("expected error without API key")
	}

	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("expected gRPC status error, got %v", err)
	}
	if st.Code() != codes.Unauthenticated {
		t.Errorf("expected Unauthenticated, got %v", st.Code())
	}
}

func TestGRPCAuth_WrongKey_Unauthenticated(t *testing.T) {
	client, cleanup := setupTestGRPCWithAuth(t, "grpc-secret-key")
	defer cleanup()

	ctx := metadata.AppendToOutgoingContext(context.Background(), "authorization", "bearer wrong-key")
	_, err := client.ListTools(ctx, &nexav1.ListToolsRequest{})
	if err == nil {
		t.Fatal("expected error with wrong API key")
	}

	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("expected gRPC status error, got %v", err)
	}
	if st.Code() != codes.Unauthenticated {
		t.Errorf("expected Unauthenticated, got %v", st.Code())
	}
}

func TestGRPCAuth_CorrectKey_Bearer(t *testing.T) {
	client, cleanup := setupTestGRPCWithAuth(t, "grpc-secret-key")
	defer cleanup()

	ctx := metadata.AppendToOutgoingContext(context.Background(), "authorization", "bearer grpc-secret-key")
	resp, err := client.ListTools(ctx, &nexav1.ListToolsRequest{})
	if err != nil {
		t.Fatalf("expected success with correct Bearer key, got: %v", err)
	}
	if len(resp.Tools) == 0 {
		t.Fatal("expected at least one tool")
	}
}

func TestGRPCAuth_CorrectKey_XAPIKey(t *testing.T) {
	client, cleanup := setupTestGRPCWithAuth(t, "grpc-secret-key")
	defer cleanup()

	ctx := metadata.AppendToOutgoingContext(context.Background(), "x-api-key", "grpc-secret-key")
	resp, err := client.ListAgents(ctx, &nexav1.ListAgentsRequest{})
	if err != nil {
		t.Fatalf("expected success with correct X-API-Key, got: %v", err)
	}
	if len(resp.Agents) == 0 {
		t.Fatal("expected at least one agent")
	}
}

func TestGRPCAuth_RunStreaming_RequiresKey(t *testing.T) {
	client, cleanup := setupTestGRPCWithAuth(t, "grpc-secret-key")
	defer cleanup()

	// Without key → error.
	stream, err := client.Run(context.Background(), &nexav1.RunRequest{
		Agent:  "test",
		Prompt: "hello",
	})
	if err != nil {
		t.Fatalf("unexpected error creating stream: %v", err)
	}
	_, err = stream.Recv()
	if err == nil {
		t.Fatal("expected error receiving without key")
	}
	st, ok := status.FromError(err)
	if !ok || st.Code() != codes.Unauthenticated {
		t.Errorf("expected Unauthenticated, got %v", err)
	}

	// With correct key → success.
	ctx := metadata.AppendToOutgoingContext(context.Background(), "authorization", "bearer grpc-secret-key")
	stream, err = client.Run(ctx, &nexav1.RunRequest{
		Agent:  "test",
		Prompt: "hello",
	})
	if err != nil {
		t.Fatalf("unexpected error creating stream with key: %v", err)
	}
	ev, err := stream.Recv()
	if err != nil {
		t.Fatalf("expected to receive event with key, got: %v", err)
	}
	if ev == nil {
		t.Fatal("expected non-nil event")
	}
}
