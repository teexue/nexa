package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/teexue/nexa/core/i18n"
)

// overlayKind is the active composer trigger palette.
type overlayKind int

const (
	overlayNone overlayKind = iota
	overlaySlash
	overlayModel // opened via /model only
	overlayFile  // opened via @
	overlayAgent
)

type slashCommand struct {
	Name string
	Desc string
}

func slashCommands() []slashCommand {
	return []slashCommand{
		{Name: "help", Desc: i18n.T("tui.app.cmd_help")},
		{Name: "model", Desc: i18n.T("tui.app.cmd_model")},
		{Name: "agent", Desc: i18n.T("tui.app.cmd_agent")},
		{Name: "clear", Desc: i18n.T("tui.app.cmd_clear")},
		{Name: "new", Desc: i18n.T("tui.app.cmd_new")},
		{Name: "exit", Desc: i18n.T("tui.app.cmd_exit")},
	}
}

func filterSlash(query string) []slashCommand {
	q := strings.ToLower(strings.TrimSpace(query))
	all := slashCommands()
	if q == "" {
		return all
	}
	out := make([]slashCommand, 0, len(all))
	for _, c := range all {
		if strings.HasPrefix(c.Name, q) {
			out = append(out, c)
		}
	}
	return out
}

func filterModels(opts []ModelOption, query string) []ModelOption {
	q := strings.ToLower(strings.TrimSpace(query))
	if q == "" {
		return opts
	}
	out := make([]ModelOption, 0, len(opts))
	for _, o := range opts {
		hay := strings.ToLower(o.Model + " " + o.Provider + " " + o.Label)
		if strings.Contains(hay, q) {
			out = append(out, o)
		}
	}
	return out
}

func renderSlashPalette(theme Theme, items []slashCommand, cursor, width, height int) string {
	var b strings.Builder
	b.WriteString(theme.Accent.Render(i18n.T("tui.app.slash_title")) + "\n\n")
	if len(items) == 0 {
		b.WriteString(theme.Muted.Render(i18n.T("tui.app.palette_empty")) + "\n")
	}
	for i, c := range items {
		line := fmt.Sprintf("/%-8s %s", c.Name, c.Desc)
		if i == cursor {
			b.WriteString(theme.Accent.Render("▶ "+line) + "\n")
		} else {
			b.WriteString(theme.Muted.Render("  "+line) + "\n")
		}
	}
	b.WriteString("\n" + theme.Dim.Render(i18n.T("tui.app.palette_hint")))
	return paletteBox(b.String(), width, height)
}

func renderModelPalette(theme Theme, items []ModelOption, cursor, width, height int, locked bool) string {
	var b strings.Builder
	b.WriteString(theme.Accent.Render(i18n.T("tui.app.model_title")) + "\n\n")
	if locked {
		b.WriteString(theme.Warn.Render(i18n.T("tui.app.model_locked")) + "\n\n")
	}
	if len(items) == 0 {
		b.WriteString(theme.Muted.Render(i18n.T("tui.app.palette_empty")) + "\n")
	}
	for i, o := range items {
		label := o.Label
		if label == "" {
			label = o.Provider
		}
		line := fmt.Sprintf("%s  %s", o.Model, label)
		if i == cursor {
			b.WriteString(theme.Accent.Render("▶ "+line) + "\n")
		} else {
			b.WriteString(theme.Muted.Render("  "+line) + "\n")
		}
	}
	b.WriteString("\n" + theme.Dim.Render(i18n.T("tui.app.palette_hint")))
	return paletteBox(b.String(), width, height)
}

func renderFilePalette(theme Theme, items []string, cursor, width, height int, root string) string {
	var b strings.Builder
	b.WriteString(theme.Accent.Render(i18n.T("tui.app.file_title")) + "\n")
	if root != "" {
		b.WriteString(theme.Dim.Render(root) + "\n")
	}
	b.WriteString("\n")
	if len(items) == 0 {
		b.WriteString(theme.Muted.Render(i18n.T("tui.app.palette_empty")) + "\n")
	}
	for i, f := range items {
		if i == cursor {
			b.WriteString(theme.Accent.Render("▶ "+f) + "\n")
		} else {
			b.WriteString(theme.Muted.Render("  "+f) + "\n")
		}
	}
	b.WriteString("\n" + theme.Dim.Render(i18n.T("tui.app.palette_hint")))
	return paletteBox(b.String(), width, height)
}

func paletteBox(body string, width, height int) string {
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(accent).
		Padding(1, 2).
		Width(min(width-4, 64)).
		MaxHeight(height - 4).
		Render(body)
}

// detectTrigger returns slash overlay when the line is a /command draft.
func detectTrigger(value string) (overlayKind, string) {
	v := value
	if i := strings.IndexAny(v, "\n"); i >= 0 {
		v = v[:i]
	}
	if strings.HasPrefix(v, "/") && !strings.Contains(v, " ") {
		return overlaySlash, strings.TrimPrefix(v, "/")
	}
	return overlayNone, ""
}
