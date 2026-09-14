package tui

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/teexue/nexakit/event"
	"github.com/teexue/nexakit/loop"
	"github.com/teexue/nexakit/provider"
	"github.com/teexue/nexakit/session"
)

// streamEventMsg delivers one agent event into the tea update loop.
type streamEventMsg struct{ Ev event.Event }

// streamDoneMsg marks the end of a run (success or failure).
type streamDoneMsg struct{ Err error }

// approvalNeededMsg asks the UI to present a tool approval.
type approvalNeededMsg struct{ Req loop.ApprovalRequest }

// sessionsLoadedMsg carries a refreshed session list.
type sessionsLoadedMsg struct {
	Metas []session.SessionMeta
	Err   error
}

// sessionUpdatedMsg delivers the session handle after a turn starts.
type sessionUpdatedMsg struct{ Session *session.Session }

// hub fans stream/approval messages into tea via a listen command.
type hub struct {
	ch chan tea.Msg
}

func newHub() *hub {
	return &hub{ch: make(chan tea.Msg, 64)}
}

func (h *hub) send(msg tea.Msg) {
	switch msg.(type) {
	case streamEventMsg:
		select {
		case h.ch <- msg:
		default:
		}
	default:
		h.ch <- msg
	}
}

func (h *hub) listen() tea.Cmd {
	return func() tea.Msg {
		msg, ok := <-h.ch
		if !ok {
			return streamDoneMsg{}
		}
		return msg
	}
}

type runTurnConfig struct {
	Runner    ChatRunner
	Hub       *hub
	Approver  *ChannelApprover
	Agent     string
	Prompt    string
	SessionID string
	Messages  []provider.Message
	Model     string
	Provider  string
	Ctx       context.Context
}

// runTurnCmd starts a turn in a goroutine and streams into the hub.
func runTurnCmd(cfg runTurnConfig) tea.Cmd {
	return func() tea.Msg {
		go cfg.execute()
		return nil
	}
}

func (cfg runTurnConfig) execute() {
	var runErr error
	defer func() { cfg.Hub.send(streamDoneMsg{Err: runErr}) }()

	result, err := cfg.Runner.RunTurn(cfg.Ctx, TurnRequest{
		Agent: cfg.Agent, Prompt: cfg.Prompt, SessionID: cfg.SessionID, Messages: cfg.Messages,
		Model: cfg.Model, Provider: cfg.Provider,
	}, cfg.Approver)
	if err != nil {
		runErr = err
		return
	}
	if result.Cleanup != nil {
		defer result.Cleanup()
	}
	if result.Session != nil {
		cfg.Hub.send(sessionUpdatedMsg{Session: result.Session})
	}
	if result.Events == nil {
		return
	}
	for ev := range result.Events {
		cfg.Hub.send(streamEventMsg{Ev: ev})
	}
}

func loadSessionsCmd(runner ChatRunner) tea.Cmd {
	return func() tea.Msg {
		if runner == nil {
			return sessionsLoadedMsg{}
		}
		metas, err := runner.ListSessions()
		return sessionsLoadedMsg{Metas: metas, Err: err}
	}
}
