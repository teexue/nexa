package audit

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/teexue/nexakit/provider"
)

func TestRequestLoggerLogDegradesOversized(t *testing.T) {
	l := NewRequestLogger(t.TempDir())
	msgs := make([]provider.Message, 40)
	for i := range msgs {
		msgs[i] = provider.Message{
			Role:    provider.RoleUser,
			Content: strings.Repeat("x", 20*1024),
		}
	}
	req, err := json.Marshal(provider.Request{Model: "big", Messages: msgs})
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}
	if len(req) <= requestRecordMaxBytes {
		t.Fatalf("fixture too small: %d bytes", len(req))
	}

	rec := RequestRecord{
		Timestamp:  time.Now(),
		SessionID:  "s-big",
		Agent:      "dev",
		Model:      "big",
		DurationMs: 1,
		Request:    req,
		Response:   json.RawMessage(`{"text":"` + strings.Repeat("y", 32*1024) + `"}`),
	}
	if err := l.Log(rec); err != nil {
		t.Fatalf("Log oversized: %v", err)
	}

	got, err := l.Query(RequestFilter{SessionID: "s-big"})
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("got %d records, want 1", len(got))
	}
	if got[0].SessionID != "s-big" || got[0].Agent != "dev" {
		t.Fatalf("metadata lost: %+v", got[0])
	}
	if len(got[0].Request) == 0 {
		t.Fatal("expected degraded request payload to remain")
	}
	data, _ := json.Marshal(got[0])
	if len(data) > requestRecordMaxBytes {
		t.Fatalf("persisted record still oversized: %d", len(data))
	}
}

func TestShrinkRequestPayloadKeepsLastMessages(t *testing.T) {
	msgs := make([]provider.Message, 12)
	for i := range msgs {
		msgs[i] = provider.Message{
			Role:    provider.RoleUser,
			Content: "msg-" + strings.Repeat(string(rune('a'+i)), 8),
		}
	}
	// Make last message identifiable.
	msgs[11].Content = "LAST_MESSAGE_MARKER"
	raw, err := json.Marshal(provider.Request{Model: "m", Messages: msgs})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	shrunk := shrinkRequestPayload(raw)
	var req provider.Request
	if err := json.Unmarshal(shrunk, &req); err != nil {
		t.Fatalf("unmarshal shrunk: %v", err)
	}
	if len(req.Messages) > auditKeepMessages {
		t.Fatalf("kept %d messages, want <= %d", len(req.Messages), auditKeepMessages)
	}
	last := req.Messages[len(req.Messages)-1].Content
	if !strings.Contains(last, "LAST_MESSAGE_MARKER") {
		t.Fatalf("last message lost: %q", last)
	}
	if len(req.Tools) != 0 {
		t.Fatal("tools should be cleared during shrink")
	}
}

func TestDegradeRecordDropsPayloadsAsLastResort(t *testing.T) {
	// Huge Error forces the metadata clamp path after bodies are already small.
	rec := degradeRecord(RequestRecord{
		Timestamp: time.Now(),
		SessionID: "s",
		Agent:     "dev",
		Model:     "m",
		Error:     strings.Repeat("e", requestRecordMaxBytes),
		Request:   json.RawMessage(`{"Model":"m"}`),
		Response:  json.RawMessage(`{"text":"ok"}`),
	})
	data, err := json.Marshal(rec)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if len(data) > requestRecordMaxBytes {
		t.Fatalf("degraded still oversized: %d", len(data))
	}
	if string(rec.Request) != `{"_truncated":true}` {
		t.Fatalf("request = %s, want truncated marker", rec.Request)
	}
	if rec.Response != nil {
		t.Fatalf("response = %s, want nil", rec.Response)
	}
	if len(rec.Error) > 1024+len("…") {
		t.Fatalf("error not clamped: %d bytes", len(rec.Error))
	}
	if rec.SessionID != "s" || rec.Agent != "dev" {
		t.Fatalf("core metadata lost: %+v", rec)
	}
}
