package tui

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/teexue/nexakit/event"
	"github.com/teexue/common-agent/core/i18n"
)

// RenderOptions controls terminal output behavior.
type RenderOptions struct {
	// QuietDone suppresses the done footer (status/turns).
	QuietDone bool
	// ShowReasoning prints reasoning deltas in dim style.
	ShowReasoning bool
}

// DefaultRenderOptions is tuned for interactive chat.
var DefaultRenderOptions = RenderOptions{
	QuietDone:     true,
	ShowReasoning: true,
}

// Renderer streams agent events to a terminal with clean, readable formatting.
type Renderer struct {
	out    io.Writer
	opts   RenderOptions
	opened bool // assistant block started
	needNL bool // text stream may lack a trailing newline before the next block
}

// NewRenderer creates a renderer writing to w.
func NewRenderer(w io.Writer, opts RenderOptions) *Renderer {
	if w == nil {
		w = os.Stdout
	}
	return &Renderer{out: w, opts: opts}
}

// RenderEvents consumes events until the channel closes.
func (r *Renderer) RenderEvents(events <-chan event.Event) {
	for ev := range events {
		r.render(ev)
	}
}

func (r *Renderer) render(ev event.Event) {
	switch ev.Type {
	case event.TypeTextDelta:
		r.ensureAssistantBlock()
		_, _ = io.WriteString(r.out, ev.Content)
		r.needNL = !strings.HasSuffix(ev.Content, "\n")

	case event.TypeReasoningDelta:
		if !r.opts.ShowReasoning {
			return
		}
		r.ensureAssistantBlock()
		_, _ = io.WriteString(r.out, dimStyle.Render(ev.Content))
		r.needNL = !strings.HasSuffix(ev.Content, "\n")

	case event.TypeToolStart:
		r.ensureAssistantBlock()
		r.closeLine()
		input := formatJSON(ev.Input)
		line := ev.Tool
		if input != "" {
			line = fmt.Sprintf("%s(%s)", ev.Tool, input)
		}
		_, _ = fmt.Fprintln(r.out, toolStyle.Render("● "+line))

	case event.TypeToolResult:
		r.closeLine()
		output := formatJSON(ev.Output)
		for _, line := range wrapToolResult(output) {
			_, _ = fmt.Fprintln(r.out, mutedStyle.Render("  ↳ "+line))
		}

	case event.TypeToolApproval:
		r.closeLine()
		_, _ = fmt.Fprintln(r.out, toolStyle.Render("◎ "+ev.Tool))

	case event.TypeCompaction:
		r.closeLine()
		_, _ = fmt.Fprintln(r.out, mutedStyle.Render("↻ "+ev.Content))

	case event.TypeSubAgentStart:
		r.closeLine()
		if ev.Status == "queued" {
			msg := i18n.T("tui.subagent.queued", "max", ev.Message)
			if ev.Content != "" {
				msg += ": " + ev.Content
			}
			_, _ = fmt.Fprintln(r.out, mutedStyle.Render("… "+msg))
			break
		}
		_, _ = fmt.Fprintln(r.out, toolStyle.Render("→ "+ev.Tool))

	case event.TypeSubAgentEnd:
		r.closeLine()
		_, _ = fmt.Fprintln(r.out, mutedStyle.Render("← "+ev.Tool))

	case event.TypeError:
		r.closeLine()
		_, _ = fmt.Fprintln(r.out, Error(ev.Message))

	case event.TypeDone:
		r.closeLine()
		if !r.opts.QuietDone {
			status := ev.Status
			if status == "" {
				status = i18n.T("tui.done.status_unknown")
			}
			_, _ = fmt.Fprintln(r.out, Muted(i18n.T("tui.done.footer", "status", status, "turns", ev.Turns)))
		}
		r.opened = false
	}
}

func (r *Renderer) ensureAssistantBlock() {
	if r.opened {
		return
	}
	r.opened = true
	_, _ = fmt.Fprintln(r.out)
	_, _ = fmt.Fprintln(r.out, labelStyle.Render("◆ "+i18n.T("tui.assistant.label")))
}

func (r *Renderer) closeLine() {
	if !r.needNL {
		return
	}
	_, _ = fmt.Fprintln(r.out)
	r.needNL = false
}

func formatJSON(v any) string {
	if v == nil {
		return ""
	}
	switch x := v.(type) {
	case json.RawMessage:
		return compactJSON(string(x))
	case []byte:
		return compactJSON(string(x))
	default:
		b, err := json.Marshal(x)
		if err != nil {
			return fmt.Sprint(v)
		}
		return compactJSON(string(b))
	}
}

func compactJSON(s string) string {
	s = strings.TrimSpace(s)
	if s == "" || s == "null" || s == "{}" {
		return ""
	}
	if len(s) > 120 {
		return s[:117] + "..."
	}
	return s
}

func wrapToolResult(s string) []string {
	if s == "" {
		return []string{""}
	}
	const max = 100
	if len(s) <= max {
		return []string{s}
	}
	return []string{s[:max] + "..."}
}

// PrintWelcome shows a compact session header without a fixed-width box
// (those break on CJK and wide glyphs).
func PrintWelcome(agentName, providerName, model string) {
	meta := i18n.T("tui.welcome.meta",
		"agent", agentName, "provider", providerName, "model", model)
	body := titleStyle.Render(i18n.T("tui.welcome.title")) + "\n" +
		mutedStyle.Render(meta)

	card := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(), false, false, false, true).
		BorderForeground(accent).
		PaddingLeft(1).
		Render(body)

	fmt.Println()
	fmt.Println(card)
	fmt.Println()
	fmt.Println(hintStyle.Render(i18n.T("tui.welcome.hint")))
	fmt.Println(ruleStyle.Render(strings.Repeat("─", 36)))
}

// PrintHelp shows slash commands.
func PrintHelp() {
	fmt.Println()
	fmt.Println(accentStyle.Render(i18n.T("tui.help.title")))
	rows := []struct{ cmd, desc string }{
		{"/help", i18n.T("tui.help.help")},
		{"/exit", i18n.T("tui.help.exit")},
		{"/clear", i18n.T("tui.help.clear")},
		{"/agent [name]", i18n.T("tui.help.agent")},
		{"/tools [agent]", i18n.T("tui.help.tools")},
	}
	for _, row := range rows {
		cmd := toolStyle.Width(16).Render(row.cmd)
		fmt.Printf("  %s %s\n", cmd, mutedStyle.Render(row.desc))
	}
	fmt.Println()
}
