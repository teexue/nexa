package tui

import (
	"bytes"
	"strings"
	"testing"

	"github.com/teexue/nexakit/event"
	"github.com/teexue/common-agent/core/i18n"
)

func TestRendererToolFlow(t *testing.T) {
	bundle, err := i18n.NewBundle(i18n.LocaleEn)
	if err != nil {
		t.Fatal(err)
	}
	prev := i18n.Global()
	i18n.SetGlobal(bundle)
	t.Cleanup(func() { i18n.SetGlobal(prev) })

	var buf bytes.Buffer
	r := NewRenderer(&buf, DefaultRenderOptions)

	events := make(chan event.Event, 4)
	events <- event.Event{Type: event.TypeTextDelta, Content: "Hi"}
	events <- event.Event{Type: event.TypeToolStart, Tool: "echo", Input: []byte(`{"message":"x"}`)}
	events <- event.Event{Type: event.TypeToolResult, Tool: "echo", Output: []byte(`{"message":"x"}`)}
	events <- event.Event{Type: event.TypeDone, Status: "completed", Turns: 1}
	close(events)

	r.RenderEvents(events)
	out := buf.String()
	if !strings.Contains(out, "Assistant") {
		t.Fatalf("expected Assistant header: %q", out)
	}
	if !strings.Contains(out, "● echo") {
		t.Fatalf("expected tool line: %q", out)
	}
	if !strings.Contains(out, "↳") {
		t.Fatalf("expected tool result: %q", out)
	}
	if strings.Contains(out, "[done]") {
		t.Fatalf("should be quiet done: %q", out)
	}
}

func TestRendererCoversAllEventTypes(t *testing.T) {
	bundle, err := i18n.NewBundle(i18n.LocaleEn)
	if err != nil {
		t.Fatal(err)
	}
	prev := i18n.Global()
	i18n.SetGlobal(bundle)
	t.Cleanup(func() { i18n.SetGlobal(prev) })

	for _, typ := range event.AllTypes() {
		var buf bytes.Buffer
		r := NewRenderer(&buf, DefaultRenderOptions)
		r.render(event.Event{Type: typ, Tool: "echo", Content: "x", Message: "m"})
	}
}
