package tui

import (
	"encoding/json"
	"strings"

	"github.com/teexue/nexakit/event"
)

// ApplyEvent reduces a stream event into the conversation entries.
// It returns the updated slice (may reuse the backing array).
func ApplyEvent(entries []Entry, ev event.Event) []Entry {
	switch ev.Type {
	case event.TypeTextDelta:
		return appendText(entries, ev.Content)
	case event.TypeReasoningDelta:
		return appendReasoning(entries, ev.Content)
	case event.TypeToolStart:
		return appendToolStart(entries, ev)
	case event.TypeToolResult:
		return appendToolResult(entries, ev)
	case event.TypeToolApproval:
		return appendToolApproval(entries, ev)
	case event.TypeCompaction:
		return append(entries, Entry{
			Kind:    EntrySystem,
			Content: ev.Content,
		})
	case event.TypeError:
		return finishStreaming(append(entries, Entry{
			Kind:    EntrySystem,
			Content: ev.Message,
		}))
	case event.TypeDone:
		return finishStreaming(entries)
	case event.TypeSubAgentStart:
		return appendSubAgentStart(entries, ev)
	case event.TypeSubAgentEnd:
		return appendToolResult(entries, event.Event{
			Type: event.TypeToolResult, Tool: "sub:" + ev.Tool, ToolCallID: ev.ToolCallID,
			Output: json.RawMessage(`"done"`),
		})
	default:
		return entries
	}
}

func appendText(entries []Entry, content string) []Entry {
	entries = ensureAssistant(entries)
	i := len(entries) - 1
	entries[i].Content += content
	return entries
}

func appendReasoning(entries []Entry, content string) []Entry {
	entries = ensureAssistant(entries)
	i := len(entries) - 1
	entries[i].Reasoning += content
	return entries
}

func appendToolStart(entries []Entry, ev event.Event) []Entry {
	entries = ensureAssistant(entries)
	i := len(entries) - 1
	id := ev.ToolCallID
	if id == "" {
		id = ev.Tool
	}
	entries[i].Tools = append(entries[i].Tools, ToolCard{
		ID: id, Name: ev.Tool, Input: rawString(ev.Input), Status: ToolRunning,
	})
	return entries
}

func appendSubAgentStart(entries []Entry, ev event.Event) []Entry {
	entries = ensureAssistant(entries)
	i := len(entries) - 1
	id := ev.ToolCallID
	name := "sub:" + ev.Tool
	status := ToolRunning
	input := ev.Content
	if ev.Status == "queued" {
		status = ToolQueued
		if ev.Message != "" {
			input = "waiting max=" + ev.Message
			if ev.Content != "" {
				input += " · " + ev.Content
			}
		}
	}
	if id != "" {
		for j := range entries[i].Tools {
			t := &entries[i].Tools[j]
			if t.ID == id {
				t.Name = name
				t.Status = status
				if input != "" {
					t.Input = input
				}
				return entries
			}
		}
	}
	if id == "" {
		id = name
	}
	entries[i].Tools = append(entries[i].Tools, ToolCard{
		ID: id, Name: name, Input: input, Status: status,
	})
	return entries
}

func appendToolResult(entries []Entry, ev event.Event) []Entry {
	if len(entries) == 0 {
		return entries
	}
	i := len(entries) - 1
	if entries[i].Kind != EntryAssistant {
		return entries
	}
	id := ev.ToolCallID
	out := rawString(ev.Output)
	for j := range entries[i].Tools {
		t := &entries[i].Tools[j]
		if (id != "" && t.ID == id) || (id == "" && t.Name == ev.Tool && (t.Status == ToolRunning || t.Status == ToolQueued)) {
			t.Output = out
			t.Status = ToolDone
			return entries
		}
	}
	entries[i].Tools = append(entries[i].Tools, ToolCard{
		ID: id, Name: ev.Tool, Output: out, Status: ToolDone,
	})
	return entries
}

func appendToolApproval(entries []Entry, ev event.Event) []Entry {
	entries = ensureAssistant(entries)
	i := len(entries) - 1
	id := ev.ApprovalID
	if id == "" {
		id = ev.ToolCallID
	}
	for j := range entries[i].Tools {
		t := &entries[i].Tools[j]
		if t.Name == ev.Tool && (t.Status == ToolRunning || t.Status == ToolPending) {
			t.Status = ToolPending
			if id != "" {
				t.ID = id
			}
			return entries
		}
	}
	entries[i].Tools = append(entries[i].Tools, ToolCard{
		ID: id, Name: ev.Tool, Input: rawString(ev.Input), Status: ToolPending,
	})
	return entries
}

func ensureAssistant(entries []Entry) []Entry {
	if n := len(entries); n > 0 && entries[n-1].Kind == EntryAssistant && entries[n-1].Streaming {
		return entries
	}
	return append(entries, Entry{Kind: EntryAssistant, Streaming: true})
}

func finishStreaming(entries []Entry) []Entry {
	for i := range entries {
		entries[i].Streaming = false
	}
	return entries
}

func rawString(v any) string {
	switch x := v.(type) {
	case nil:
		return ""
	case json.RawMessage:
		return compact(string(x))
	case []byte:
		return compact(string(x))
	case string:
		return compact(x)
	default:
		b, err := json.Marshal(x)
		if err != nil {
			return ""
		}
		return compact(string(b))
	}
}

func compact(s string) string {
	s = strings.TrimSpace(s)
	if s == "" || s == "null" || s == "{}" {
		return ""
	}
	if len(s) > 120 {
		return s[:117] + "..."
	}
	return s
}
