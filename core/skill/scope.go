package skill

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// Skill scope constants.
const (
	// ScopeGlobal marks skills shared by all agents (~/.nexa/skills).
	ScopeGlobal = "global"
	// ScopeAgent marks skills private to one agent (~/.nexa/agent-skills/<agent>).
	ScopeAgent = "agent"
)

// LoadScoped loads global skills from globalDir plus agent-private skills from
// agentDir (empty = skip), tagging each Skill with its scope. Agent-scoped
// skills override global ones with the same name.
func LoadScoped(globalDir, agentDir, agent string) ([]*Skill, error) {
	byName := make(map[string]*Skill)
	var errs []string

	global, err := NewLoader(globalDir).LoadAll()
	if err != nil {
		errs = append(errs, err.Error())
	}
	for _, s := range global {
		s.Scope = ScopeGlobal
		byName[s.Name] = s
	}

	if agentDir != "" {
		local, lerr := NewLoader(agentDir).LoadAll()
		if lerr != nil {
			errs = append(errs, lerr.Error())
		}
		for _, s := range local {
			s.Scope = ScopeAgent
			s.Agent = agent
			byName[s.Name] = s
		}
	}

	skills := make([]*Skill, 0, len(byName))
	for _, s := range byName {
		skills = append(skills, s)
	}
	sort.Slice(skills, func(i, j int) bool { return skills[i].Name < skills[j].Name })

	if len(errs) > 0 {
		return skills, fmt.Errorf("errors loading skills: %s", strings.Join(errs, "; "))
	}
	return skills, nil
}

// LoadAllScoped loads global skills and every agent's private skills under
// agentSkillsRoot, for listing and management. Agent-scoped skills keep their
// owning agent name; duplicate names across scopes are all returned.
func LoadAllScoped(globalDir, agentSkillsRoot string) ([]*Skill, error) {
	var skills []*Skill
	var errs []string

	global, err := NewLoader(globalDir).LoadAll()
	if err != nil {
		errs = append(errs, err.Error())
	}
	for _, s := range global {
		s.Scope = ScopeGlobal
		skills = append(skills, s)
	}

	entries, rerr := os.ReadDir(agentSkillsRoot)
	switch {
	case rerr == nil:
	case os.IsNotExist(rerr):
		entries = nil
	default:
		errs = append(errs, fmt.Sprintf("read agent skills root: %v", rerr))
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		agentSkills, lerr := NewLoader(filepath.Join(agentSkillsRoot, e.Name())).LoadAll()
		if lerr != nil {
			errs = append(errs, lerr.Error())
		}
		for _, s := range agentSkills {
			s.Scope = ScopeAgent
			s.Agent = e.Name()
			skills = append(skills, s)
		}
	}

	sort.Slice(skills, func(i, j int) bool {
		if skills[i].Name != skills[j].Name {
			return skills[i].Name < skills[j].Name
		}
		return skills[i].Agent < skills[j].Agent
	})

	if len(errs) > 0 {
		return skills, fmt.Errorf("errors loading skills: %s", strings.Join(errs, "; "))
	}
	return skills, nil
}

// WriteSkill validates the frontmatter and writes a SKILL.md into dir.
// The directory base name must equal the skill name (spec requirement).
func WriteSkill(dir string, fm *SkillFrontmatter, body string) error {
	if err := fm.Validate(); err != nil {
		return fmt.Errorf("validate skill: %w", err)
	}
	if base := filepath.Base(dir); base != fm.Name {
		return fmt.Errorf("skill directory name %q must match skill name %q", base, fm.Name)
	}

	fmYAML, err := yaml.Marshal(fm)
	if err != nil {
		return fmt.Errorf("marshal frontmatter: %w", err)
	}
	content := fmt.Sprintf("---\n%s---\n\n%s\n", fmYAML, strings.TrimSpace(body))

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create skill dir: %w", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(content), 0o644); err != nil {
		return fmt.Errorf("write SKILL.md: %w", err)
	}
	return nil
}

// RemoveSkill deletes the skill directory (SKILL.md or legacy skill.yaml).
func RemoveSkill(dir string) error {
	if _, err := os.Stat(filepath.Join(dir, "SKILL.md")); err != nil {
		if _, lerr := os.Stat(filepath.Join(dir, "skill.yaml")); lerr != nil {
			return fmt.Errorf("skill not found at %s", dir)
		}
	}
	if err := os.RemoveAll(dir); err != nil {
		return fmt.Errorf("remove skill dir: %w", err)
	}
	return nil
}
