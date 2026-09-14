package audit

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/teexue/nexakit/provider"
)

// stubProvider returns a fixed chunk stream.
type stubProvider struct {
	chunks []provider.Chunk
	err    error
}

func (s *stubProvider) Stream(_ context.Context, _ provider.Request) (<-chan provider.Chunk, error) {
	if s.err != nil {
		return nil, s.err
	}
	ch := make(chan provider.Chunk, len(s.chunks))
	for _, c := range s.chunks {
		ch <- c
	}
	close(ch)
	return ch, nil
}

func queryOne(t *testing.T, l *RequestLogger) RequestRecord {
	t.Helper()
	deadline := 0
	for {
		recs, err := l.Query(RequestFilter{})
		if err != nil {
			t.Fatalf("Query: %v", err)
		}
		if len(recs) == 1 {
			return recs[0]
		}
		deadline++
		if deadline > 100 {
			t.Fatalf("expected 1 record, got %d", len(recs))
		}
	}
}

func TestWrapProviderRecordsRequest(t *testing.T) {
	logger := NewRequestLogger(t.TempDir())
	inner := &stubProvider{chunks: []provider.Chunk{
		{TextDelta: "hello "},
		{TextDelta: "world", InputTokens: 10, OutputTokens: 2, Done: true},
	}}
	p := WrapProvider(inner, logger)

	ctx := provider.WithRunMeta(context.Background(), provider.RunMeta{Agent: "dev", SessionID: "sess1", Source: "kanban"})
	out, err := p.Stream(ctx, provider.Request{Model: "m1"})
	if err != nil {
		t.Fatalf("Stream: %v", err)
	}
	var got string
	for c := range out {
		got += c.TextDelta
	}
	if got != "hello world" {
		t.Fatalf("stream text = %q, want %q", got, "hello world")
	}

	rec := queryOne(t, logger)
	if rec.SessionID != "sess1" || rec.Agent != "dev" || rec.Source != "kanban" {
		t.Fatalf("bad meta in record: %+v", rec)
	}
	if rec.Model != "m1" {
		t.Fatalf("model = %q, want m1", rec.Model)
	}
	if rec.InputTokens != 10 || rec.OutputTokens != 2 {
		t.Fatalf("tokens = %d/%d, want 10/2", rec.InputTokens, rec.OutputTokens)
	}
	var resp struct {
		Text string `json:"text"`
	}
	if err := json.Unmarshal(rec.Response, &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp.Text != "hello world" {
		t.Fatalf("recorded text = %q", resp.Text)
	}
}

func TestWrapProviderRecordsError(t *testing.T) {
	logger := NewRequestLogger(t.TempDir())
	inner := &stubProvider{err: errors.New("connection refused")}
	p := WrapProvider(inner, logger)

	_, err := p.Stream(context.Background(), provider.Request{Model: "m1"})
	if err == nil {
		t.Fatal("expected stream error")
	}
	rec := queryOne(t, logger)
	if rec.Error != "connection refused" {
		t.Fatalf("recorded error = %q", rec.Error)
	}
}

func TestWrapProviderNilLoggerPassthrough(t *testing.T) {
	inner := &stubProvider{}
	if WrapProvider(inner, nil) != provider.Provider(inner) {
		t.Fatal("nil logger should return the inner provider unchanged")
	}
}

type stubResolver struct {
	stubProvider
	gotModel string
	gotCfg   int
	window   int
}

func (s *stubResolver) ResolveContextWindow(_ context.Context, model string, configured int) int {
	s.gotModel = model
	s.gotCfg = configured
	return s.window
}

func TestWrapProviderForwardsContextResolver(t *testing.T) {
	logger := NewRequestLogger(t.TempDir())
	inner := &stubResolver{window: 1_000_000}
	p := WrapProvider(inner, logger)
	r, ok := p.(provider.ContextResolver)
	if !ok {
		t.Fatal("audited provider must implement ContextResolver")
	}
	got := r.ResolveContextWindow(context.Background(), "glm5", 0)
	if got != 1_000_000 || inner.gotModel != "glm5" {
		t.Fatalf("forwarded window = %d model = %q", got, inner.gotModel)
	}
}
