package tui

import (
	"context"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/teexue/nexa/core/agent"
	"github.com/teexue/nexa/core/i18n"
	"github.com/teexue/nexakit/loop"
	"github.com/teexue/nexakit/session"
)

// focusPane tracks which region receives keys.
type focusPane int

const (
	focusComposer focusPane = iota
	focusSidebar
	focusApproval
)

// ChatModel is the fullscreen Bubble Tea chat application.
type ChatModel struct {
	cfg      ChatConfig
	theme    Theme
	hub      *hub
	approver *ChannelApprover
	runner   ChatRunner

	width, height int
	sidebarHide   bool
	focus         focusPane
	status        StreamStatus
	errText       string

	agentID     string
	agentName   string
	provider    string
	model       string
	modelLocked bool

	sess     *session.Session
	sessions []session.SessionMeta
	entries  []Entry
	models   []ModelOption
	files    []string
	workDir  string

	composer textarea.Model
	spin     spinner.Model
	frame    int

	pending   *loop.ApprovalRequest
	runCancel context.CancelFunc

	agents      []*agent.Agent
	agentCursor int

	overlay    overlayKind
	pickCursor int
	filter     string
	atPrefix   string // composer text before the active @ token

	quitting bool
}

// Init starts sidebar load, spinner, and hub listener.
func (m ChatModel) Init() tea.Cmd {
	return tea.Batch(m.spin.Tick, m.hub.listen(), loadSessionsCmd(m.runner), pulseCmd())
}

// Update handles tea messages.
func (m ChatModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.composer.SetWidth(m.mainWidth() - 4)
		return m, nil
	case tea.KeyMsg:
		return m.handleKey(msg)
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spin, cmd = m.spin.Update(msg)
		return m, cmd
	case pulseMsg:
		m.frame++
		if m.status == StatusStreaming {
			return m, pulseCmd()
		}
		return m, nil
	case streamEventMsg:
		m.entries = ApplyEvent(m.entries, msg.Ev)
		return m, m.hub.listen()
	case sessionUpdatedMsg:
		m.applySessionMeta(msg.Session)
		return m, tea.Batch(m.hub.listen(), loadSessionsCmd(m.runner))
	case streamDoneMsg:
		m.status = StatusIdle
		m.runCancel = nil
		if msg.Err != nil {
			m.status = StatusError
			m.errText = msg.Err.Error()
			m.entries = append(m.entries, Entry{Kind: EntrySystem, Content: msg.Err.Error()})
		}
		m.entries = finishStreaming(m.entries)
		return m, tea.Batch(m.hub.listen(), loadSessionsCmd(m.runner))
	case approvalNeededMsg:
		req := msg.Req
		m.pending = &req
		m.focus = focusApproval
		return m, m.hub.listen()
	case sessionsLoadedMsg:
		if msg.Err == nil {
			m.sessions = msg.Metas
		}
		return m, nil
	}
	var cmd tea.Cmd
	m.composer, cmd = m.composer.Update(msg)
	return m, cmd
}

// View renders the fullscreen layout.
func (m ChatModel) View() string {
	if m.quitting {
		return ""
	}
	if m.width == 0 {
		return i18n.T("tui.app.loading")
	}
	side := renderSidebar(m.theme, m.sessions, m.activeSessionID(), m.sidebarHide, m.height)
	mainW := m.mainWidth()
	header := renderHeader(m.theme, m.agentName, m.model, m.status, m.spin.View(), m.frame, mainW)

	approval := ""
	if m.pending != nil {
		approval = renderApproval(m.theme, m.pending, mainW)
	}
	composer := renderComposer(m.theme, m.composer, m.status == StatusStreaming, mainW)

	used := lipgloss.Height(header) + lipgloss.Height(composer) + lipgloss.Height(approval) + 1
	msgH := m.height - used
	if msgH < 3 {
		msgH = 3
	}
	msgs := renderMessages(m.theme, m.entries, mainW, msgH)
	main := lipgloss.JoinVertical(lipgloss.Left, header, msgs, approval, composer)
	if m.errText != "" && m.status == StatusError {
		main = lipgloss.JoinVertical(lipgloss.Left, main, m.theme.Err.Render(m.errText))
	}
	screen := lipgloss.JoinHorizontal(lipgloss.Top, side, main)
	if overlay := m.renderOverlay(); overlay != "" {
		screen = lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, overlay,
			lipgloss.WithWhitespaceChars(" "),
			lipgloss.WithWhitespaceForeground(dim),
		)
	}
	return screen
}

func (m ChatModel) renderOverlay() string {
	switch m.overlay {
	case overlaySlash:
		return renderSlashPalette(m.theme, filterSlash(m.filter), m.pickCursor, m.width, m.height)
	case overlayModel:
		return renderModelPalette(m.theme, filterModels(m.models, m.filter), m.pickCursor, m.width, m.height, m.modelLocked)
	case overlayFile:
		return renderFilePalette(m.theme, filterFiles(m.files, m.filter), m.pickCursor, m.width, m.height, m.workDir)
	case overlayAgent:
		return renderAgentPicker(m.theme, m.agentNames(), m.agentCursor, m.width, m.height)
	default:
		return ""
	}
}

func (m *ChatModel) applySessionMeta(sess *session.Session) {
	m.sess = sess
	if sess == nil {
		m.modelLocked = false
		return
	}
	meta := sess.GetMetadata()
	if p := meta[session.MetadataKeyProvider]; p != "" {
		m.provider = p
	}
	if model := meta[session.MetadataKeyModel]; model != "" {
		m.model = model
		m.modelLocked = true
	}
	if wd := meta[session.MetadataKeyWorkdir]; wd != "" {
		m.workDir = wd
		m.files = indexWorkdir(wd)
	}
}

func (m ChatModel) mainWidth() int {
	w := m.width
	if !m.sidebarHide {
		w -= sidebarWidth + 1
	} else {
		w -= 4
	}
	if w < 20 {
		w = 20
	}
	return w
}

func (m ChatModel) activeSessionID() string {
	if m.sess == nil {
		return ""
	}
	return m.sess.ID
}

func (m ChatModel) agentNames() []string {
	out := make([]string, 0, len(m.agents))
	for _, a := range m.agents {
		out = append(out, a.Name)
	}
	return out
}
