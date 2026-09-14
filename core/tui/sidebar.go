package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/teexue/common-agent/core/i18n"
	"github.com/teexue/nexakit/session"
)

const sidebarWidth = 28

func renderSidebar(theme Theme, metas []session.SessionMeta, activeID string, collapsed bool, height int) string {
	if collapsed {
		return theme.Sidebar.Width(3).Height(height).Render(theme.Accent.Render("›"))
	}
	var b strings.Builder
	b.WriteString(theme.Accent.Render(i18n.T("tui.app.sessions")) + "\n")
	b.WriteString(theme.Dim.Render(i18n.T("tui.app.new_session_hint")) + "\n\n")
	if len(metas) == 0 {
		b.WriteString(theme.Muted.Render(i18n.T("tui.app.no_sessions")) + "\n")
	}
	max := height - 4
	if max < 1 {
		max = 1
	}
	shown := metas
	if len(shown) > max {
		shown = shown[:max]
	}
	for _, m := range shown {
		title := m.Title
		if title == "" {
			title = m.ID
		}
		if len([]rune(title)) > 20 {
			r := []rune(title)
			title = string(r[:17]) + "..."
		}
		line := fmt.Sprintf("%s\n  %s", title, m.UpdatedAt.Local().Format("01-02 15:04"))
		if m.ID == activeID {
			line = theme.Accent.Render("● " + line)
		} else {
			line = theme.Muted.Render("  " + line)
		}
		if m.Running {
			line += " " + theme.Warn.Render("…")
		}
		b.WriteString(line + "\n")
	}
	return theme.Sidebar.Width(sidebarWidth).Height(height).Render(b.String())
}

func renderHeader(theme Theme, agentName, model string, status StreamStatus, spin string, frame, width int) string {
	dot := statusGlyph(status == StatusStreaming, frame, spin)
	left := theme.Title.Render(agentName)
	if model != "" {
		left += theme.Muted.Render(" · "+model)
	}
	right := theme.StatusDot.Render(dot+" ") + theme.Muted.Render(statusLabel(status))
	gap := width - lipgloss.Width(left) - lipgloss.Width(right) - 2
	if gap < 1 {
		gap = 1
	}
	line := left + strings.Repeat(" ", gap) + right
	body := line
	if status == StatusStreaming {
		body += "\n" + pulseBar(theme, max(width-2, 8), frame)
	}
	return theme.Header.Width(width).Render(body)
}

func statusLabel(s StreamStatus) string {
	switch s {
	case StatusStreaming:
		return i18n.T("tui.app.status_streaming")
	case StatusError:
		return i18n.T("tui.app.status_error")
	default:
		return i18n.T("tui.app.status_idle")
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
