// Package skill provides a plugin system for extending agent capabilities.
// It supports two formats:
//   - Agent Skills standard: SKILL.md with YAML frontmatter (preferred)
//   - Legacy: skill.yaml with tool definitions (backward compatible)
//
// Agent Skills standard: https://agentskills.io/specification
package skill

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/teexue/nexakit/tool"
	"gopkg.in/yaml.v3"
)

const defaultScriptTimeout = 30 * time.Second

// SkillFrontmatter represents the YAML frontmatter in SKILL.md.
type SkillFrontmatter struct {
	Name          string            `yaml:"name"`
	Description   string            `yaml:"description"`
	License       string            `yaml:"license,omitempty"`
	Compatibility string            `yaml:"compatibility,omitempty"`
	Metadata      map[string]string `yaml:"metadata,omitempty"`
	AllowedTools  string            `yaml:"allowed-tools,omitempty"`
}

// SkillManifest represents a loaded skill from SKILL.md.
type SkillManifest struct {
	Frontmatter SkillFrontmatter
	Body        string // Markdown instructions after frontmatter
	Dir         string // directory where the skill lives
}

// Validate checks the frontmatter for required fields per the spec.
func (f *SkillFrontmatter) Validate() error {
	if f.Name == "" {
		return fmt.Errorf("skill name is required")
	}
	if len(f.Name) > 64 {
		return fmt.Errorf("skill name must be 64 characters or less")
	}
	if f.Description == "" {
		return fmt.Errorf("skill description is required")
	}
	if len(f.Description) > 1024 {
		return fmt.Errorf("skill description must be 1024 characters or less")
	}
	// Validate name format: lowercase, numbers, hyphens only.
	for i, c := range f.Name {
		if (c < 'a' || c > 'z') && (c < '0' || c > '9') && c != '-' {
			return fmt.Errorf("skill name contains invalid character %q at position %d", c, i)
		}
	}
	if strings.HasPrefix(f.Name, "-") || strings.HasSuffix(f.Name, "-") {
		return fmt.Errorf("skill name must not start or end with a hyphen")
	}
	if strings.Contains(f.Name, "--") {
		return fmt.Errorf("skill name must not contain consecutive hyphens")
	}
	return nil
}

// LoadSkillMD reads a SKILL.md file and parses the frontmatter + body.
func LoadSkillMD(dir string) (*SkillManifest, error) {
	path := filepath.Join(dir, "SKILL.md")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read SKILL.md: %w", err)
	}

	fm, body, err := parseSkillMD(data)
	if err != nil {
		return nil, err
	}

	if err := fm.Validate(); err != nil {
		return nil, fmt.Errorf("validate skill: %w", err)
	}

	return &SkillManifest{
		Frontmatter: fm,
		Body:        body,
		Dir:         dir,
	}, nil
}

// parseSkillMD extracts YAML frontmatter and markdown body from SKILL.md content.
func parseSkillMD(data []byte) (SkillFrontmatter, string, error) {
	var fm SkillFrontmatter

	scanner := bufio.NewScanner(bytes.NewReader(data))

	// Find opening ---
	if !scanner.Scan() {
		return fm, "", fmt.Errorf("empty SKILL.md")
	}
	if strings.TrimSpace(scanner.Text()) != "---" {
		return fm, "", fmt.Errorf("SKILL.md must start with YAML frontmatter (---)")
	}

	// Collect frontmatter lines
	var fmLines []string
	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "---" {
			break
		}
		fmLines = append(fmLines, line)
	}

	// Parse frontmatter
	if err := yaml.Unmarshal([]byte(strings.Join(fmLines, "\n")), &fm); err != nil {
		return fm, "", fmt.Errorf("parse frontmatter: %w", err)
	}

	// Collect body (everything after closing ---)
	var bodyLines []string
	for scanner.Scan() {
		bodyLines = append(bodyLines, scanner.Text())
	}

	body := strings.TrimSpace(strings.Join(bodyLines, "\n"))
	return fm, body, nil
}

// LegacyManifest represents a skill.yaml file (backward compatible).
type LegacyManifest struct {
	Name         string    `yaml:"name"`
	Version      string    `yaml:"version"`
	Description  string    `yaml:"description"`
	Author       string    `yaml:"author,omitempty"`
	Tools        []ToolDef `yaml:"tools"`
	Prompt       string    `yaml:"prompt,omitempty"`
	Dependencies []string  `yaml:"dependencies,omitempty"`
	MinVersion   string    `yaml:"min_version,omitempty"`
}

