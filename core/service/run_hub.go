package service

import (
	"sync"
	"time"

	"github.com/teexue/nexakit/event"
	"github.com/teexue/nexakit/provider"
)

const runSubBuffer = 256

// RunAttach starts publishing a loop's event channel on the hub.
type RunAttach struct {
	SessionID string
	Agent     string
	Events    <-chan event.Event
	Cancel    func()
	Finish    func(completed bool)
	Messages  []provider.Message
	Metadata  map[string]string
}

// RunSub is a live (and optionally replayed) view of one attached run.
type RunSub struct {
	Agent    string
	Messages []provider.Message
	Metadata map[string]string
	Replay   []event.Event
	C        <-chan event.Event
	close    func()
}

// Close unsubscribes. The underlying run keeps going.
func (s *RunSub) Close() {
	if s != nil && s.close != nil {
		s.close()
	}
}

type liveRun struct {
	cancel   func()
	buf      []event.Event
	subs     map[chan event.Event]struct{}
	messages []provider.Message
	metadata map[string]string
	agent    string
	finished chan struct{}
}

// RunHub fans one loop event stream out to any number of HTTP subscribers.
type RunHub struct {
	mu      sync.Mutex
	startMu sync.Mutex
	runs    map[string]*liveRun
}

// NewRunHub creates an empty hub.
func NewRunHub() *RunHub {
	return &RunHub{runs: make(map[string]*liveRun)}
}

// Attach publishes Events until the channel closes. A second attach for the
// same session cancels the previous run first.
func (h *RunHub) Attach(a RunAttach) {
	if h == nil || a.SessionID == "" || a.Events == nil {
		return
	}
	h.startMu.Lock()
	defer h.startMu.Unlock()
	h.AbortAndWait(a.SessionID)
	r := &liveRun{
		cancel:   a.Cancel,
		subs:     make(map[chan event.Event]struct{}),
		messages: a.Messages,
		metadata: a.Metadata,
		agent:    a.Agent,
		finished: make(chan struct{}),
	}
	if r.cancel == nil {
		r.cancel = func() {}
	}
	h.mu.Lock()
	h.runs[a.SessionID] = r
	h.mu.Unlock()
	go h.pump(a.SessionID, a.Events, a.Finish)
}

// Subscribe joins a running session. replay includes events already buffered.
func (h *RunHub) Subscribe(sessionID string, replay bool) *RunSub {
	if h == nil || sessionID == "" {
		return nil
	}
	ch := make(chan event.Event, runSubBuffer)
	h.mu.Lock()
	r := h.runs[sessionID]
	if r == nil {
		h.mu.Unlock()
		return nil
	}
	var copied []event.Event
	if replay {
		copied = append([]event.Event(nil), r.buf...)
	}
	r.subs[ch] = struct{}{}
	sub := newRunSub(r, copied, ch)
	h.mu.Unlock()
	sub.close = func() { h.unsubscribe(sessionID, ch) }
	return sub
}

// Running reports whether a loop is still attached for sessionID.
func (h *RunHub) Running(sessionID string) bool {
	if h == nil || sessionID == "" {
		return false
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	_, ok := h.runs[sessionID]
	return ok
}

// Abort cancels the run. The returned channel closes when the pump exits.
func (h *RunHub) Abort(sessionID string) <-chan struct{} {
	if h == nil || sessionID == "" {
		return nil
	}
	h.mu.Lock()
	r := h.runs[sessionID]
	h.mu.Unlock()
	if r == nil {
		return nil
	}
	r.cancel()
	return r.finished
}

// AbortAndWait cancels the run and waits up to 10s for it to finish.
func (h *RunHub) AbortAndWait(sessionID string) {
	done := h.Abort(sessionID)
	if done == nil {
		return
	}
	select {
	case <-done:
	case <-time.After(10 * time.Second):
	}
}

func newRunSub(r *liveRun, replay []event.Event, ch <-chan event.Event) *RunSub {
	meta := map[string]string{}
	for k, v := range r.metadata {
		meta[k] = v
	}
	return &RunSub{
		Agent:    r.agent,
		Messages: append([]provider.Message(nil), r.messages...),
		Metadata: meta,
		Replay:   replay,
		C:        ch,
	}
}

func (h *RunHub) unsubscribe(sessionID string, ch chan event.Event) {
	h.mu.Lock()
	defer h.mu.Unlock()
	r := h.runs[sessionID]
	if r == nil {
		return
	}
	delete(r.subs, ch)
}

func (h *RunHub) pump(sessionID string, src <-chan event.Event, finish func(bool)) {
	completed := false
	for ev := range src {
		if ev.Type == event.TypeDone && ev.Status == "completed" {
			completed = true
		}
		h.dispatch(sessionID, ev)
	}
	h.finish(sessionID)
	if finish != nil {
		finish(completed)
	}
}

func (h *RunHub) dispatch(sessionID string, ev event.Event) {
	h.mu.Lock()
	r := h.runs[sessionID]
	if r == nil {
		h.mu.Unlock()
		return
	}
	r.buf = append(r.buf, ev)
	subs := make([]chan event.Event, 0, len(r.subs))
	for ch := range r.subs {
		subs = append(subs, ch)
	}
	h.mu.Unlock()
	for _, ch := range subs {
		select {
		case ch <- ev:
		default:
		}
	}
}

func (h *RunHub) finish(sessionID string) {
	h.mu.Lock()
	r := h.runs[sessionID]
	if r == nil {
		h.mu.Unlock()
		return
	}
	delete(h.runs, sessionID)
	for ch := range r.subs {
		close(ch)
	}
	close(r.finished)
	h.mu.Unlock()
}
