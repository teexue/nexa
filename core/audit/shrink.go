package audit

import (
	"encoding/json"

	"github.com/teexue/nexakit/provider"
)

// auditKeepMessages is how many trailing messages survive request shrink.
const auditKeepMessages = 8

// auditMessageContentBytes caps each message Content after shrink.
const auditMessageContentBytes = 4 * 1024

// auditResponseBytes caps the response payload after shrink.
const auditResponseBytes = 8 * 1024

// auditRequestTailBytes is used when request JSON cannot be parsed as a
// provider.Request: keep only the trailing bytes (recent context).
const auditRequestTailBytes = 32 * 1024

// degradeRecord shrinks bulky request/response payloads so the record can
// still be persisted under requestRecordMaxBytes. Metadata is preserved.
func degradeRecord(rec RequestRecord) RequestRecord {
	rec.Request = shrinkRequestPayload(rec.Request)
	rec.Response = truncateRawJSON(rec.Response, auditResponseBytes)
	if !recordOversize(rec) {
		return rec
	}
	// Last resort: drop bodies and clamp strings that can still blow the cap
	// (e.g. a multi-megabyte provider error).
	rec.Request = json.RawMessage(`{"_truncated":true}`)
	rec.Response = nil
	rec.Error = truncateRunes(rec.Error, 1024)
	rec.SessionID = truncateRunes(rec.SessionID, 256)
	rec.UserID = truncateRunes(rec.UserID, 256)
	rec.KeyID = truncateRunes(rec.KeyID, 256)
	rec.Agent = truncateRunes(rec.Agent, 256)
	rec.Source = truncateRunes(rec.Source, 64)
	rec.Model = truncateRunes(rec.Model, 256)
	return rec
}

func recordOversize(rec RequestRecord) bool {
	data, err := json.Marshal(rec)
	if err != nil {
		return true
	}
	return len(data) > requestRecordMaxBytes
}

// shrinkRequestPayload keeps the last N messages (and truncates long
// contents), falling back to a raw JSON tail when the payload is not a
// provider.Request.
func shrinkRequestPayload(raw json.RawMessage) json.RawMessage {
	if len(raw) == 0 {
		return raw
	}
	var req provider.Request
	if err := json.Unmarshal(raw, &req); err != nil {
		return truncateRawJSON(raw, auditRequestTailBytes)
	}
	if len(req.Messages) > auditKeepMessages {
		req.Messages = req.Messages[len(req.Messages)-auditKeepMessages:]
	}
	for i := range req.Messages {
		req.Messages[i].Content = truncateRunes(req.Messages[i].Content, auditMessageContentBytes)
		req.Messages[i].ReasoningContent = truncateRunes(req.Messages[i].ReasoningContent, auditMessageContentBytes/2)
		// Multimodal parts dominate size; drop them after keeping text Content.
		req.Messages[i].ContentParts = nil
	}
	req.Tools = nil
	out, err := json.Marshal(req)
	if err != nil {
		return truncateRawJSON(raw, auditRequestTailBytes)
	}
	if len(out) <= requestRecordMaxBytes/2 {
		return out
	}
	if len(req.Messages) > 2 {
		req.Messages = req.Messages[len(req.Messages)-2:]
		if shrunk, err := json.Marshal(req); err == nil {
			return shrunk
		}
	}
	return truncateRawJSON(out, auditRequestTailBytes)
}

// truncateRawJSON keeps the trailing n bytes of a JSON blob (UTF-8 safe) and
// wraps it so the field remains valid JSON when the cut is mid-object.
func truncateRawJSON(raw json.RawMessage, n int) json.RawMessage {
	if len(raw) <= n {
		return raw
	}
	start := len(raw) - n
	for start < len(raw) && raw[start]&0xC0 == 0x80 {
		start++
	}
	wrapped, err := json.Marshal(map[string]string{
		"_truncated": "true",
		"tail":       string(raw[start:]),
	})
	if err != nil {
		return json.RawMessage(`{"_truncated":true}`)
	}
	return wrapped
}
