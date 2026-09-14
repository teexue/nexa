package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/teexue/nexa/core/config"
	"github.com/teexue/nexa/core/i18n"
	"github.com/teexue/nexa/core/skill"
)

func runSkills(args []string) {
	if len(args) == 0 {
		printSkillsUsage()
		return
	}

	switch args[0] {
	case "list":
		runSkillsList(args[1:])
	case "validate":
		runSkillsValidate(args[1:])
	case "info":
		runSkillsInfo(args[1:])
	case "install":
		runSkillsInstall(args[1:])
	case "create":
		runSkillsCreate(args[1:])
	case "remove":
		runSkillsRemove(args[1:])
	default:
		fmt.Fprintln(os.Stderr, i18n.T("cli.error.unknown_subcommand", "name", args[0]))
		printSkillsUsage()
		os.Exit(1)
	}
}

func printSkillsUsage() {
	fmt.Print(i18n.T("cli.usage.skills"))
}

func skillsDir(home string) string {
	return config.SkillsDir(home)
}

func runSkillsList(args []string) {
	fs := flag.NewFlagSet("skills list", flag.ExitOnError)
	homeFlag := fs.String("home", "", i18n.T("cli.flag.home"))
	_ = fs.Parse(args)

	home := resolveHome(*homeFlag)
	dir := skillsDir(home)

	loader := skill.NewLoader(dir)
	skills, err := loader.LoadAll()
	if err != nil {
		fmt.Fprintln(os.Stderr, i18n.T("cli.warning.generic", "error", err.Error()))
	}

	if len(skills) == 0 {
		fmt.Println(i18n.T("cli.skills.none"))
		fmt.Println(i18n.T("cli.skills.directory", "path", dir))
		return
	}

	fmt.Println(i18n.T("cli.skills.installed_header", "count", len(skills)))
	fmt.Println()
	for _, s := range skills {
		toolNames := s.ToolNames()
		fmt.Printf("  %-20s v%-8s [%s] %s\n", s.Name, s.Version, s.Format, s.Description)
		if len(toolNames) > 0 {
			fmt.Println(i18n.T("cli.skills.tools_line", "name", fmt.Sprintf("%-20s", ""), "tools", fmt.Sprint(toolNames)))
		}
		fmt.Println()
	}
}

func runSkillsValidate(args []string) {
	fs := flag.NewFlagSet("skills validate", flag.ExitOnError)
	homeFlag := fs.String("home", "", i18n.T("cli.flag.home"))
	_ = fs.Parse(args)

	home := resolveHome(*homeFlag)
	dir := skillsDir(home)

	loader := skill.NewLoader(dir)
	skills, err := loader.LoadAll()
	if err != nil {
		fmt.Fprintln(os.Stderr, i18n.T("cli.skills.load_warning", "error", err.Error()))
	}

	if len(skills) == 0 {
		fmt.Println(i18n.T("cli.skills.none_to_validate"))
		return
	}

	exitCode := 0
	for _, s := range skills {
		toolNames := s.ToolNames()
		fmt.Println(i18n.T("cli.skills.validate_ok", "name", s.Name, "version", s.Version, "format", s.Format, "tools", len(toolNames)))
	}
	os.Exit(exitCode)
}

func runSkillsInfo(args []string) {
	fs := flag.NewFlagSet("skills info", flag.ExitOnError)
	homeFlag := fs.String("home", "", i18n.T("cli.flag.home"))
	_ = fs.Parse(args)

	if fs.NArg() == 0 {
		fmt.Fprintln(os.Stderr, i18n.T("cli.usage.skills_info"))
		os.Exit(1)
	}
	name := fs.Arg(0)

	home := resolveHome(*homeFlag)
	dir := skillsDir(home)

	loader := skill.NewLoader(dir)
	s, err := loader.LoadByName(name)
	if err != nil {
		fmt.Fprintln(os.Stderr, i18n.T("cli.error.generic", "error", err.Error()))
		os.Exit(1)
	}

	fmt.Printf("%-12s %s\n", i18n.T("cli.skills.info.name"), s.Name)
	fmt.Printf("%-12s %s\n", i18n.T("cli.skills.info.version"), s.Version)
	fmt.Printf("%-12s %s\n", i18n.T("cli.skills.info.format"), s.Format)
	fmt.Printf("%-12s %s\n", i18n.T("cli.skills.info.description"), s.Description)
	fmt.Printf("%-12s %s\n", i18n.T("cli.skills.info.directory"), s.Dir)
	printMDSkillInfo(s)
	printLegacySkillInfo(s)
}

