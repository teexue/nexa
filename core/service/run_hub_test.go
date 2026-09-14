package service

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/teexue/nexakit/event"
	"github.com/teexue/nexakit/provider"
)

func TestRunHub_SubscribeReplaysThenLive(t *testing.T) {
	h := NewRunHub()
	src := make(chan event.Event, 4)
	var finished bool
	h.Attach(RunAttach{
		SessionID: "s1",
		Agent:     "demo",
		Events:    src,
		Messages:  []provider.Message{{Role: provider.RoleUser, Content: "hi"}},
		Finish:    func(bool) { finished = true },
	})

	src <- event.Event{Type: event.TypeTextDelta, Content: "a"}
	waitBuf(t, h, "s1", 1)

	sub := h.Subscribe("s1", true)
	require.NotNil(t, sub)
	assert.Equal(t, "demo", sub.Agent)
	require.Len(t, sub.Messages, 1)
	require.Len(t, sub.Replay, 1)
	assert.Equal(t, "a", sub.Replay[0].Content)

	src <- event.Event{Type: event.TypeTextDelta, Content: "b"}
	select {
	case ev := <-sub.C:
		assert.Equal(t, "b", ev.Content)
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for live event")
	}

	close(src)
	waitNotRunning(t, h, "s1")
	assert.True(t, finished)
	sub.Close()
}

func TestRunHub_DisconnectDoesNotCancel(t *testing.T) {
	h := NewRunHub()
	src := make(chan event.Event)
	h.Attach(RunAttach{SessionID: "s1", Events: src})
	sub := h.Subscribe("s1", false)
	require.NotNil(t, sub)
	sub.Close()
	assert.True(t, h.Running("s1"))
	close(src)
	waitNotRunning(t, h, "s1")
}

func TestRunHub_AbortStopsRun(t *testing.T) {
	h := NewRunHub()
	src := make(chan event.Event)
	cancelled := make(chan struct{})
	h.Attach(RunAttach{
		SessionID: "s1",
		Events:    src,
		Cancel:    func() { close(cancelled) },
	})
	done := h.Abort("s1")
	require.NotNil(t, done)
	select {
	case <-cancelled:
	case <-time.After(time.Second):
		t.Fatal("cancel not called")
	}
	close(src)
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("pump did not finish")
	}
	assert.False(t, h.Running("s1"))
}

func waitBuf(t *testing.T, h *RunHub, id string, n int) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		h.mu.Lock()
		r := h.runs[id]
		size := 0
		if r != nil {
			size = len(r.buf)
		}
		h.mu.Unlock()
		if size >= n {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("buffer never reached %d events", n)
}

func waitNotRunning(t *testing.T, h *RunHub, id string) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if !h.Running(id) {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("run still attached")
}
