package config

import (
	"fmt"
	"os"
	"path/filepath"
)

const dirName = ".nexa"

// Home returns ~/.nexa, creating it when ensure is true.
func Home(ensure bool) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("user home dir: %w", err)
	}
	dir := filepath.Join(home, dirName)
	if ensure {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return "", fmt.Errorf("create config dir: %w", err)
		}
		if err := os.MkdirAll(AgentsDir(dir), 0o755); err != nil {
			return "", fmt.Errorf("create agents dir: %w", err)
		}
		if err := os.MkdirAll(SessionsDir(dir), 0o755); err != nil {
			return "", fmt.Errorf("create sessions dir: %w", err)
		}
	}
	return dir, nil
}

// ProvidersFile returns the providers.yaml path under home.
func ProvidersFile(home string) string {
	return filepath.Join(home, "providers.yaml")
}

// SettingsFile returns the config.yaml path under home.
func SettingsFile(home string) string {
	return filepath.Join(home, "config.yaml")
}

// CredentialsFile returns the credentials.yaml path under home.
func CredentialsFile(home string) string {
	return filepath.Join(home, "credentials.yaml")
}

// AgentsDir returns the agents directory under home.
func AgentsDir(home string) string {
	return filepath.Join(home, "agents")
}

// SessionsDir returns the sessions directory under home.
func SessionsDir(home string) string {
	return filepath.Join(home, "sessions")
}

// KnowledgeDir returns the knowledge bases directory under home.
func KnowledgeDir(home string) string {
	return filepath.Join(home, "knowledge")
}

// SkillsDir returns the global skills directory under home.
func SkillsDir(home string) string {
	return filepath.Join(home, "skills")
}

// AgentSkillsRoot returns the root directory holding per-agent skills.
func AgentSkillsRoot(home string) string {
	return filepath.Join(home, "agent-skills")
}

// AgentSkillsDir returns the skills directory private to one agent under home.
func AgentSkillsDir(home, agent string) string {
	return filepath.Join(AgentSkillsRoot(home), agent)
}

// MCPFile returns the mcp.yaml path under home, holding global shared MCP servers.
func MCPFile(home string) string {
	return filepath.Join(home, "mcp.yaml")
}

// EnsureDirs creates the ~/.nexa directory structure (home, agents,
// sessions, knowledge, skills) without writing any default content. Use this on startup so
// the runtime has a place to read/write without pre-installing vendors or
// agents — initial content is provided by `config init` or the Settings UI.
func EnsureDirs(home string) error {
	for _, d := range []string{home, AgentsDir(home), SessionsDir(home), KnowledgeDir(home), SkillsDir(home)} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return fmt.Errorf("create dir %s: %w", d, err)
		}
	}
	return nil
}
