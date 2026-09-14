package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/teexue/nexa/core/i18n"
)

// Template represents a built-in agent template.
type Template struct {
	Name        string
	Description string
	Content     string
}

// templateDef is an internal template with an i18n description key.
type templateDef struct {
	Name    string
	DescKey string
	Content string
}

var templateDefs = []templateDef{
	{
		Name:    "chat-assistant",
		DescKey: "template.chat_assistant.description",
		Content: `name: chat-assistant
version: 1
provider: moonshot
model: kimi-k2.6
system_prompt: |
  You are a helpful assistant. Answer questions clearly and concisely.
  Use tools when they can help accomplish the task.
tools:
  - get_time
max_turns: 0
max_tokens: 8000
tool_execution:
  mode: parallel
  max_parallel: 4
permissions:
  auto_approve:
    - get_time
  always_deny: []
`,
	},
	{
		Name:    "code-reviewer",
		DescKey: "template.code_reviewer.description",
		Content: `name: code-reviewer
version: 1
provider: moonshot
model: kimi-k2.6
system_prompt: |
  You are a code review assistant. You can read, search, and analyze code files.
  When reviewing code, look for:
  - Bugs and logic errors
  - Security vulnerabilities
  - Performance issues
  - Code style and readability
  - Missing error handling
  Provide specific, actionable feedback with line references.
tools:
  - read_file
  - list_directory
  - search_files
  - get_time
max_turns: 0
max_tokens: 8000
tool_execution:
  mode: parallel
  max_parallel: 4
`,
	},
	{
		Name:    "data-analyst",
		DescKey: "template.data_analyst.description",
		Content: `name: data-analyst
version: 1
provider: moonshot
model: kimi-k2.6
system_prompt: |
  You are a data analysis assistant. You can execute shell commands,
  read files, and write results. When analyzing data:
  - Start by understanding the data structure
  - Use shell commands (awk, sort, uniq, etc.) for quick analysis
  - Write analysis scripts when needed
  - Present findings with clear summaries
  Always explain your approach before executing commands.
tools:
  - read_file
  - write_file
  - list_directory
  - run_command
  - search_files
  - delete_file
  - get_time
max_turns: 0
max_tokens: 8000
tool_execution:
  mode: serial
  max_parallel: 1
permissions:
  auto_approve:
    - read_file
    - list_directory
    - search_files
    - get_time
  always_deny: []
`,
	},
	{
		Name:    "devops",
		DescKey: "template.devops.description",
		Content: `name: devops
version: 1
provider: moonshot
model: kimi-k2.6
system_prompt: |
  You are a DevOps assistant. You can execute shell commands, read and
  write files, and help with operational tasks. When working:
  - Always check current state before making changes
  - Explain what a command does before running it
  - Use safe options (e.g., dry-run) when available
  - Keep backups before destructive operations
  - Report results clearly
  Be cautious with commands that modify system state.
tools:
  - read_file
  - write_file
  - edit_file
  - list_directory
  - run_command
  - search_files
  - create_directory
  - delete_file
  - web_fetch
  - get_time
max_turns: 0
max_tokens: 8000
tool_execution:
  mode: serial
  max_parallel: 1
permissions:
  auto_approve:
    - read_file
    - list_directory
    - search_files
    - get_time
    - web_fetch
`,
	},
	{
		Name:    "software-dev",
		DescKey: "template.software_dev.description",
		Content: `name: software-dev
version: 1
provider: moonshot
model: kimi-k2.6
system_prompt: |
  You are a software development assistant. You can read, write, edit,
  and search code files, execute shell commands, and help with development tasks.

  When working on code:
  - Read existing code before making changes
  - Use search_files to find relevant code patterns
  - Use edit_file for precise, targeted modifications
  - Use write_file for creating new files
  - Use delete_file to remove files that are no longer needed
  - Run tests and build commands to verify changes
  - Explain your reasoning before making significant changes

  Best practices:
  - Follow existing code style and conventions
  - Write clear commit-message-style descriptions of changes
  - Test changes before considering them complete
  - Keep changes focused and minimal
tools:
  - read_file
  - write_file
  - edit_file
  - list_directory
  - run_command
  - search_files
  - create_directory
  - delete_file
  - web_fetch
  - get_time
max_turns: 0
max_tokens: 8000
tool_execution:
  mode: parallel
  max_parallel: 4
permissions:
  auto_approve:
    - read_file
    - list_directory
    - search_files
    - get_time
    - web_fetch
`,
	},
}

// Templates returns built-in agent templates with localized descriptions.
func Templates() []Template {
	return LocalizedTemplates()
}

// LocalizedTemplates returns templates with Description set via i18n.T.
func LocalizedTemplates() []Template {
	out := make([]Template, len(templateDefs))
	for i, d := range templateDefs {
		out[i] = Template{
			Name:        d.Name,
			Description: i18n.T(d.DescKey),
			Content:     d.Content,
		}
	}
	return out
}

// TemplateNames returns the names of all available templates.
func TemplateNames() []string {
	names := make([]string, len(templateDefs))
	for i, t := range templateDefs {
		names[i] = t.Name
	}
	return names
}

// GetTemplate returns a template by name, or nil if not found.
func GetTemplate(name string) *Template {
	for _, d := range templateDefs {
		if d.Name == name {
			t := Template{
				Name:        d.Name,
				Description: i18n.T(d.DescKey),
				Content:     d.Content,
			}
			return &t
		}
	}
	return nil
}

// InstallTemplate writes a template's agent YAML to the agents directory.
// Returns an error if the agent already exists (unless overwrite is true).
func InstallTemplate(home, templateName string, overwrite bool) error {
	tmpl := GetTemplate(templateName)
	if tmpl == nil {
		return fmt.Errorf("unknown template: %q", templateName)
	}

	agentsDir := AgentsDir(home)
	if err := os.MkdirAll(agentsDir, 0o755); err != nil {
		return fmt.Errorf("create agents dir: %w", err)
	}

	path := filepath.Join(agentsDir, templateName+".yaml")
	if !overwrite {
		if _, err := os.Stat(path); err == nil {
			return fmt.Errorf("agent %q already exists", templateName)
		}
	}

	if err := os.WriteFile(path, []byte(tmpl.Content), 0o644); err != nil {
		return fmt.Errorf("write agent yaml: %w", err)
	}

	return nil
}

// InstallAllTemplates writes all built-in templates to the agents directory.
// Skips templates that already exist.
func InstallAllTemplates(home string) (installed []string, skipped []string, err error) {
	for _, tmpl := range templateDefs {
		agentsDir := AgentsDir(home)
		path := filepath.Join(agentsDir, tmpl.Name+".yaml")
		if _, statErr := os.Stat(path); statErr == nil {
			skipped = append(skipped, tmpl.Name)
			continue
		}
		if installErr := InstallTemplate(home, tmpl.Name, false); installErr != nil {
			return installed, skipped, installErr
		}
		installed = append(installed, tmpl.Name)
	}
	return installed, skipped, nil
}