// ToolDef describes a tool provided by a legacy skill.
type ToolDef struct {
	Name        string         `yaml:"name"`
	Description string         `yaml:"description"`
	InputSchema map[string]any `yaml:"input_schema"`
	Type        string         `yaml:"type,omitempty"` // "prompt" | "script"
	Command     string         `yaml:"command,omitempty"`
	Args        []string       `yaml:"args,omitempty"`
	Timeout     int            `yaml:"timeout,omitempty"`
}

// Validate checks the legacy manifest for required fields.
func (m *LegacyManifest) Validate() error {
	if m.Name == "" {
		return fmt.Errorf("skill name is required")
	}
	if m.Version == "" {
		return fmt.Errorf("skill version is required")
	}
	if len(m.Tools) == 0 {
		return fmt.Errorf("skill must declare at least one tool")
	}
	for i, t := range m.Tools {
		if t.Name == "" {
			return fmt.Errorf("tool[%d]: name is required", i)
		}
		if t.Description == "" {
			return fmt.Errorf("tool[%d] (%s): description is required", i, t.Name)
		}
	}
	return nil
}

// ToolNames returns the names of all tools declared by this skill.
func (m *LegacyManifest) ToolNames() []string {
	names := make([]string, len(m.Tools))
	for i, t := range m.Tools {
		names[i] = t.Name
	}
	return names
}

// LoadLegacy reads and parses a skill.yaml from the given directory.
func LoadLegacy(dir string) (*LegacySkill, error) {
	path := filepath.Join(dir, "skill.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read skill.yaml: %w", err)
	}

	var m LegacyManifest
	if err := yaml.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parse skill.yaml: %w", err)
	}

	if err := m.Validate(); err != nil {
		return nil, fmt.Errorf("validate skill: %w", err)
	}

	return &LegacySkill{Manifest: &m, Dir: dir}, nil
}

// LegacySkill is a loaded legacy skill.
type LegacySkill struct {
	Manifest *LegacyManifest
	Dir      string
}

// Skill is a unified representation that works with both formats.
type Skill struct {
	Name        string
	Description string
	Version     string
	Dir         string
	Format      string // "skill.md" or "skill.yaml"
	Scope       string // "global" or "agent" (ScopeGlobal / ScopeAgent)
	Agent       string // owning agent name when Scope is "agent"

	// For Agent Skills standard
	MDManifest *SkillManifest

	// For legacy format
	LegacyManifest *LegacyManifest
}

// Load loads a skill from a directory, trying SKILL.md first, then skill.yaml.
func Load(dir string) (*Skill, error) {
	// Try Agent Skills standard first.
	if _, err := os.Stat(filepath.Join(dir, "SKILL.md")); err == nil {
		md, err := LoadSkillMD(dir)
		if err != nil {
			return nil, err
		}
		version := "1.0"
		if v, ok := md.Frontmatter.Metadata["version"]; ok {
			version = v
		}
		return &Skill{
			Name:        md.Frontmatter.Name,
			Description: md.Frontmatter.Description,
			Version:     version,
			Dir:         dir,
			Format:      "skill.md",
			MDManifest:  md,
		}, nil
	}

	// Fall back to legacy format.
	legacy, err := LoadLegacy(dir)
	if err != nil {
		return nil, err
	}
	return &Skill{
		Name:           legacy.Manifest.Name,
		Description:    legacy.Manifest.Description,
		Version:        legacy.Manifest.Version,
		Dir:            dir,
		Format:         "skill.yaml",
		LegacyManifest: legacy.Manifest,
	}, nil
}

// Body returns the full instructions (SKILL.md body, or empty for legacy).
func (s *Skill) Body() string {
	if s.MDManifest != nil {
		return s.MDManifest.Body
	}
	return ""
}

// ToolNames returns the tool names for this skill.
func (s *Skill) ToolNames() []string {
	if s.LegacyManifest != nil {
		return s.LegacyManifest.ToolNames()
	}
	// Agent Skills standard skills don't define tools directly;
	// they use allowed-tools to pre-approve existing tools.
	return nil
}

// Loader loads skills from a directory.
type Loader struct {
	skillsDir string
}

// NewLoader creates a Loader that reads from the given directory.
func NewLoader(skillsDir string) *Loader {
	return &Loader{skillsDir: skillsDir}
}

