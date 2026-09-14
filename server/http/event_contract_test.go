package httpapi

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/teexue/nexakit/event"
	"github.com/teexue/nexa/core/telemetry"
)

func TestEncodeRunSSECoversAllTypes(t *testing.T) {
	for _, typ := range event.AllTypes() {
		frame := encodeRunSSE(event.Event{Type: typ})
		require.True(t, strings.HasPrefix(frame, "data: "), string(typ))
		payload := strings.TrimPrefix(strings.TrimSpace(frame), "data: ")
		var got event.Event
		require.NoError(t, json.Unmarshal([]byte(payload), &got), string(typ))
		require.Equal(t, typ, got.Type)
	}
}

func TestStreamEventsEncodesAllTypes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	srv := &Server{
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		health: telemetry.NewHealthServer(),
	}
	for _, typ := range event.AllTypes() {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		ch := make(chan event.Event, 2)
		ch <- event.Event{Type: typ}
		if typ != event.TypeDone {
			ch <- event.Event{Type: event.TypeDone, Status: "completed"}
		}
		close(ch)
		srv.streamEvents(c, ch, "test", "sess-1")
		require.Contains(t, w.Body.String(), `"type":"`+string(typ)+`"`, string(typ))
	}
}
