package grpcapi

import (
	"context"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/teexue/nexakit/session"
	nexav1 "github.com/teexue/nexa/proto"
)

func TestListSessions_NotConfigured(t *testing.T) {
	client, _, cleanup := setupTestGRPC(t)
	defer cleanup()

	_, err := client.ListSessions(context.Background(), &nexav1.ListSessionsRequest{})
	if err == nil {
		t.Fatal("expected error when store not configured")
	}

	st, ok := status.FromError(err)
	if !ok {
		t.Fatal("expected gRPC status error")
	}
	if st.Code() != codes.FailedPrecondition {
		t.Errorf("expected FailedPrecondition, got %v", st.Code())
	}
}

func TestListSessions_WithStore(t *testing.T) {
	client, _, store, cleanup := setupTestGRPCWithStore(t)
	defer cleanup()

	// Create a session.
	sess := session.New("test")
	if err := store.Save(sess); err != nil {
		t.Fatal(err)
	}

	resp, err := client.ListSessions(context.Background(), &nexav1.ListSessionsRequest{})
	if err != nil {
		t.Fatal(err)
	}

	if len(resp.Sessions) == 0 {
		t.Fatal("expected at least one session")
	}

	found := false
	for _, s := range resp.Sessions {
		if s.Id == sess.ID {
			found = true
		}
	}
	if !found {
		t.Error("expected session in list")
	}
}

func TestDeleteSession_NotFound(t *testing.T) {
	client, _, _, cleanup := setupTestGRPCWithStore(t)
	defer cleanup()

	_, err := client.DeleteSession(context.Background(), &nexav1.DeleteSessionRequest{Id: "nonexistent"})
	if err == nil {
		t.Fatal("expected error for nonexistent session")
	}

	st, ok := status.FromError(err)
	if !ok {
		t.Fatal("expected gRPC status error")
	}
	if st.Code() != codes.NotFound {
		t.Errorf("expected NotFound, got %v", st.Code())
	}
}