// LoadAll loads all skills from the skills directory.
// Each subdirectory containing SKILL.md or skill.yaml is treated as a skill.
func (l *Loader) LoadAll() ([]*Skill, error) {
	entries, err := os.ReadDir(l.skillsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read skills dir: %w", err)
	}

	var skills []*Skill
	var errs []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		dir := filepath.Join(l.skillsDir, e.Name())
		s, err := Load(dir)
		if err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", e.Name(), err))
			continue
		}
		skills = append(skills, s)
	}

	sort.Slice(skills, func(i, j int) bool {
		return skills[i].Name < skills[j].Name
	})

	if len(errs) > 0 {
		return skills, fmt.Errorf("errors loading skills: %s", strings.Join(errs, "; "))
	}
	return skills, nil
}

// LoadByName loads a specific skill by name.
func (l *Loader) LoadByName(name string) (*Skill, error) {
	dir := filepath.Join(l.skillsDir, name)
	return Load(dir)
}

// SkillTool wraps a legacy skill tool definition as a tool.Tool implementation.
type SkillTool struct {
	def   ToolDef
	skill *LegacySkill
}

// NewSkillTool creates a tool.Tool from a legacy skill tool definition.
func NewSkillTool(def ToolDef, skill *LegacySkill) *SkillTool {
	return &SkillTool{def: def, skill: skill}
}

// Name returns the skill tool name.
func (t *SkillTool) Name() string { return t.def.Name }

// Description returns the skill tool description.
func (t *SkillTool) Description() string { return t.def.Description }

// InputSchema returns the skill tool input schema.
func (t *SkillTool) InputSchema() map[string]any { return t.def.InputSchema }

// Execute runs the skill tool.
func (t *SkillTool) Execute(ctx context.Context, input json.RawMessage) (tool.Result, error) {
	switch t.def.Type {
	case "script":
		return t.executeScript(ctx, input)
	default: // "prompt" or empty
		return t.executePrompt(input)
	}
}

func (t *SkillTool) executePrompt(input json.RawMessage) (tool.Result, error) {
	output, _ := json.Marshal(map[string]any{
		"skill":    t.skill.Manifest.Name,
		"tool":     t.def.Name,
		"prompt":   t.skill.Manifest.Prompt,
		"input":    input,
		"executed": true,
	})
	return tool.Result{Output: output}, nil
}

func (t *SkillTool) executeScript(ctx context.Context, input json.RawMessage) (tool.Result, error) {
	if t.def.Command == "" {
		return tool.Result{}, fmt.Errorf("script tool %q has no command", t.def.Name)
	}

	timeout := defaultScriptTimeout
	if t.def.Timeout > 0 {
		timeout = time.Duration(t.def.Timeout) * time.Second
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	inputStr := string(input)
	args := make([]string, len(t.def.Args))
	for i, a := range t.def.Args {
		args[i] = strings.ReplaceAll(a, "{{input}}", inputStr)
	}

	cmd := exec.CommandContext(ctx, t.def.Command, args...)
	cmd.Dir = t.skill.Dir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	exitCode := 0
	timedOut := false
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else if ctx.Err() == context.DeadlineExceeded {
			timedOut = true
			exitCode = -1
		} else {
			return tool.Result{}, fmt.Errorf("execute script: %w", err)
		}
	}

	output, _ := json.Marshal(map[string]any{
		"skill":     t.skill.Manifest.Name,
		"tool":      t.def.Name,
		"command":   t.def.Command,
		"args":      args,
		"stdout":    stdout.String(),
		"stderr":    stderr.String(),
		"exit_code": exitCode,
		"timed_out": timedOut,
	})
	return tool.Result{Output: output}, nil
}

// Tools returns tool.Tool implementations for all tools in a legacy skill.
func Tools(s *Skill) []tool.Tool {
	if s.LegacyManifest == nil {
		return nil
	}
	result := make([]tool.Tool, len(s.LegacyManifest.Tools))
	for i, def := range s.LegacyManifest.Tools {
		result[i] = NewSkillTool(def, &LegacySkill{Manifest: s.LegacyManifest, Dir: s.Dir})
	}
	return result
}

// AllTools returns tool.Tool implementations for all tools across multiple skills.
func AllTools(skills []*Skill) []tool.Tool {
	var result []tool.Tool
	for _, s := range skills {
		result = append(result, Tools(s)...)
	}
	return result
}
