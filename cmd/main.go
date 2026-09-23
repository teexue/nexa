package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/teexue/nexakit/event"
	"github.com/teexue/nexakit/loop"
	kitcatalog "github.com/teexue/nexakit/provider/catalog"
	"github.com/teexue/nexakit/session"

	"github.com/teexue/nexa/core/agent"
	"github.com/teexue/nexa/core/config"
	"github.com/teexue/nexa/core/i18n"
	"github.com/teexue/nexa/core/service"
	"github.com/teexue/nexa/core/store"
	"github.com/teexue/nexa/core/tui"
)

func main() {
	locale := i18n.ResolveLocale("", "")
	bundle, err := i18n.NewBundle(locale)
	if err != nil {
		bundle = i18n.Global()
	} else {
		i18n.SetGlobal(bundle)
	}
	logger := slog.New(i18n.NewSlogHandler(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}), bundle))
	slog.SetDefault(logger)

	cmd, rest := parseCommand(os.Args[1:])
	switch cmd {
	case "web":
		runWeb(rest, logger)
	case "run":
		runCLI(rest, logger)
	case "chat":
		runChat(rest, logger)
	case "sessions":
		runSessions(rest, logger)
	case "tools":
		runTools(rest, logger)
	case "config":
		runConfig(rest)
	case "templates":
		runTemplates(rest)
	case "validate":
		runValidate(rest)
	case "skills":
		runSkills(rest)
	case "version":
		runVersion(rest)
	case "help":
		usage()
	default:
		usage()
		os.Exit(1)
	}
}

// parseCommand maps argv to a subcommand. No args, or a leading flag, starts Web.
func parseCommand(args []string) (cmd string, rest []string) {
	if len(args) == 0 {
		return "web", nil
	}
	switch args[0] {
	case "web", "serve":
		return "web", args[1:]
	case "help", "-h", "--help":
		return "help", nil
	case "version", "-v", "--version":
		return "version", args[1:]
	case "run", "chat", "sessions", "tools", "config", "templates", "validate", "skills":
		return args[0], args[1:]
	default:
		if strings.HasPrefix(args[0], "-") {
			return "web", args
		}
		return "", nil
	}
}

// newLocaleLogger builds a slog logger that translates log catalog keys.
func newLocaleLogger(flagLocale, settingsLocale string) *slog.Logger {
	locale := i18n.ResolveLocale(flagLocale, settingsLocale)
	bundle, err := i18n.NewBundle(locale)
	if err != nil {
		bundle = i18n.Global()
	} else {
		i18n.SetGlobal(bundle)
	}
	logger := slog.New(i18n.NewSlogHandler(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}), bundle))
	slog.SetDefault(logger)
	return logger
}

func usage() {
	fmt.Fprint(os.Stderr, i18n.T("cli.usage.main"))
}

// stringList is a repeatable flag.Value for collecting multiple --api-key values.
type stringList []string

func (s *stringList) String() string { return fmt.Sprint([]string(*s)) }
func (s *stringList) Set(v string) error {
	*s = append(*s, v)
	return nil
}

// runFlags holds parsed `run` subcommand options.
type runFlags struct {
	agent, prompt, format, home, locale string
	session                             string
	mock, cont, yes                     bool
}

func parseRunFlags(args []string) runFlags {
	fs := flag.NewFlagSet("run", flag.ExitOnError)
	agentName := fs.String("agent", "", i18n.T("cli.flag.agent"))
	prompt := fs.String("prompt", "", i18n.T("cli.flag.prompt"))
	format := fs.String("format", "text", i18n.T("cli.flag.format"))
	homeFlag := fs.String("home", "", i18n.T("cli.flag.home_short"))
	localeFlag := fs.String("locale", "", i18n.T("cli.flag.locale"))
	mock := fs.Bool("mock", false, i18n.T("cli.flag.mock"))
	sessionID := fs.String("session", "", i18n.T("cli.flag.session_run"))
	continueFlag := fs.Bool("continue", false, i18n.T("cli.flag.continue_run"))
	yes := fs.Bool("yes", false, i18n.T("cli.flag.yes"))
	_ = fs.Parse(args)
	return runFlags{
		agent: *agentName, prompt: *prompt, format: *format,
		home: *homeFlag, locale: *localeFlag, session: *sessionID,
		mock: *mock, cont: *continueFlag, yes: *yes,
	}
}

