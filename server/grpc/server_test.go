package grpcapi

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net"
	"os"
	"path/filepath"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"

	"github.com/teexue/common-agent/core/agent"
	"github.com/teexue/nexakit/provider"
	"github.com/teexue/nexakit/session"
	"github.com/teexue/nexakit/tool"
	commonagentv1 "github.com/teexue/common-agent/proto"
	"github.com/teexue/nexakit/registry"
)

const bufSize = 1024 * 1024

// testTool is a minimal tool for testing.
type testTool struct{}

func (t *testTool) Name() string        { return "test_tool" }
func (t *testTool) Description() string { return "A test tool" }
func (t *testTool) InputSchema() map[string]any {
	return map[string]any{"type": "object", "properties": map[string]any{}}
}
func (t *testTool) Execute(_ context.Context, _ json.RawMessage) (tool.Result, error) {
	return tool.Result{Output: json.RawMessage(`"ok"`)}, nil
}

func setupTestGRPC(t *testing.T) (commonagentv1.AgentServiceClient, *GRPCServer, func()) {
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
		return &provider.MockProvider{
			Calls: [][]provider.MockStep{
				{{Text: "test response"}},
			},
		}, nil
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	grpcSrv := NewGRPCServer(dir, reg, newProvider, logger, nil)
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

	client := commonagentv1.NewAgentServiceClient(conn)
	cleanup := func() {
		conn.Close()
		srv.Stop()
	}

	return client, grpcSrv, cleanup
}

func setupTestGRPCWithStore(t *testing.T) (commonagentv1.AgentServiceClient, *GRPCServer, session.Store, func()) {
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
		return &provider.MockProvider{
			Calls: [][]provider.MockStep{
				{{Text: "test response"}},
			},
		}, nil
	}

	store, err := session.NewFileStore(filepath.Join(dir, "sessions"))
	if err != nil {
		t.Fatal(err)
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	grpcSrv := NewGRPCServer(dir, reg, newProvider, logger, store)
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

	client := commonagentv1.NewAgentServiceClient(conn)
	cleanup := func() {
		conn.Close()
		srv.Stop()
	}

	return client, grpcSrv, store, cleanup
}

func TestRun_BasicStream(t *testing.T) {
	client, _, cleanup := setupTestGRPC(t)
	defer cleanup()

	stream, err := client.Run(context.Background(), &commonagentv1.RunRequest{
		Agent:  "test",
		Prompt: "hello",
	})
	if err != nil {
		t.Fatal(err)
	}

	var events []*commonagentv1.AgentEvent
	for {
		ev, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		events = append(events, ev)
	}

	if len(events) == 0 {
		t.Fatal("expected at least one event")
	}

	// Last event should be done.
	last := events[len(events)-1]
	if last.Type != commonagentv1.EventType_EVENT_TYPE_DONE {
		t.Errorf("expected last event type DONE, got %v", last.Type)
	}
}

func TestRun_MissingAgent(t *testing.T) {
	client, _, cleanup := setupTestGRPC(t)
	defer cleanup()

	stream, err := client.Run(context.Background(), &commonagentv1.RunRequest{
		Agent:  "nonexistent",
		Prompt: "hello",
	})
	if err != nil {
		// Some gRPC versions return the error on the Run call itself.
		st, ok := status.FromError(err)
		if ok && st.Code() == codes.InvalidArgument {
			return
		}
		t.Fatalf("unexpected error: %v", err)
	}

	// The error is received on the first Recv for server-streaming RPCs.
	_, err = stream.Recv()
	if err == nil {
		t.Fatal("expected error for missing agent")
	}

	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("expected gRPC status error, got %v", err)
	}
	if st.Code() != codes.InvalidArgument {
		t.Errorf("expected InvalidArgument, got %v", st.Code())
	}
}

func TestRun_MissingPrompt(t *testing.T) {
	client, _, cleanup := setupTestGRPC(t)
	defer cleanup()

	stream, err := client.Run(context.Background(), &commonagentv1.RunRequest{
		Agent: "test",
	})
	if err != nil {
		st, ok := status.FromError(err)
		if ok && st.Code() == codes.InvalidArgument {
			return
		}
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = stream.Recv()
	if err == nil {
		t.Fatal("expected error for missing prompt")
	}

	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("expected gRPC status error, got %v", err)
	}
	if st.Code() != codes.InvalidArgument {
		t.Errorf("expected InvalidArgument, got %v", st.Code())
	}
}

func TestRun_SessionResume_NotConfigured(t *testing.T) {
	client, _, cleanup := setupTestGRPC(t)
	defer cleanup()

	stream, err := client.Run(context.Background(), &commonagentv1.RunRequest{
		Agent:     "test",
		Prompt:    "hello",
		SessionId: "some-id",
	})
	if err != nil {
		st, ok := status.FromError(err)
		if ok && st.Code() == codes.FailedPrecondition {
			return
		}
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = stream.Recv()
	if err == nil {
		t.Fatal("expected error when session store not configured")
	}

	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("expected gRPC status error, got %v", err)
	}
	if st.Code() != codes.FailedPrecondition {
		t.Errorf("expected FailedPrecondition, got %v", st.Code())
	}
}

func TestListTools(t *testing.T) {
	client, _, cleanup := setupTestGRPC(t)
	defer cleanup()

	resp, err := client.ListTools(context.Background(), &commonagentv1.ListToolsRequest{})
	if err != nil {
		t.Fatal(err)
	}

	if len(resp.Tools) == 0 {
		t.Fatal("expected at least one tool")
	}

	found := false
	for _, tl := range resp.Tools {
		if tl.Name == "test_tool" {
			found = true
			if tl.Description != "A test tool" {
				t.Errorf("expected description 'A test tool', got %q", tl.Description)
			}
		}
	}
	if !found {
		t.Error("expected test_tool in list")
	}
}

func TestEventToProto_And_Back(t *testing.T) {
	original := commonagentv1.AgentEvent{
		Type:       commonagentv1.EventType_EVENT_TYPE_TEXT_DELTA,
		Content:    "hello",
		Tool:       "",
		Input:      nil,
		ToolCallId: "",
		ApprovalId: "",
		Output:     nil,
		Code:       "",
		Message:    "",
		Status:     "",
		Turns:      0,
	}

	// Verify round-trip through our conversion functions.
	converted := EventToProto(ProtoToEvent(&original))
	if converted.Type != original.Type {
		t.Errorf("type mismatch: %v vs %v", converted.Type, original.Type)
	}
	if converted.Content != original.Content {
		t.Errorf("content mismatch: %q vs %q", converted.Content, original.Content)
	}
}
