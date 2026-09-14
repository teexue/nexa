package grpcapi

import (
	"context"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	nexav1 "github.com/teexue/nexa/proto"
)

func TestListAgents(t *testing.T) {
	client, _, cleanup := setupTestGRPC(t)
	defer cleanup()

	resp, err := client.ListAgents(context.Background(), &nexav1.ListAgentsRequest{})
	if err != nil {
		t.Fatal(err)
	}

	if len(resp.Agents) == 0 {
		t.Fatal("expected at least one agent")
	}

	found := false
	for _, a := range resp.Agents {
		if a.Name == "test" {
			found = true
			if a.Provider != "mock" {
				t.Errorf("expected provider 'mock', got %q", a.Provider)
			}
		}
	}
	if !found {
		t.Error("expected agent 'test' in list")
	}
}

func TestGetAgent(t *testing.T) {
	client, _, cleanup := setupTestGRPC(t)
	defer cleanup()

	resp, err := client.GetAgent(context.Background(), &nexav1.GetAgentRequest{Name: "test"})
	if err != nil {
		t.Fatal(err)
	}

	if resp.Name != "test" {
		t.Errorf("expected name 'test', got %q", resp.Name)
	}
	if resp.Provider != "mock" {
		t.Errorf("expected provider 'mock', got %q", resp.Provider)
	}
	if resp.Model != "test-model" {
		t.Errorf("expected model 'test-model', got %q", resp.Model)
	}
}

func TestGetAgent_NotFound(t *testing.T) {
	client, _, cleanup := setupTestGRPC(t)
	defer cleanup()

	_, err := client.GetAgent(context.Background(), &nexav1.GetAgentRequest{Name: "nonexistent"})
	if err == nil {
		t.Fatal("expected error for missing agent")
	}

	st, ok := status.FromError(err)
	if !ok {
		t.Fatal("expected gRPC status error")
	}
	if st.Code() != codes.NotFound {
		t.Errorf("expected NotFound, got %v", st.Code())
	}
}