// runBootstrap holds the runtime resources opened for one CLI run.
type runBootstrap struct {
	paths    runtimePaths
	catalog  *kitcatalog.Catalog
	creds    *config.CredentialStore
	stateDB  *store.DB
	agent    *agent.Agent
	settings config.Settings
}

// bootstrapRunTarget opens state.db, loads settings, and resolves the agent
// (by display name or id) with its provider.
func bootstrapRunTarget(opts runFlags, logger *slog.Logger) runBootstrap {
	paths, err := resolvePaths(opts.home)
	if err != nil {
		logger.Error("log.cmd.resolve_paths", "error", err)
		os.Exit(1)
	}
	catalog, creds, stateDB, err := bootstrapRuntime(paths, opts.mock, logger)
	if err != nil {
		logger.Error("log.cmd.bootstrap", "error", err)
		os.Exit(1)
	}
	settings, err := config.LoadSettings(paths.home)
	if err != nil {
		logger.Error("log.config.load_settings", "error", err)
		os.Exit(1)
	}

	a, err := resolveRunAgent(paths.agentsDir, opts.agent, settings.DefaultAgent)
	if err != nil {
		logger.Error("log.agent.load", "error", err)
		os.Exit(1)
	}
	// Probe provider resolution early so config errors surface before a run.
	if _, err := resolveProvider(catalog, opts.mock)(a); err != nil {
		logger.Error("log.provider.create", "error", err)
		os.Exit(1)
	}
	return runBootstrap{paths: paths, catalog: catalog, creds: creds, stateDB: stateDB, agent: a, settings: settings}
}

// resolveContinueSession returns the session id to resume: explicit --session
// or the agent's latest persisted session for --continue.
func resolveContinueSession(opts runFlags, svc *service.Service, a *agent.Agent) string {
	if opts.session != "" {
		return opts.session
	}
	if !opts.cont {
		return ""
	}
	sid, err := latestSessionID(svc, a.ID)
	if err != nil {
		fmt.Fprintln(os.Stderr, i18n.T("cli.error.continue_session", "error", err.Error()))
		os.Exit(1)
	}
	if sid == "" {
		fmt.Fprintln(os.Stderr, i18n.T("cli.error.no_session_to_continue"))
		os.Exit(1)
	}
	return sid
}

func runCLI(args []string, logger *slog.Logger) {
	opts := parseRunFlags(args)
	if opts.format != "text" && opts.format != "json" {
		fmt.Fprintln(os.Stderr, i18n.T("cli.error.format_invalid"))
		os.Exit(1)
	}
	if opts.prompt == "" {
		fmt.Fprintln(os.Stderr, i18n.T("cli.error.prompt_required"))
		os.Exit(1)
	}

	boot := bootstrapRunTarget(opts, logger)
	if boot.stateDB != nil {
		defer boot.stateDB.Close()
	}
	logger = newLocaleLogger(opts.locale, boot.settings.Locale)

	reg := newRegistry("")
	svc := wireCLIService(cliServiceConfig{
		paths: boot.paths, reg: reg, catalog: boot.catalog,
		creds: boot.creds, stateDB: boot.stateDB, settings: boot.settings,
		mock: opts.mock, logger: logger,
	})
	sid := resolveContinueSession(opts, svc, boot.agent)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	approver := cliRunApprover(opts.yes, opts.format == "json")
	result, err := svc.PrepareRun(ctx, service.RunRequest{
		Agent: boot.agent.Name, Prompt: opts.prompt, SessionID: sid, Source: "cli",
	}, approver)
	if err != nil {
		logger.Error("log.agent.prepare", "error", err)
		os.Exit(1)
	}
	defer result.Cleanup(svc.Registry)

	events, err := loop.Run(ctx, result.Config)
	if err != nil {
		logger.Error("log.agent.run", "error", err)
		os.Exit(1)
	}

	outputCLIResult(CLIOutputConfig{Format: opts.format, Events: events, Session: result.Session, Logger: logger})
}

