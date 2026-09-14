package agent_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/teexue/nexa/core/agent"
)

func TestLoadDemoAgent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "demo.yaml")
	content := `name: demo
provider: moonshot
model: kimi-k2.6
system_prompt: test
tools: [echo, get_time]
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	a, err := agent.Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if a.Name != "demo" {
		t.Fatalf("name = %q, want demo", a.Name)
	}
	if a.Provider != "moonshot" {
		t.Fatalf("provider = %q, want moonshot", a.Provider)
	}
	if len(a.Tools) != 2 {
		t.Fatalf("tools = %v, want 2", a.Tools)
	}
}

func TestDefaults(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "minimal.yaml")
	content := `name: minimal
provider: test
model: gpt-4
system_prompt: hello
tools: [echo]
version: 1
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	a, err := agent.Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if a.MaxTurns != 0 {
		t.Fatalf("MaxTurns = %d, want 0 (unlimited)", a.MaxTurns)
	}
	if a.Compaction != nil {
		t.Fatal("expected Compaction to remain nil when unset")
	}
	if a.MaxTokens != 0 {
		t.Fatalf("MaxTokens = %d, want 0 (auto from model spec)", a.MaxTokens)
	}
	if a.ToolExecMode() != "parallel" {
		t.Fatalf("ToolExecMode = %q, want parallel", a.ToolExecMode())
	}
	if a.ToolMaxParallel() != 4 {
		t.Fatalf("ToolMaxParallel = %d, want 4", a.ToolMaxParallel())
	}
}

// TestLoadEmptyPermissions verifies that a permissions block with empty
// auto_approve/always_deny lists parses to a non-nil *Permissions, so the
// backend uses AgentPolicy (confirm-by-default) rather than AllowAll.
func TestLoadEmptyPermissions(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "perm.yaml")
	content := `name: perm
provider: test
model: gpt-4
system_prompt: hello
tools: [echo, read_file]
permissions:
  auto_approve: []
  always_deny: []
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	a, err := agent.Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if a.Permissions == nil {
		t.Fatal("expected Permissions non-nil for empty permissions block, got nil (would fall back to AllowAll)")
	}
}

func TestSerializeToolExecution(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "serial.yaml")
	content := `name: serial-test
provider: test
model: gpt-4
system_prompt: hello
tools: [echo]
max_tokens: 8000
tool_execution:
  mode: serial
  max_parallel: 1
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	a, err := agent.Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if a.ToolExecMode() != "serial" {
		t.Fatalf("ToolExecMode = %q, want serial", a.ToolExecMode())
	}
	if a.MaxTokens != 8000 {
		t.Fatalf("MaxTokens = %d, want 8000", a.MaxTokens)
	}
}

func TestLoadWithDefaults(t *testing.T) {
	// Test constructing an Agent directly (without Load) and using accessors.
	a := &agent.Agent{
		Name:     "direct",
		MaxTurns: 0,
	}
	if a.ToolExecMode() != "parallel" {
		t.Fatalf("ToolExecMode = %q, want parallel", a.ToolExecMode())
	}
	if a.ToolMaxParallel() != 4 {
		t.Fatalf("ToolMaxParallel = %d, want 4", a.ToolMaxParallel())
	}
}

