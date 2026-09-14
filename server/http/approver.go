package httpapi

import (
	"context"
	"sync"

	"github.com/teexue/nexakit/loop"
)

// HTTPApprover implements loop.Approver for HTTP/SSE clients.
// When a tool needs approval, it emits an event and waits for
// the frontend to call Approve/Deny via the API.
type HTTPApprover struct {
	mu      sync.Mutex
	pending map[string]chan bool // approval ID → approval channel
}

// NewHTTPApprover creates an HTTPApprover.
func NewHTTPApprover() *HTTPApprover {
	return &HTTPApprover{
		pending: make(map[string]chan bool),
	}
}

// Approve emits a tool_approval_required event and blocks until
// the frontend responds via ResolveApproval or the context is cancelled.
func (a *HTTPApprover) Approve(ctx context.Context, req loop.ApprovalRequest) bool {
	if req.ApprovalID == "" {
		// Without a stable approval ID we cannot safely correlate the response.
		return false
	}

	ch := make(chan bool, 1)

	a.mu.Lock()
	a.pending[req.ApprovalID] = ch
	a.mu.Unlock()

	// Clean up on exit.
	defer func() {
		a.mu.Lock()
		delete(a.pending, req.ApprovalID)
		a.mu.Unlock()
	}()

	// Wait for approval or context cancellation.
	select {
	case <-ctx.Done():
		return false
	case approved := <-ch:
		return approved
	}
}

// ResolveApproval signals the pending approval for a tool.
// Returns true if a pending approval was found and resolved.
func (a *HTTPApprover) ResolveApproval(approvalID string, approved bool) bool {
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

// HasPending returns true if there are pending approvals.
func (a *HTTPApprover) HasPending() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return len(a.pending) > 0
}
