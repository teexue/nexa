package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/teexue/nexa/core/agent"
	"github.com/teexue/nexa/core/config"
	"github.com/teexue/nexa/core/i18n"
)

func runValidate(args []string) {
	fs := flag.NewFlagSet("validate", flag.ExitOnError)
	homeFlag := fs.String("home", "", i18n.T("cli.flag.home"))
	_ = fs.Parse(args)

	if fs.NArg() == 0 {
		fmt.Fprintln(os.Stderr, i18n.T("cli.usage.validate"))
		os.Exit(1)
	}

	home := *homeFlag
	if home == "" {
		var err error
		home, err = config.Home(false)
		if err != nil {
			fmt.Fprintln(os.Stderr, i18n.T("cli.error.generic", "error", err.Error()))
			os.Exit(1)
		}
	}

	// Load all tool names from registry.
	reg := newRegistry(home)
	toolNames := reg.Names()

	exitCode := 0
	for _, path := range fs.Args() {
		a, err := agent.LoadAndValidate(path, toolNames)
		if err != nil {
			fmt.Fprintln(os.Stderr, i18n.T("cli.validate.fail", "path", path, "error", err.Error()))
			exitCode = 1
			continue
		}
		fmt.Println(i18n.T("cli.validate.ok",
			"path", path,
			"name", a.Name,
			"provider", a.Provider,
			"model", a.Model,
			"tools", len(a.Tools),
		))
	}
	os.Exit(exitCode)
}
