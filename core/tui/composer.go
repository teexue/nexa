package tui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/lipgloss"

	"github.com/teexue/common-agent/core/i18n"
	"github.com/teexue/nexakit/loop"
)

func newComposer() textarea.Model {
	ta := textarea.New()
	ta.Placeholder = i18n.T("tui.app.composer_placeholder")
	ta.Prompt = " "
	ta.ShowLineNumbers = false
	ta.SetHeight(3)
	ta.FocusedStyle.CursorLine = lipgloss.NewStyle()
	ta.BlurredStyle.CursorLine = lipgloss.NewStyle()
	ta.Focus()
	return ta
}

func renderComposer(theme Theme, ta textarea.Model, streaming bool, width int) string {
	hint := i18n.T("tui.app.composer_send")
	if streaming {
		hint = i18n.T("tui.app.composer_stop")
	}
	ta.SetWidth(max(width-4, 10))
	body := ta.View() + "\n" + theme.Dim.Render(hint)
	return theme.Composer.Width(width).Render(body)
}

func renderApproval(theme Theme, req *loop.ApprovalRequest, width int) string {
	if req == nil {
		return ""
	}
	args := string(req.Arguments)
	if len(args) > 100 {
		args = args[:97] + "..."
	}
	body := theme.Warn.Render(i18n.T("tui.approval.tool_request", "name", req.Tool)) + "\n" +
		theme.Muted.Render(args) + "\n" +
		theme.Dim.Render(i18n.T("tui.app.approval_keys"))
	return theme.Approval.Width(width).Render(body)
}

func renderAgentPicker(theme Theme, names []string, cursor, width, height int) string {
	var body string
	body += theme.Accent.Render(i18n.T("tui.app.pick_agent")) + "\n\n"
	for i, n := range names {
		line := fmt.Sprintf("  %s", n)
		if i == cursor {
			line = theme.Accent.Render("▶ " + n)
		} else {
			line = theme.Muted.Render(line)
		}
		body += line + "\n"
	}
	body += "\n" + theme.Dim.Render(i18n.T("tui.app.pick_agent_hint"))
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(accent).
		Padding(1, 2).
		Width(min(width-4, 48)).
		MaxHeight(height - 4).
		Render(body)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
