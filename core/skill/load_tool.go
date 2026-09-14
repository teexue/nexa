package skill

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/teexue/nexakit/tool"
)

// loadSkillTool implements the activation stage of progressive disclosure
// (Agent Skills standard): the agent calls it with a skill name to pull the
// full SKILL.md instructions into context on demand.
type loadSkillTool struct {
	byName map[string]*Skill
}

// LoadSkillTool returns a tool.Tool that loads full SKILL.md instructions by
// name from the given skill set.
func LoadSkillTool(skills []*Skill) tool.Tool {
	byName := make(map[string]*Skill, len(skills))
	for _, s := range skills {
		if s.MDManifest != nil {
			byName[s.Name] = s
		}
	}
	return &loadSkillTool{byName: byName}
}

// Name returns the tool name.
func (t *loadSkillTool) Name() string { return "load_skill" }

// Description returns the tool description.
func (t *loadSkillTool) Description() string {
	return "Load the full instructions of an available skill by name. Call this when the task matches a skill's description from the available skills list."
}

// InputSchema returns the tool input schema.
func (t *loadSkillTool) InputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"name": map[string]any{
				"type":        "string",
				"description": "Name of the skill from the available skills list",
			},
		},
		"required": []string{"name"},
	}
}

// Execute loads the named skill's SKILL.md instructions.
func (t *loadSkillTool) Execute(_ context.Context, input json.RawMessage) (tool.Result, error) {
	var args struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(input, &args); err != nil {
		return tool.Result{}, fmt.Errorf("parse input: %w", err)
	}
	sk, ok := t.byName[args.Name]
	if !ok {
		return tool.Result{}, fmt.Errorf("skill %q not found in available skills", args.Name)
	}
	output, _ := json.Marshal(map[string]any{
		"name":         sk.Name,
		"base_dir":     sk.Dir,
		"instructions": sk.MDManifest.Body,
		"note":         "File references in the instructions are relative to base_dir.",
	})
	return tool.Result{Output: output}, nil
}
