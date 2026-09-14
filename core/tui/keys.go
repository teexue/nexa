package tui

import (
	"context"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/teexue/nexa/core/i18n"
	"github.com/teexue/nexakit/provider"
	"github.com/teexue/nexakit/session"
)

func (m ChatModel) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.overlay != overlayNone {
		return m.handleOverlayKeys(msg)
	}
	if m.focus == focusApproval && m.pending != nil {
		return m.handleApprovalKeys(msg)
	}
	return m.handleGlobalKeys(msg)
}

func (m ChatModel) handleOverlayKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.overlay {
	case overlayAgent:
		return m.handleAgentKeys(msg)
	case overlaySlash, overlayModel, overlayFile:
		return m.handlePaletteKeys(msg)
	default:
		return m, nil
	}
}

func (m ChatModel) handleGlobalKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return m.handleInterrupt()
	case "ctrl+d":
		m.quitting = true
		return m, tea.Quit
	case "ctrl+n":
		return m.newSession()
	case "[":
		m.sidebarHide = !m.sidebarHide
		return m, nil
	case "tab":
		return m.openAgentOverlay()
	case "ctrl+s":
		return m.toggleSidebarFocus()
	case "enter":
		return m.handleEnter()
	case "esc":
		m.overlay = overlayNone
		return m, nil
	case "up", "k":
		if m.focus == focusSidebar {
			return m.moveSidebar(-1)
		}
	case "down", "j":
		if m.focus == focusSidebar {
			return m.moveSidebar(1)
		}
	}
	if m.focus == focusComposer {
		return m.typeInComposer(msg)
	}
	return m, nil
}

func (m ChatModel) typeInComposer(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.composer, cmd = m.composer.Update(msg)
	m = m.syncTriggerOverlay()
	return m, cmd
}

func (m ChatModel) syncTriggerOverlay() ChatModel {
	if m.overlay == overlayModel {
		m.filter = strings.TrimSpace(m.composer.Value())
		m.clampPickCursor(len(filterModels(m.models, m.filter)))
		return m
	}
	if kind, filter := detectTrigger(m.composer.Value()); kind == overlaySlash {
		m.overlay = overlaySlash
		m.filter = filter
		m.atPrefix = ""
		m.clampPickCursor(len(filterSlash(filter)))
		return m
	}
	if ok, query, prefix := atTrigger(m.composer.Value()); ok {
		m.refreshFiles()
		m.overlay = overlayFile
		m.filter = query
		m.atPrefix = prefix
		m.clampPickCursor(len(filterFiles(m.files, query)))
		return m
	}
	if m.overlay == overlaySlash || m.overlay == overlayFile {
		m.overlay = overlayNone
		m.filter = ""
		m.atPrefix = ""
		m.pickCursor = 0
	}
	return m
}

func (m *ChatModel) refreshModels() {
	if m.runner == nil {
		return
	}
	m.models = m.runner.ListModels()
}

func (m *ChatModel) refreshFiles() {
	if m.workDir == "" {
		return
	}
	if len(m.files) == 0 {
		m.files = indexWorkdir(m.workDir)
	}
}

func (m *ChatModel) clampPickCursor(n int) {
	if n <= 0 {
		m.pickCursor = 0
		return
	}
	if m.pickCursor >= n {
		m.pickCursor = n - 1
	}
	if m.pickCursor < 0 {
		m.pickCursor = 0
	}
}

func (m ChatModel) handlePaletteKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		was := m.overlay
		m.overlay = overlayNone
		m.filter = ""
		m.atPrefix = ""
		if was == overlaySlash {
			m.composer.Reset()
		}
		return m, nil
	case "up", "k":
		if m.pickCursor > 0 {
			m.pickCursor--
		}
		return m, nil
	case "down", "j":
		m.pickCursor++
		m.clampPickCursor(m.paletteLen())
		return m, nil
	case "enter", "tab":
		return m.confirmPalette()
	}
	return m.typeInComposer(msg)
}

func (m ChatModel) paletteLen() int {
	switch m.overlay {
	case overlaySlash:
		return len(filterSlash(m.filter))
	case overlayModel:
		return len(filterModels(m.models, m.filter))
	case overlayFile:
		return len(filterFiles(m.files, m.filter))
	default:
		return 0
	}
}

