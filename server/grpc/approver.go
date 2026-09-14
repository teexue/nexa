package grpcapi

import (
	"context"
	"sync"

	"github.com/teexue/nexakit/loop"
)

// GRPCApprover implements loop.Approver for gRPC clients.
// When a tool needs approval, the Run stream emits a tool_approval_required
// event and the client calls the Approve RPC to resolve it.
type GRPCApprover struct {
	mu      sync.Mutex
	pending map[string]chan bool
}

// NewGRPCApprover creates a GRPCApprover.
func NewGRPCApprover() *GRPCApprover {
	return &GRPCApprover{
		pending: make(map[string]chan bool),
	}
}

// Approve registers the approval request and blocks until
// ResolveApproval is called or the context is cancelled.
func (a *GRPCApprover) Approve(ctx context.Context, req loop.ApprovalRequest) bool {
	if req.ApprovalID == "" {
		return false
	}

	ch := make(chan bool, 1)

	a.mu.Lock()
	a.pending[req.ApprovalID] = ch
	a.mu.Unlock()

	defer func() {
		a.mu.Lock()
		delete(a.pending, req.ApprovalID)
		a.mu.Unlock()
	}()

	select {
	case <-ctx.Done():
		return false
	case approved := <-ch:
		return approved
	}
}

// ResolveApproval signals the pending approval for a tool.
// Returns true if a pending approval was found and resolved.
func (a *GRPCApprover) ResolveApproval(approvalID string, approved bool) bool {
	if approvalID == "" {
		return false
	}

	a.mu.Lock()
	ch, ok := a.pending[approvalID]
	a.mu.Unlock()

	if !ok {
		return false
	}

	select {
	case ch <- approved:
		return true
	default:
		return false
	}
}
