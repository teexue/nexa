package tui

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"golang.org/x/term"

	"github.com/teexue/common-agent/core/agent"
	"github.com/teexue/nexakit/loop"
	"github.com/teexue/nexakit/session"
)

// ChatConfig wires the fullscreen chat app.
type ChatConfig struct {
	Runner    ChatRunner
	Agent     *agent.Agent
	Session   *session.Session // optional resume
	AgentsDir string
}

// RunChat launches the fullscreen Bubble Tea chat. Requires a TTY.
func RunChat(cfg ChatConfig) error {
	if cfg.Runner == nil {
		return fmt.Errorf("runner is required")
	}
	if cfg.Agent == nil {
		return fmt.Errorf("agent is required")
	}
	if !term.IsTerminal(int(os.Stdin.Fd())) || !term.IsTerminal(int(os.Stdout.Fd())) {
		return fmt.Errorf("fullscreen chat requires an interactive terminal")
	}

	h := newHub()
	approver := NewChannelApprover()
	approver.OnRequest = func(req loop.ApprovalRequest) {
		h.send(approvalNeededMsg{Req: req})
	}

	m := newChatModel(cfg, h, approver)
	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err := p.Run()
	return err
}

func newChatModel(cfg ChatConfig, h *hub, approver *ChannelApprover) ChatModel {
	a := cfg.Agent
	m := ChatModel{
		cfg:       cfg,
		theme:     DefaultTheme(),
		hub:       h,
		approver:  approver,
		runner:    cfg.Runner,
		agentID:   a.ID,
		agentName: a.Name,
		provider:  a.Provider,
		model:     a.Model,
		sess:      cfg.Session,
		composer:  newComposer(),
		spin:      NewSpinner(),
		status:    StatusIdle,
		focus:     focusComposer,
		workDir:   resolveChatWorkDir(cfg),
	}
	m.files = indexWorkdir(m.workDir)
	if cfg.Session != nil {
		m.entries = entriesFromSession(cfg.Session)
		m.applySessionMeta(cfg.Session)
	}
	if cfg.AgentsDir != "" {
		if all, err := agent.LoadAll(cfg.AgentsDir); err == nil {
			m.agents = all.Agents
		}
	}
	if cfg.Runner != nil {
		m.models = cfg.Runner.ListModels()
	}
	return m
}

func resolveChatWorkDir(cfg ChatConfig) string {
	if cfg.Session != nil {
		if wd := cfg.Session.GetMetadata()[session.MetadataKeyWorkdir]; wd != "" {
			return wd
		}
	}
	if cwd, err := os.Getwd(); err == nil {
		return cwd
	}
	return ""
}