func (m ChatModel) confirmPalette() (tea.Model, tea.Cmd) {
	switch m.overlay {
	case overlaySlash:
		items := filterSlash(m.filter)
		if m.pickCursor < 0 || m.pickCursor >= len(items) {
			return m, nil
		}
		return m.runSlash(items[m.pickCursor].Name)
	case overlayModel:
		return m.confirmModelPick()
	case overlayFile:
		return m.confirmFilePick()
	default:
		return m, nil
	}
}

func (m ChatModel) runSlash(name string) (tea.Model, tea.Cmd) {
	m.composer.Reset()
	m.overlay = overlayNone
	m.filter = ""
	m.pickCursor = 0
	switch name {
	case "help":
		m.entries = append(m.entries, Entry{
			Kind: EntrySystem, Content: i18n.T("tui.app.help_body"),
		})
		return m, nil
	case "model":
		m.refreshModels()
		m.overlay = overlayModel
		m.filter = ""
		m.pickCursor = 0
		return m, nil
	case "agent":
		return m.openAgentOverlay()
	case "clear":
		m.entries = nil
		return m, nil
	case "new":
		return m.newSession()
	case "exit":
		m.quitting = true
		return m, tea.Quit
	default:
		return m, nil
	}
}

func (m ChatModel) confirmModelPick() (tea.Model, tea.Cmd) {
	if m.modelLocked {
		m.errText = i18n.T("tui.app.model_locked")
		m.overlay = overlayNone
		return m, nil
	}
	items := filterModels(m.models, m.filter)
	if m.pickCursor < 0 || m.pickCursor >= len(items) {
		return m, nil
	}
	opt := items[m.pickCursor]
	m.provider = opt.Provider
	m.model = opt.Model
	m.overlay = overlayNone
	m.filter = ""
	m.pickCursor = 0
	label := opt.Label
	if label == "" {
		label = opt.Provider
	}
	m.entries = append(m.entries, Entry{
		Kind: EntrySystem,
		Content: i18n.T("tui.app.model_switched",
			"model", opt.Model, "provider", label),
	})
	return m, nil
}

func (m ChatModel) confirmFilePick() (tea.Model, tea.Cmd) {
	items := filterFiles(m.files, m.filter)
	if m.pickCursor < 0 || m.pickCursor >= len(items) {
		return m, nil
	}
	path := items[m.pickCursor]
	m.composer.SetValue(m.atPrefix + "@" + path + " ")
	m.composer.Focus()
	m.overlay = overlayNone
	m.filter = ""
	m.atPrefix = ""
	m.pickCursor = 0
	return m, nil
}

func (m ChatModel) openAgentOverlay() (tea.Model, tea.Cmd) {
	m.overlay = overlayAgent
	m.agentCursor = 0
	m.composer.Blur()
	return m, nil
}

func (m ChatModel) handleInterrupt() (tea.Model, tea.Cmd) {
	if m.status == StatusStreaming && m.runCancel != nil {
		m.runCancel()
		return m, nil
	}
	m.quitting = true
	return m, tea.Quit
}

func (m ChatModel) toggleSidebarFocus() (tea.Model, tea.Cmd) {
	if m.focus == focusSidebar {
		m.focus = focusComposer
		m.composer.Focus()
		return m, nil
	}
	m.focus = focusSidebar
	m.composer.Blur()
	return m, nil
}

func (m ChatModel) handleEnter() (tea.Model, tea.Cmd) {
	if m.status == StatusStreaming {
		return m, nil
	}
	if m.focus == focusSidebar {
		return m.activateSidebarSession()
	}
	raw := strings.TrimSpace(m.composer.Value())
	if kind, filter := detectTrigger(raw); kind == overlaySlash {
		items := filterSlash(filter)
		if len(items) >= 1 && filter != "" {
			m.pickCursor = 0
			m.filter = filter
			return m.runSlash(items[0].Name)
		}
		if len(items) > 0 {
			m.overlay = overlaySlash
			m.filter = filter
			return m, nil
		}
	}
	if ok, query, prefix := atTrigger(m.composer.Value()); ok {
		m.refreshFiles()
		m.overlay = overlayFile
		m.filter = query
		m.atPrefix = prefix
		m.pickCursor = 0
		return m.confirmFilePick()
	}
	return m.submitPrompt()
}

func (m ChatModel) handleApprovalKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	id := m.pending.ApprovalID
	if id == "" {
		id = m.pending.Tool
	}
	switch strings.ToLower(msg.String()) {
	case "y", "enter":
		m.approver.Reply(id, true)
	case "n", "esc":
		m.approver.Reply(id, false)
	default:
		return m, nil
	}
	m.pending = nil
	m.focus = focusComposer
	m.composer.Focus()
	return m, nil
}

