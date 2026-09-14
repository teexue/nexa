package grpcapi

import (
	"context"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	nexav1 "github.com/teexue/nexa/proto"
)

func TestApprove_NoPending(t *testing.T) {
	client, _, cleanup := setupTestGRPC(t)
	defer cleanup()

	_, err := client.Approve(context.Background(), &nexav1.ApproveRequest{
		ApprovalId: "nonexistent",
		Approved:   true,
	})
	if err == nil {
		t.Fatal("expected error for nonexistent approval")
	}

	st, ok := status.FromError(err)
	if !ok {
		t.Fatal("expected gRPC status error")
	}
	if st.Code() != codes.NotFound {
		t.Errorf("expected NotFound, got %v", st.Code())
	}
}

func TestApprove_MissingID(t *testing.T) {
	client, _, cleanup := setupTestGRPC(t)
	defer cleanup()

	_, err := client.Approve(context.Background(), &nexav1.ApproveRequest{
		Approved: true,
	})
	if err == nil {
		t.Fatal("expected error for missing approval_id")
	}

	st, ok := status.FromError(err)
	if !ok {
		t.Fatal("expected gRPC status error")
	}
	if st.Code() != codes.InvalidArgument {
		t.Errorf("expected InvalidArgument, got %v", st.Code())
	}
}

func TestGRPCApprover_ResolveApproval(t *testing.T) {
	approver := NewGRPCApprover()

	// No pending approval.
	resolved := approver.ResolveApproval("id1", true)
	if resolved {
		t.Error("expected false for non-pending approval")
	}
}
