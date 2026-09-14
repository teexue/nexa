package tui

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/teexue/nexakit/event"
	"github.com/teexue/nexakit/loop"
)

func TestApplyEvent_TextAndTools(t *testing.T) {
	var entries []Entry
	entries = ApplyEvent(entries, event.Event{Type: event.TypeTextDelta, Content: "Hi"})
	require.Len(t, entries, 1)
	assert.Equal(t, EntryAssistant, entries[0].Kind)
	assert.True(t, entries[0].Streaming)
	assert.Equal(t, "Hi", entries[0].Content)

	entries = ApplyEvent(entries, event.Event{
		Type: event.TypeToolStart, Tool: "echo", ToolCallID: "1",
		Input: []byte(`{"x":1}`),
	})
	require.Len(t, entries[0].Tools, 1)
	assert.Equal(t, ToolRunning, entries[0].Tools[0].Status)

	entries = ApplyEvent(entries, event.Event{
		Type: event.TypeToolResult, Tool: "echo", ToolCallID: "1",
		Output: []byte(`{"ok":true}`),
	})
	assert.Equal(t, ToolDone, entries[0].Tools[0].Status)

	entries = ApplyEvent(entries, event.Event{Type: event.TypeDone, Status: "completed"})
	assert.False(t, entries[0].Streaming)
}

func TestApplyEvent_Reasoning(t *testing.T) {
	var entries []Entry
	entries = ApplyEvent(entries, event.Event{Type: event.TypeReasoningDelta, Content: "think"})
	require.Len(t, entries, 1)
	assert.Equal(t, "think", entries[0].Reasoning)
}

func TestChannelApprover_Reply(t *testing.T) {
	a := NewChannelApprover()
	seen := make(chan loop.ApprovalRequest, 1)
	a.OnRequest = func(req loop.ApprovalRequest) { seen <- req }

	done := make(chan bool, 1)
	go func() {
		ok := a.Approve(context.Background(), loop.ApprovalRequest{
			Tool: "run_command", ApprovalID: "ap1", Arguments: []byte(`{}`),
		})
		done <- ok
	}()

	select {
	case req := <-seen:
		assert.Equal(t, "run_command", req.Tool)
	case <-time.After(time.Second):
		t.Fatal("OnRequest not called")
	}
	a.Reply("ap1", true)
	select {
	case ok := <-done:
		assert.True(t, ok)
	case <-time.After(time.Second):
		t.Fatal("Approve did not return")
	}
}

func TestChannelApprover_Cancel(t *testing.T) {
	a := NewChannelApprover()
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan bool, 1)
	go func() {
		done <- a.Approve(ctx, loop.ApprovalRequest{Tool: "x", ApprovalID: "a"})
	}()
	cancel()
	select {
	case ok := <-done:
		assert.False(t, ok)
	case <-time.After(time.Second):
		t.Fatal("Approve did not return on cancel")
	}
}