func (m ChatModel) handleAgentKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "tab":
		m.overlay = overlayNone
		m.focus = focusComposer
		m.composer.Focus()
		return m, nil
	case "up", "k":
		if m.agentCursor > 0 {
			m.agentCursor--
		}
		return m, nil
	case "down", "j":
		if m.agentCursor < len(m.agents)-1 {
			m.agentCursor++
		}
		return m, nil
	case "enter":
		return m.confirmAgentPick()
	}
	return m, nil
}

func (m ChatModel) confirmAgentPick() (tea.Model, tea.Cmd) {
	if m.agentCursor >= 0 && m.agentCursor < len(m.agents) {
		a := m.agents[m.agentCursor]
		m.agentID = a.ID
		m.agentName = a.Name
		m.provider = a.Provider
		m.model = a.Model
		m.modelLocked = false
		m.sess = nil
		m.entries = nil
	}
	m.overlay = overlayNone
	m.focus = focusComposer
	m.composer.Focus()
	return m, nil
}

func (m ChatModel) submitPrompt() (tea.Model, tea.Cmd) {
	prompt := strings.TrimSpace(m.composer.Value())
	if prompt == "" || m.runner == nil {
		return m, nil
	}
	display := prompt
	prompt = expandFileMentions(m.workDir, prompt)
	m.composer.Reset()
	m.overlay = overlayNone
	m.entries = append(m.entries, Entry{Kind: EntryUser, Content: display})
	m.entries = append(m.entries, Entry{Kind: EntryAssistant, Streaming: true})
	m.status = StatusStreaming
	m.errText = ""
	m.frame = 0

	ctx, cancel := context.WithCancel(context.Background())
	m.runCancel = cancel

	cfg := runTurnConfig{
		Runner: m.runner, Hub: m.hub, Approver: m.approver,
		Agent: m.agentID, Prompt: prompt, Ctx: ctx,
		Model: m.model, Provider: m.provider,
	}
	if m.sess != nil {
		cfg.SessionID = m.sess.ID
	}
	return m, tea.Batch(runTurnCmd(cfg), pulseCmd(), m.spin.Tick)
}

func (m ChatModel) newSession() (tea.Model, tea.Cmd) {
	if m.status == StatusStreaming {
		return m, nil
	}
	m.sess = nil
	m.entries = nil
	m.errText = ""
	m.modelLocked = false
	m.overlay = overlayNone
	return m, nil
}

func (m ChatModel) moveSidebar(delta int) (tea.Model, tea.Cmd) {
	if len(m.sessions) == 0 {
		return m, nil
	}
	idx := 0
	for i, s := range m.sessions {
		if s.ID == m.activeSessionID() {
			idx = i
			break
		}
	}
	idx += delta
	if idx < 0 {
		idx = 0
	}
	if idx >= len(m.sessions) {
		idx = len(m.sessions) - 1
	}
	return m.loadSession(m.sessions[idx].ID)
}

func (m ChatModel) activateSidebarSession() (tea.Model, tea.Cmd) {
	id := m.activeSessionID()
	if id == "" && len(m.sessions) > 0 {
		id = m.sessions[0].ID
	}
	if id == "" {
		return m, nil
	}
	return m.loadSession(id)
}

func (m ChatModel) loadSession(id string) (tea.Model, tea.Cmd) {
	if m.runner == nil {
		return m, nil
	}
	sess, err := m.runner.LoadSession(id)
	if err != nil {
		m.errText = err.Error()
		return m, nil
	}
	m.applySessionMeta(sess)
	m.entries = entriesFromSession(sess)
	m.agentID = sess.Agent
	m.focus = focusComposer
	m.composer.Focus()
	return m, nil
}

func entriesFromSession(sess *session.Session) []Entry {
	msgs := sess.GetMessages()
	out := make([]Entry, 0, len(msgs))
	for _, msg := range msgs {
		switch msg.Role {
		case provider.RoleUser:
			if provider.IsToolImageUserMessage(msg) {
				continue
			}
			out = append(out, Entry{Kind: EntryUser, Content: msg.Content})
		case provider.RoleAssistant:
			out = append(out, Entry{Kind: EntryAssistant, Content: msg.Content, Reasoning: msg.ReasoningContent})
		}
	}
	return out
}
