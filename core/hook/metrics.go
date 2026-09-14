package hook

import (
	"context"
	"sync"
	"time"

	kithook "github.com/teexue/nexakit/hook"
)

// ToolMetrics holds timing information for a single tool execution.
type ToolMetrics struct {
	Name     string
	Duration time.Duration
	Error    bool
}

// MetricsHook collects tool execution timing metrics.
type MetricsHook struct {
	kithook.BaseHook
	mu      sync.Mutex
	starts  map[string]time.Time
	results []ToolMetrics
}

// NewMetricsHook creates a MetricsHook.
func NewMetricsHook() *MetricsHook {
	return &MetricsHook{
		starts: make(map[string]time.Time),
	}
}

// OnToolStart records the start time of a tool execution.
func (h *MetricsHook) OnToolStart(_ context.Context, info kithook.ToolStartInfo) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.starts[info.Name] = time.Now()
	return nil
}

// OnToolResult records the duration and error status of a completed tool execution.
func (h *MetricsHook) OnToolResult(_ context.Context, info kithook.ToolResultInfo) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	start, ok := h.starts[info.Name]
	if !ok {
		return nil
	}
	delete(h.starts, info.Name)
	h.results = append(h.results, ToolMetrics{
		Name:     info.Name,
		Duration: time.Since(start),
		Error:    info.Error != nil,
	})
	return nil
}

// Results returns a copy of the collected metrics.
func (h *MetricsHook) Results() []ToolMetrics {
	h.mu.Lock()
	defer h.mu.Unlock()
	out := make([]ToolMetrics, len(h.results))
	copy(out, h.results)
	return out
}

// Reset clears the collected metrics.
func (h *MetricsHook) Reset() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.results = nil
}
