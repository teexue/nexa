package tui

import (
	"context"
	"sync"

	"github.com/teexue/nexakit/loop"
)

// ChannelApprover blocks the agent loop until the TUI replies on a channel.
// OnRequest is invoked (from the loop goroutine) before waiting so the UI can
// show an approval bar; it must be safe to call from a non-UI goroutine.
type ChannelApprover struct {
	OnRequest func(req loop.ApprovalRequest)

	mu      sync.Mutex
	pending map[string]chan bool
}

// NewChannelApprover creates an empty approver.
func NewChannelApprover() *ChannelApprover {
	return &ChannelApprover{pending: make(map[string]chan bool)}
}

// Approve implements loop.Approver.
func (a *ChannelApprover) Approve(ctx context.Context, req loop.ApprovalRequest) bool {
	id := req.ApprovalID
	if id == "" {
		id = req.Tool
	}
	ch := make(chan bool, 1)
	a.mu.Lock()
	a.pending[id] = ch
	a.mu.Unlock()

	if a.OnRequest != nil {
		a.OnRequest(req)
	}

	select {
	case <-ctx.Done():
		a.clear(id)
		return false
	case ok := <-ch:
		a.clear(id)
		return ok
	}
}

// Reply resolves a pending approval. Unknown ids are ignored.
func (a *ChannelApprover) Reply(approvalID string, approved bool) {
	a.mu.Lock()
	ch := a.pending[approvalID]
	a.mu.Unlock()
	if ch == nil {
		return
	}
	select {
	case ch <- approved:
	default:
	}
}

func (a *ChannelApprover) clear(id string) {
	a.mu.Lock()
	delete(a.pending, id)
	a.mu.Unlock()
}
