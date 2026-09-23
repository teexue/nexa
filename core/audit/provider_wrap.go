package audit

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/teexue/nexakit/provider"

	"github.com/teexue/nexa/core/auth"
)

// textTruncateLimit caps aggregated text/reasoning stored per record.
const textTruncateLimit = 16 * 1024

// auditedResponse is the aggregated response stored in a RequestRecord.
type auditedResponse struct {
	Text      string              `json:"text,omitempty"`
	Reasoning string              `json:"reasoning,omitempty"`
	ToolCalls []provider.ToolCall `json:"tool_calls,omitempty"`
}

// auditedProvider wraps a Provider and logs every request/response pair.
type auditedProvider struct {
	inner  provider.Provider
	logger *RequestLogger
}

// WrapProvider returns a Provider that logs each Stream call to logger.
// A nil logger returns the inner provider unchanged.
func WrapProvider(p provider.Provider, logger *RequestLogger) provider.Provider {
	if logger == nil || p == nil {
		return p
	}
	return &auditedProvider{inner: p, logger: logger}
}

// ResolveContextWindow forwards to the inner provider when it knows the
// model's real window (e.g. Ollama). Without this, the audit wrapper hides
// ContextResolver and the loop falls back to the 128K default.
func (a *auditedProvider) ResolveContextWindow(ctx context.Context, model string, configured int) int {
	if r, ok := a.inner.(provider.ContextResolver); ok {
		return r.ResolveContextWindow(ctx, model, configured)
	}
	return provider.EffectiveContextWindow(configured)
}

// Stream delegates to the inner provider, draining the chunk stream and
// persisting one audit record per call.
func (a *auditedProvider) Stream(ctx context.Context, req provider.Request) (<-chan provider.Chunk, error) {
	start := time.Now()
	meta := provider.RunMetaFrom(ctx)
	out, err := a.inner.Stream(ctx, req)
	if err != nil {
		a.record(ctx, recordArgs{meta: meta, req: req, callErr: err, start: start})
		return nil, err
	}

	// Tee the chunk stream: forward chunks while aggregating a summary.
	tee := make(chan provider.Chunk, 16)
	go func() {
		defer close(tee)
		var resp auditedResponse
		var inTok, outTok, cacheRead, cacheCreation int
		for c := range out {
			resp.Text += c.TextDelta
			resp.Reasoning += c.ReasoningDelta
			resp.ToolCalls = append(resp.ToolCalls, c.ToolCalls...)
			if c.InputTokens > 0 {
				inTok = c.InputTokens
			}
			if c.OutputTokens > 0 {
				outTok = c.OutputTokens
			}
			if c.CacheReadInputTokens > 0 {
				cacheRead = c.CacheReadInputTokens
			}
			if c.CacheCreationInputTokens > 0 {
				cacheCreation = c.CacheCreationInputTokens
			}
			select {
			case tee <- c:
			case <-ctx.Done():
				// Record the aborted call too — timeouts/cancellations are
				// exactly what the audit log is for.
				a.record(ctx, recordArgs{
					meta: meta, req: req, callErr: ctx.Err(), start: start,
					inTok: inTok, outTok: outTok, cacheRead: cacheRead, cacheCreation: cacheCreation,
				})
				return
			}
		}
		resp.Text = truncateRunes(resp.Text, textTruncateLimit)
		resp.Reasoning = truncateRunes(resp.Reasoning, textTruncateLimit)
		data, _ := json.Marshal(resp)
		a.record(ctx, recordArgs{
			meta: meta, req: req, resp: data, start: start,
			inTok: inTok, outTok: outTok, cacheRead: cacheRead, cacheCreation: cacheCreation,
		})
	}()
	return tee, nil
}

type recordArgs struct {
	meta          provider.RunMeta
	req           provider.Request
	resp          json.RawMessage
	callErr       error
	start         time.Time
	inTok         int
	outTok        int
	cacheRead     int
	cacheCreation int
}

func (a *auditedProvider) record(ctx context.Context, args recordArgs) {
	reqData, _ := json.Marshal(args.req)
	id := auth.IdentityFromContext(ctx)
	rec := RequestRecord{
		Timestamp:                args.start,
		SessionID:                args.meta.SessionID,
		UserID:                   id.UserID,
		KeyID:                    id.KeyID,
		Agent:                    args.meta.Agent,
		Source:                   args.meta.Source,
		Model:                    args.req.Model,
		DurationMs:               time.Since(args.start).Milliseconds(),
		Request:                  reqData,
		Response:                 args.resp,
		InputTokens:              args.inTok,
		OutputTokens:             args.outTok,
		CacheReadInputTokens:     args.cacheRead,
		CacheCreationInputTokens: args.cacheCreation,
	}
	if args.callErr != nil {
		rec.Error = args.callErr.Error()
	}
	if err := a.logger.Log(rec); err != nil {
		slog.Warn("log.audit.persist_failed",
			"error", err,
			"session_id", rec.SessionID,
			"agent", rec.Agent,
			"model", rec.Model,
		)
	}
}

// truncateRunes truncates s to at most n bytes on a UTF-8 boundary.
func truncateRunes(s string, n int) string {
	if len(s) <= n {
		return s
	}
	for n > 0 && (s[n]&0xC0) == 0x80 {
		n--
	}
	return s[:n] + "…"
}
