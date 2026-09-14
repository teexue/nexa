package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/teexue/nexa/core/i18n"
)

func renderMessages(theme Theme, entries []Entry, width, height int) string {
	if width < 10 {
		width = 10
	}
	var b strings.Builder
	if len(entries) == 0 {
		b.WriteString(theme.Dim.Render(i18n.T("tui.app.empty_chat")))
	}
	for _, e := range entries {
		b.WriteString(renderEntry(theme, e, width))
		b.WriteString("\n\n")
	}
	content := strings.TrimRight(b.String(), "\n")
	// Keep the viewport bottom-aligned by taking the last height lines.
	lines := strings.Split(content, "\n")
	if len(lines) > height {
		lines = lines[len(lines)-height:]
	}
	for len(lines) < height {
		lines = append([]string{""}, lines...)
	}
	return lipgloss.NewStyle().Width(width).Height(height).Render(strings.Join(lines, "\n"))
}

func renderEntry(theme Theme, e Entry, width int) string {
	switch e.Kind {
	case EntryUser:
		return theme.MsgUser.Render("› "+e.Content)
	case EntrySystem:
		return theme.Warn.Render("⚙ "+e.Content)
	default:
		var b strings.Builder
		label := "◆ " + i18n.T("tui.assistant.label")
		if e.Streaming {
			label += " …"
		}
		b.WriteString(theme.Accent.Render(label) + "\n")
		if e.Reasoning != "" {
			r := e.Reasoning
			if len(r) > 200 {
				r = r[:197] + "..."
			}
			b.WriteString(theme.Dim.Render("  "+i18n.T("tui.app.thinking")+": "+r) + "\n")
		}
		for _, t := range e.Tools {
			b.WriteString(renderTool(theme, t) + "\n")
		}
		if e.Content != "" {
			b.WriteString(theme.MsgAssistant.Width(width).Render(e.Content))
		}
		return b.String()
	}
}

func renderTool(theme Theme, t ToolCard) string {
	icon := "●"
	style := theme.Tool
	switch t.Status {
	case ToolDone:
		icon = "✓"
		style = theme.Ok
	case ToolFailed, ToolDenied:
		icon = "✗"
		style = theme.Err
	case ToolPending:
		icon = "◎"
		style = theme.Warn
	case ToolQueued:
		icon = "…"
		style = theme.Muted
	}
	line := fmt.Sprintf("%s %s", icon, t.Name)
	if t.Input != "" {
		line += "(" + t.Input + ")"
	}
	out := style.Render("  " + line)
	if t.Output != "" {
		out += "\n" + theme.Muted.Render("    ↳ "+t.Output)
	}
	return out
}