func printMDSkillInfo(s *skill.Skill) {
	if s.MDManifest == nil {
		return
	}
	fm := s.MDManifest.Frontmatter
	if fm.License != "" {
		fmt.Printf("%-12s %s\n", i18n.T("cli.skills.info.license"), fm.License)
	}
	if fm.Compatibility != "" {
		fmt.Printf("%-12s %s\n", i18n.T("cli.skills.info.compat"), fm.Compatibility)
	}
	if len(fm.Metadata) > 0 {
		fmt.Printf("%-12s %v\n", i18n.T("cli.skills.info.metadata"), fm.Metadata)
	}
	if fm.AllowedTools != "" {
		fmt.Printf("%-12s %s\n", i18n.T("cli.skills.info.allowed"), fm.AllowedTools)
	}
	if s.Body() != "" {
		fmt.Printf("\n%s\n%s\n", i18n.T("cli.skills.info.instructions"), s.Body())
	}
}

func printLegacySkillInfo(s *skill.Skill) {
	if s.LegacyManifest == nil {
		return
	}
	m := s.LegacyManifest
	if m.Author != "" {
		fmt.Printf("%-12s %s\n", i18n.T("cli.skills.info.author"), m.Author)
	}
	fmt.Println(i18n.T("cli.skills.info.tools_header", "count", len(m.Tools)))
	for _, t := range m.Tools {
		typ := t.Type
		if typ == "" {
			typ = i18n.T("cli.skills.tool_type_prompt")
		}
		fmt.Printf("  %-20s [%s] %s\n", t.Name, typ, t.Description)
	}
}

func resolveHome(homeFlag string) string {
	if homeFlag != "" {
		return homeFlag
	}
	home, err := config.Home(false)
	if err != nil {
		fmt.Fprintln(os.Stderr, i18n.T("cli.error.generic", "error", err.Error()))
		os.Exit(1)
	}
	return home
}

// skillsDestRoot resolves the install/create/remove target root from --agent.
func skillsDestRoot(home, agentName string) string {
	if agentName != "" {
		return config.AgentSkillsDir(home, agentName)
	}
	return config.SkillsDir(home)
}

func runSkillsInstall(args []string) {
	fs := flag.NewFlagSet("skills install", flag.ExitOnError)
	homeFlag := fs.String("home", "", i18n.T("cli.flag.home"))
	agentFlag := fs.String("agent", "", i18n.T("cli.skills.flag_agent"))
	overwrite := fs.Bool("overwrite", false, i18n.T("cli.skills.flag_overwrite"))
	_ = fs.Parse(args)

	if fs.NArg() == 0 {
		fmt.Fprintln(os.Stderr, i18n.T("cli.usage.skills_install"))
		os.Exit(1)
	}
	home := resolveHome(*homeFlag)

	installed, err := skill.Install(context.Background(), fs.Arg(0), skillsDestRoot(home, *agentFlag), *overwrite)
	if err != nil {
		fmt.Fprintln(os.Stderr, i18n.T("cli.error.generic", "error", err.Error()))
		os.Exit(1)
	}
	for _, name := range installed {
		fmt.Println(i18n.T("cli.skills.installed", "name", name))
	}
}

func runSkillsCreate(args []string) {
	fs := flag.NewFlagSet("skills create", flag.ExitOnError)
	homeFlag := fs.String("home", "", i18n.T("cli.flag.home"))
	agentFlag := fs.String("agent", "", i18n.T("cli.skills.flag_agent"))
	desc := fs.String("description", "", i18n.T("cli.skills.flag_description"))
	_ = fs.Parse(args)

	if fs.NArg() == 0 {
		fmt.Fprintln(os.Stderr, i18n.T("cli.usage.skills_create"))
		os.Exit(1)
	}
	name := fs.Arg(0)
	home := resolveHome(*homeFlag)
	dir := filepath.Join(skillsDestRoot(home, *agentFlag), name)

	if *desc == "" {
		*desc = fmt.Sprintf("TODO: describe what %s does and when to use it.", name)
	}
	fm := &skill.SkillFrontmatter{Name: name, Description: *desc}
	body := fmt.Sprintf("# %s\n\n## Instructions\n\n1. ...\n", name)
	if err := skill.WriteSkill(dir, fm, body); err != nil {
		fmt.Fprintln(os.Stderr, i18n.T("cli.error.generic", "error", err.Error()))
		os.Exit(1)
	}
	fmt.Println(i18n.T("cli.skills.created", "name", name, "path", dir))
}

func runSkillsRemove(args []string) {
	fs := flag.NewFlagSet("skills remove", flag.ExitOnError)
	homeFlag := fs.String("home", "", i18n.T("cli.flag.home"))
	agentFlag := fs.String("agent", "", i18n.T("cli.skills.flag_agent"))
	_ = fs.Parse(args)

	if fs.NArg() == 0 {
		fmt.Fprintln(os.Stderr, i18n.T("cli.usage.skills_remove"))
		os.Exit(1)
	}
	name := fs.Arg(0)
	home := resolveHome(*homeFlag)
	dir := filepath.Join(skillsDestRoot(home, *agentFlag), name)

	if err := skill.RemoveSkill(dir); err != nil {
		fmt.Fprintln(os.Stderr, i18n.T("cli.error.generic", "error", err.Error()))
		os.Exit(1)
	}
	fmt.Println(i18n.T("cli.skills.removed", "name", name))
}
