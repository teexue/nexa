package service

import (
	"fmt"

	"github.com/teexue/common-agent/core/config"
	"github.com/teexue/nexakit/builtin"
)

// ShellView is the public DTO for command-terminal settings.
type ShellView struct {
	OS         string          `json:"os"`
	Selectable bool            `json:"selectable"`
	Shell      string          `json:"shell"`
	Resolved   builtin.Shell   `json:"resolved"`
	Available  []builtin.Shell `json:"available"`
}

// GetShellSettings returns the preferred shell and hosts that are installed.
func (s *Service) GetShellSettings() (ShellView, error) {
	pref := ""
	if s.HomeDir != "" {
		settings, err := s.loadSettings()
		if err != nil {
			return ShellView{}, err
		}
		pref = settings.Shell
	}
	return buildShellView(pref)
}

// SaveShellSettings persists the preferred shell id (empty or "auto" = detect).
func (s *Service) SaveShellSettings(id string) (ShellView, error) {
	id = normalizeShellID(id)
	if id != "" {
		if _, err := builtin.ResolveShell(id); err != nil {
			return ShellView{}, &ArgError{Field: "shell", Message: err.Error()}
		}
	}
	settings, err := s.loadSettings()
	if err != nil {
		return ShellView{}, err
	}
	settings.Shell = id
	if err := config.SaveSettings(s.HomeDir, settings); err != nil {
		return ShellView{}, err
	}
	return buildShellView(id)
}

func (s *Service) preferredShell() string {
	if s.HomeDir == "" {
		return ""
	}
	settings, err := config.LoadSettings(s.HomeDir)
	if err != nil {
		return ""
	}
	return normalizeShellID(settings.Shell)
}

func buildShellView(pref string) (ShellView, error) {
	pref = normalizeShellID(pref)
	resolved, err := builtin.ResolveShell(pref)
	if err != nil {
		resolved, err = builtin.ResolveShell("")
		if err != nil {
			return ShellView{}, fmt.Errorf("detect shell: %w", err)
		}
	}
	display := pref
	if display == "" {
		display = "auto"
	}
	return ShellView{
		OS:         builtin.HostOS(),
		Selectable: builtin.ShellSelectable(),
		Shell:      display,
		Resolved:   resolved,
		Available:  builtin.DetectShells(),
	}, nil
}

func normalizeShellID(id string) string {
	if id == "auto" {
		return ""
	}
	return id
}
