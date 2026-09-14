package tui

import (
	"os"

	"github.com/teexue/nexakit/event"
)

// PrintEvents renders events with DefaultRenderOptions.
func PrintEvents(events <-chan event.Event) {
	NewRenderer(os.Stdout, DefaultRenderOptions).RenderEvents(events)
}

// PrintEventsVerbose renders events including done footer.
func PrintEventsVerbose(events <-chan event.Event) {
	opts := DefaultRenderOptions
	opts.QuietDone = false
	NewRenderer(os.Stdout, opts).RenderEvents(events)
}
