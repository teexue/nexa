package hook_test

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/teexue/common-agent/core/hook"
	kithook "github.com/teexue/nexakit/hook"
)

func TestMetricsHook_CollectsTimings(t *testing.T) {
	m := hook.NewMetricsHook()
	ctx := context.Background()

	m.OnToolStart(ctx, kithook.ToolStartInfo{Name: "echo", Arguments: json.RawMessage(`{}`)})
	time.Sleep(10 * time.Millisecond)
	m.OnToolResult(ctx, kithook.ToolResultInfo{Name: "echo", Output: json.RawMessage(`"ok"`)})

	results := m.Results()
	if len(results) != 1 {
		t.Fatalf("got %d results, want 1", len(results))
	}
	if results[0].Name != "echo" {
		t.Fatalf("name = %q, want echo", results[0].Name)
	}
	if results[0].Duration < 5*time.Millisecond {
		t.Fatalf("duration = %v, want >= 5ms", results[0].Duration)
	}
	if results[0].Error {
		t.Fatal("expected no error")
	}
}

func TestMetricsHook_ErrorTracking(t *testing.T) {
	m := hook.NewMetricsHook()
	ctx := context.Background()

	m.OnToolStart(ctx, kithook.ToolStartInfo{Name: "fail_tool"})
	m.OnToolResult(ctx, kithook.ToolResultInfo{Name: "fail_tool", Error: fmt.Errorf("simulated error")})

	results := m.Results()
	if len(results) != 1 {
		t.Fatalf("got %d results, want 1", len(results))
	}
	if !results[0].Error {
		t.Fatal("expected error flag to be true")
	}
}

func TestMetricsHook_Reset(t *testing.T) {
	m := hook.NewMetricsHook()
	ctx := context.Background()

	m.OnToolStart(ctx, kithook.ToolStartInfo{Name: "echo"})
	m.OnToolResult(ctx, kithook.ToolResultInfo{Name: "echo"})

	if len(m.Results()) != 1 {
		t.Fatal("expected 1 result before reset")
	}

	m.Reset()
	if len(m.Results()) != 0 {
		t.Fatal("expected 0 results after reset")
	}
}