func TestLoadInvalidAgent(t *testing.T) {
	tests := []struct {
		name    string
		content string
	}{
		{"missing name", "system_prompt: x\ntools: [echo]\nmodel: m\nprovider: p\n"},
		{"missing provider", "name: x\nsystem_prompt: x\nmodel: m\ntools: [echo]\n"},
		{"missing tools", "name: x\nsystem_prompt: x\nmodel: m\nprovider: p\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "bad.yaml")
			if err := os.WriteFile(path, []byte(tt.content), 0o644); err != nil {
				t.Fatal(err)
			}
			if _, err := agent.Load(path); err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func TestListAvailable(t *testing.T) {
	dir := t.TempDir()
	// Create multiple agent files.
	for _, name := range []string{"alpha", "beta", "gamma"} {
		path := filepath.Join(dir, name+".yaml")
		content := fmt.Sprintf("name: %s\nprovider: p\nmodel: m\nsystem_prompt: hi\ntools: [echo]\n", name)
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	// Create a non-yaml file that should be ignored.
	if err := os.WriteFile(filepath.Join(dir, "readme.txt"), []byte("ignore"), 0o644); err != nil {
		t.Fatal(err)
	}

	names, err := agent.ListAvailable(dir)
	if err != nil {
		t.Fatalf("ListAvailable: %v", err)
	}
	if len(names) != 3 {
		t.Fatalf("got %d names, want 3: %v", len(names), names)
	}
	// Should be sorted.
	expected := []string{"alpha", "beta", "gamma"}
	for i, want := range expected {
		if names[i] != want {
			t.Fatalf("names[%d] = %q, want %q", i, names[i], want)
		}
	}
}

func TestListAvailableEmpty(t *testing.T) {
	dir := t.TempDir()
	names, err := agent.ListAvailable(dir)
	if err != nil {
		t.Fatalf("ListAvailable: %v", err)
	}
	if len(names) != 0 {
		t.Fatalf("got %d names, want 0", len(names))
	}
}

func TestListAvailableNotExist(t *testing.T) {
	_, err := agent.ListAvailable("/nonexistent/path")
	if err == nil {
		t.Fatal("expected error for nonexistent dir")
	}
}

func TestLoadAndValidate(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.yaml")
	content := `name: test
provider: p
model: m
system_prompt: hi
tools: [echo, get_time]
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	a, err := agent.LoadAndValidate(path, []string{"echo", "get_time", "extra_tool"})
	if err != nil {
		t.Fatalf("LoadAndValidate: %v", err)
	}
	if a.Name != "test" {
		t.Fatalf("name = %q, want test", a.Name)
	}
}

func TestLoadAndValidateMissingTool(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.yaml")
	content := `name: test
provider: p
model: m
system_prompt: hi
tools: [echo, nonexistent]
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := agent.LoadAndValidate(path, []string{"echo", "get_time"})
	if err == nil {
		t.Fatal("expected error for missing tool")
	}
	if !strings.Contains(err.Error(), "nonexistent") {
		t.Errorf("error %q should mention nonexistent", err.Error())
	}
}

func TestLoadByNameAndValidate(t *testing.T) {
	dir := t.TempDir()
	content := `name: demo
provider: p
model: m
system_prompt: hi
tools: [echo]
`
	if err := os.WriteFile(filepath.Join(dir, "demo.yaml"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	a, err := agent.LoadByNameAndValidate(dir, "demo", []string{"echo", "get_time"})
	if err != nil {
		t.Fatalf("LoadByNameAndValidate: %v", err)
	}
	if a.Name != "demo" {
		t.Fatalf("name = %q, want demo", a.Name)
	}
}

func TestLoadAll(t *testing.T) {
	dir := t.TempDir()

	// Valid agents.
	writeAgent := func(name, content string) {
		path := filepath.Join(dir, name+".yaml")
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	writeAgent("beta", "name: beta\nprovider: pb\nmodel: mb\nsystem_prompt: hi\ntools: [echo]\n")
	writeAgent("alpha", "name: alpha\nprovider: pa\nmodel: ma\nsystem_prompt: hi\ntools: [echo]\n")

	// Invalid agent (missing required fields).
	writeAgent("bad", "name: bad\n")

	// Non-yaml file should be ignored.
	if err := os.WriteFile(filepath.Join(dir, "readme.txt"), []byte("ignore"), 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := agent.LoadAll(dir)
	if err != nil {
		t.Fatalf("LoadAll: %v", err)
	}

	if len(result.Agents) != 2 {
		t.Fatalf("got %d agents, want 2: %+v", len(result.Agents), result.Agents)
	}
	// Should be sorted by name.
	if result.Agents[0].Name != "alpha" || result.Agents[1].Name != "beta" {
		t.Fatalf("agents not sorted: %q, %q", result.Agents[0].Name, result.Agents[1].Name)
	}

	if len(result.Errors) != 1 {
		t.Fatalf("got %d errors, want 1: %+v", len(result.Errors), result.Errors)
	}
	if result.Errors[0].ID != "bad" {
		t.Fatalf("error id = %q, want bad", result.Errors[0].ID)
	}
}

func TestLoadAll_DirNotExist(t *testing.T) {
	_, err := agent.LoadAll("/nonexistent/path/for/agents")
	if err == nil {
		t.Fatal("expected error for nonexistent directory")
	}
}

func TestLoadAllAndValidate(t *testing.T) {
	dir := t.TempDir()
	writeAgent := func(name, content string) {
		path := filepath.Join(dir, name+".yaml")
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	writeAgent("ok", "name: ok\nprovider: p\nmodel: m\nsystem_prompt: hi\ntools: [echo]\n")
	writeAgent("missing", "name: missing\nprovider: p\nmodel: m\nsystem_prompt: hi\ntools: [echo, unknown]\n")

	result, err := agent.LoadAllAndValidate(dir, []string{"echo"})
	if err != nil {
		t.Fatalf("LoadAllAndValidate: %v", err)
	}
	if len(result.Agents) != 1 {
		t.Fatalf("got %d valid agents, want 1", len(result.Agents))
	}
	if result.Agents[0].Name != "ok" {
		t.Fatalf("agent name = %q, want ok", result.Agents[0].Name)
	}
	if len(result.Errors) != 1 {
		t.Fatalf("got %d errors, want 1", len(result.Errors))
	}
	if result.Errors[0].Name != "missing" {
		t.Fatalf("error name = %q, want missing", result.Errors[0].Name)
	}
}
