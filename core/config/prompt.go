package config

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/teexue/nexa/core/i18n"
	"golang.org/x/term"
)

func isInteractive() bool {
	return term.IsTerminal(int(os.Stdin.Fd()))
}

func huhTheme() *huh.Theme {
	return huh.ThemeCharm()
}

func selectOption(label string, items []string, defaultIdx int) (string, error) {
	if !isInteractive() {
		return fallbackSelect(label, items, defaultIdx)
	}
	if defaultIdx < 0 || defaultIdx >= len(items) {
		defaultIdx = 0
	}

	var choice string
	opts := make([]huh.Option[string], len(items))
	for i, item := range items {
		opts[i] = huh.NewOption(item, item)
	}
	choice = items[defaultIdx]

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title(label).
				Description(i18n.T("wizard.prompt.select_hint")).
				Options(opts...).
				Value(&choice),
		),
	).WithTheme(huhTheme())

	if err := form.Run(); err != nil {
		return "", err
	}
	return choice, nil
}

func inputString(label, defaultVal string) (string, error) {
	if !isInteractive() {
		return fallbackInput(label, defaultVal), nil
	}
	var value string
	if defaultVal != "" {
		value = defaultVal
	}
	field := huh.NewInput().
		Title(label).
		Value(&value)
	if defaultVal != "" {
		field = field.Placeholder(defaultVal)
	}
	form := huh.NewForm(huh.NewGroup(field)).WithTheme(huhTheme())
	if err := form.Run(); err != nil {
		return "", err
	}
	if value == "" {
		return defaultVal, nil
	}
	return value, nil
}

// InputSecret prompts for a secret value (password echo mode).
func InputSecret(label string) (string, error) {
	if !isInteractive() {
		return fallbackInput(label, ""), nil
	}
	var value string
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title(label).
				EchoMode(huh.EchoModePassword).
				Value(&value),
		),
	).WithTheme(huhTheme())
	if err := form.Run(); err != nil {
		return "", err
	}
	return strings.TrimSpace(value), nil
}

func fallbackSelect(label string, items []string, defaultIdx int) (string, error) {
	fmt.Println(label)
	for i, item := range items {
		mark := " "
		if i == defaultIdx {
			mark = "*"
		}
		fmt.Printf("  %s [%d] %s\n", mark, i+1, item)
	}
	fmt.Print(i18n.T("wizard.prompt.fallback_select", "max", len(items), "default", defaultIdx+1))
	in := bufio.NewReader(os.Stdin)
	line, _ := in.ReadString('\n')
	line = strings.TrimSpace(line)
	if line == "" {
		return items[defaultIdx], nil
	}
	var n int
	if _, err := fmt.Sscanf(line, "%d", &n); err != nil || n < 1 || n > len(items) {
		return items[defaultIdx], nil
	}
	return items[n-1], nil
}

func fallbackInput(label, defaultVal string) string {
	if defaultVal != "" {
		fmt.Print(i18n.T("wizard.prompt.fallback_input_default", "label", label, "default", defaultVal))
	} else {
		fmt.Print(i18n.T("wizard.prompt.fallback_input", "label", label))
	}
	line, _ := bufio.NewReader(os.Stdin).ReadString('\n')
	line = strings.TrimSpace(line)
	if line == "" {
		return defaultVal
	}
	return line
}

// selectOrInput shows a list; choosing the custom option opens a text prompt.
func selectOrInput(label string, options []string, defaultIdx int, customLabel string) (string, error) {
	custom := i18n.T("wizard.option.custom")
	items := append(append([]string{}, options...), custom)
	choice, err := selectOption(label, items, defaultIdx)
	if err != nil {
		return "", err
	}
	if choice != custom {
		return choice, nil
	}
	return inputString(customLabel, "")
}
