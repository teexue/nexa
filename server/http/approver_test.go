package httpapi

import (
	"context"
	"testing"
	"time"

	"github.com/teexue/nexakit/loop"
)

func TestHTTPApprover_ApproveAndResolve(t *testing.T) {
	a := NewHTTPApprover()
	ctx := context.Background()

	// Simulate two concurrent approval requests for the same tool name.
	req1 := loop.ApprovalRequest{Tool: "read_file", Arguments: []byte(`{"path":"a.txt"}`), ApprovalID: "appr-1"}
	req2 := loop.ApprovalRequest{Tool: "read_file", Arguments: []byte(`{"path":"b.txt"}`), ApprovalID: "appr-2"}

	ch1 := make(chan bool, 1)
	ch2 := make(chan bool, 1)

	go func() { ch1 <- a.Approve(ctx, req1) }()
	go func() { ch2 <- a.Approve(ctx, req2) }()

	// Give goroutines time to register.
	time.Sleep(50 * time.Millisecond)

	if !a.HasPending() {
		t.Fatal("expected pending approvals")
	}

	// Resolve the second approval (deny) first to prove IDs are independent.
	if !a.ResolveApproval(req2.ApprovalID, false) {
		t.Fatal("failed to resolve second approval")
	}
	if !a.ResolveApproval(req1.ApprovalID, true) {
		t.Fatal("failed to resolve first approval")
	}

	select {
	case approved := <-ch1:
		if !approved {
			t.Fatal("expected first approval to be approved")
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for first approval")
	}

	select {
	case approved := <-ch2:
		if approved {
			t.Fatal("expected second approval to be denied")
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for second approval")
	}
}

func TestHTTPApprover_EmptyApprovalID(t *testing.T) {
	a := NewHTTPApprover()
	ctx := context.Background()

	req := loop.ApprovalRequest{Tool: "read_file", ApprovalID: ""}
	if a.Approve(ctx, req) {
		t.Fatal("expected approval without ID to be denied")
	}

	if a.ResolveApproval("", true) {
		t.Fatal("expected resolving empty ID to fail")
	}
}

func TestHTTPApprover_ResolveUnknownID(t *testing.T) {
	a := NewHTTPApprover()
	if a.ResolveApproval("does-not-exist", true) {
		t.Fatal("expected resolving unknown ID to fail")
	}
}

func TestHTTPApprover_ContextCancellation(t *testing.T) {
	a := NewHTTPApprover()
	ctx, cancel := context.WithCancel(context.Background())

	req := loop.ApprovalRequest{Tool: "read_file", ApprovalID: "appr-cancel"}

	ch := make(chan bool, 1)
	go func() { ch <- a.Approve(ctx, req) }()

	time.Sleep(50 * time.Millisecond)
	cancel()

	select {
	case approved := <-ch:
		if approved {
			t.Fatal("expected cancellation to deny approval")
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for cancellation")
	}
}