// cliRunApprover picks the approver for a non-interactive run:
// --yes approves everything; non-TTY or --json denies explicitly with an error.
func cliRunApprover(yes, jsonFormat bool) loop.Approver {
	if yes {
		return loop.AutoApprover{}
	}
	if jsonFormat || !isInteractiveStdin() {
		return &denyWithNotice{}
	}
	return CLIApprover{}
}

// denyWithNotice denies approval requests and explains why once per process.
type denyWithNotice struct{ noticed bool }

func (d *denyWithNotice) Approve(_ context.Context, req loop.ApprovalRequest) bool {
	if !d.noticed {
		fmt.Fprintln(os.Stderr, i18n.T("cli.error.approval_denied", "tool", req.Tool))
		d.noticed = true
	}
	return false
}

// isInteractiveStdin reports whether stdin is a terminal.
func isInteractiveStdin() bool {
	info, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}

// latestSessionID returns the most recent persisted session for an agent.
func latestSessionID(svc *service.Service, agentName string) (string, error) {
	metas, err := svc.ListSessions("")
	if err != nil {
		return "", err
	}
	for _, m := range metas {
		if m.Agent == agentName {
			return m.ID, nil
		}
	}
	return "", nil
}

// CLIOutputConfig holds configuration for CLI output formatting.
type CLIOutputConfig struct {
	Format  string
	Events  <-chan event.Event
	Session *session.Session
	Logger  *slog.Logger
}

// outputCLIResult writes agent run results in the requested format.
func outputCLIResult(cfg CLIOutputConfig) {
	if cfg.Format == "json" {
		if err := event.StreamEvents(context.Background(), os.Stdout, cfg.Events); err != nil {
			cfg.Logger.Error("log.event.stream", "error", err)
			os.Exit(1)
		}
		return
	}
	cfg.Logger.Info("log.agent.run_started", "session_id", cfg.Session.ID, "agent", cfg.Session.Agent)
	tui.PrintEvents(cfg.Events)
}

func runTools(args []string, _ *slog.Logger) {
	fs := flag.NewFlagSet("tools", flag.ExitOnError)
	agentName := fs.String("agent", "", i18n.T("cli.flag.agent_validate"))
	_ = fs.Parse(args)

	home, err := config.Home(false)
	if err != nil {
		fmt.Fprintln(os.Stderr, i18n.T("cli.error.generic", "error", err.Error()))
		os.Exit(1)
	}
	reg := newRegistry("") // uses current working directory

	if *agentName != "" {
		// Validate agent tools against registry.
		a, err := agent.LoadByNameAndValidate(config.AgentsDir(home), *agentName, reg.Names())
		if err != nil {
			fmt.Fprintln(os.Stderr, i18n.T("cli.error.generic", "error", err.Error()))
			os.Exit(1)
		}
		fmt.Println(i18n.T("cli.tools.validated_ok", "name", a.Name, "tools", fmt.Sprint(a.Tools)))
		return
	}

	// List all registered tools.
	names := reg.Names()
	if len(names) == 0 {
		fmt.Println(i18n.T("cli.tools.none"))
		return
	}
	fmt.Println(i18n.T("cli.tools.registered_header", "count", len(names)))
	for _, t := range reg.List() {
		fmt.Printf("  %-20s %s\n", t.Name(), t.Description())
	}
}
