package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/teexue/nexa/core/config"
	"github.com/teexue/nexa/core/i18n"
)

func runTemplates(args []string) {
	if len(args) == 0 {
		printTemplatesUsage()
		return
	}

	switch args[0] {
	case "list":
		runTemplatesList()
	case "install":
		runTemplatesInstall(args[1:])
	default:
		fmt.Fprintln(os.Stderr, i18n.T("cli.error.unknown_subcommand", "name", args[0]))
		printTemplatesUsage()
		os.Exit(1)
	}
}

func printTemplatesUsage() {
	fmt.Print(i18n.T("cli.usage.templates"))
}

func runTemplatesList() {
	fmt.Println(i18n.T("cli.templates.available_header"))
	fmt.Println()
	for _, t := range config.LocalizedTemplates() {
		fmt.Printf("  %-20s %s\n", t.Name, t.Description)
	}
}

func runTemplatesInstall(args []string) {
	fs := flag.NewFlagSet("templates install", flag.ExitOnError)
	homeFlag := fs.String("home", "", i18n.T("cli.flag.home"))
	_ = fs.Parse(args)

	home := *homeFlag
	if home == "" {
		var err error
		home, err = config.Home(true)
		if err != nil {
			fmt.Fprintln(os.Stderr, i18n.T("cli.error.generic", "error", err.Error()))
			os.Exit(1)
		}
	}

	// If specific template names are provided, install those.
	remaining := fs.Args()
	if len(remaining) > 0 {
		for _, name := range remaining {
			if err := config.InstallTemplate(home, name, false); err != nil {
				fmt.Fprintln(os.Stderr, i18n.T("cli.templates.install_error", "name", name, "error", err.Error()))
				continue
			}
			fmt.Println(i18n.T("cli.templates.installed", "name", name))
		}
		return
	}

	// Otherwise install all templates.
	installed, skipped, err := config.InstallAllTemplates(home)
	if err != nil {
		fmt.Fprintln(os.Stderr, i18n.T("cli.error.generic", "error", err.Error()))
		os.Exit(1)
	}

	for _, name := range installed {
		fmt.Println(i18n.T("cli.templates.installed", "name", name))
	}
	for _, name := range skipped {
		fmt.Println(i18n.T("cli.templates.skipped_exists", "name", name))
	}
}
