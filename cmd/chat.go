package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/chzyer/readline"
	"github.com/teexue/nexa/core/agent"
	"github.com/teexue/nexa/core/config"
	"github.com/teexue/nexa/core/i18n"
	"github.com/teexue/nexakit/loop"
	"github.com/teexue/nexa/core/service"
	"github.com/teexue/nexakit/session"
	"github.com/teexue/nexa/core/tui"
	"github.com/teexue/nexakit/registry"
	"golang.org/x/term"
)

// chatState is the legacy readline REPL fallback for non-TTY environments.
type chatState struct {
	svc      *service.Service
	paths    runtimePaths
	agent    string
	sess     *session.Session
	reg      *registry.Registry
	readline *readline.Instance
	sigCtx   context.Context
}

func runChat(args []string, logger *slog.Logger) {
	fs := flag.NewFlagSet("chat", flag.ExitOnError)
	agentName := fs.String("agent", "", i18n.T("cli.flag.agent"))
	homeFlag := fs.String("home", "", i18n.T("cli.flag.home_short"))
	mock := fs.Bool("mock", false, i18n.T("cli.flag.mock"))
	_ = fs.Parse(args)

	paths, err := resolvePaths(*homeFlag)
	if err != nil {
		logger.Error("log.cmd.resolve_paths", "error", err)
		os.Exit(1)
	}
	catalog, creds, stateDB, err := bootstrapRuntime(paths, *mock, logger)
	if err != nil {
		logger.Error("log.cmd.bootstrap", "error", err)
		os.Exit(1)
	}
	if stateDB != nil {
		defer stateDB.Close()
	}

	settings, err := config.LoadSettings(paths.home)
	if err != nil {
		logger.Error("log.config.load_settings", "error", err)
		os.Exit(1)
	}
	reg := newRegistry("")
	svc := wireCLIService(cliServiceConfig{
		paths: paths, reg: reg, catalog: catalog,
		creds: creds, stateDB: stateDB, settings: settings,
		mock: *mock, logger: logger,
	})

	a, err := resolveRunAgent(paths.agentsDir, *agentName, settings.DefaultAgent)
	if err != nil {
		logger.Error("log.agent.load", "error", err)
		os.Exit(1)
	}

	if term.IsTerminal(int(os.Stdin.Fd())) && term.IsTerminal(int(os.Stdout.Fd())) {
		err := tui.RunChat(tui.ChatConfig{
			Runner:    serviceChatRunner{svc: svc},
			Agent:     a,
			AgentsDir: paths.agentsDir,
		})
		if err != nil {
			logger.Error("log.chat.tui", "error", err)
			os.Exit(1)
		}
		return
	}

	runChatREPL(svc, paths, a, reg, logger)
}

func runChatREPL(svc *service.Service, paths runtimePaths, a *agent.Agent, reg *registry.Registry, logger *slog.Logger) {
	rl, err := newChatReadline(paths.home)
	if err != nil {
		logger.Error("log.chat.readline", "error", err)
		os.Exit(1)
	}
	defer rl.Close()

	state := &chatState{svc: svc, paths: paths, agent: a.ID, reg: reg, readline: rl}
	defer withSignalContext(state)()
	tui.PrintWelcome(a.Name, a.Provider, a.Model)
	runChatLoop(state)
}

func launchChatTUI(svc *service.Service, paths runtimePaths, a *agent.Agent, sess *session.Session) error {
	return tui.RunChat(tui.ChatConfig{
		Runner:    serviceChatRunner{svc: svc},
		Agent:     a,
		Session:   sess,
		AgentsDir: paths.agentsDir,
	})
}

// withSignalContext registers SIGINT/SIGTERM cancellation for the REPL and
// returns the stop func for the caller to defer.
func withSignalContext(state *chatState) func() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	state.sigCtx = ctx
	return stop
}

func newChatReadline(home string) (*readline.Instance, error) {
	historyPath := filepath.Join(home, ".chat_history")
	return readline.NewEx(&readline.Config{
		Prompt:          tui.Prompt(),
		HistoryFile:     historyPath,
		HistoryLimit:    500,
		InterruptPrompt: "^C",
		EOFPrompt:       "/exit",
	})
}

// runChatLoop runs the interactive chat REPL until the user exits.
func runChatLoop(state *chatState) {
	for {
		line, err := state.readline.Readline()
		if err != nil {
			fmt.Println()
			return
		}
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "/") {
			if handleChatCommand(line, state) {
				return
			}
			continue
		}
		runChatTurn(line, state)
	}
}

// runChatTurn prepares and executes one user turn via the shared service.
func runChatTurn(line string, state *chatState) {
	runCtx, cancel := context.WithCancel(state.sigCtx)
	defer cancel()

	result, err := prepareChatTurn(line, state, runCtx)
	if err != nil {
		fmt.Println(tui.Error(err.Error()))
		return
	}
	defer result.Cleanup(state.svc.Registry)
	state.sess = result.Session

	events, err := loop.Run(runCtx, result.Config)
	if err != nil {
		fmt.Println(tui.Error(err.Error()))
		return
	}
	tui.PrintEvents(events)
}

// chatTurnRequest builds the PrepareRun request for one REPL turn.
func chatTurnRequest(line string, state *chatState) service.RunRequest {
	req := service.RunRequest{Agent: state.agent, Prompt: line, Source: "chat"}
	if state.sess == nil {
		return req
	}
	if state.svc.Store != nil {
		req.SessionID = state.sess.ID
		return req
	}
	req.Messages = state.sess.GetMessages()
	return req
}

// prepareChatTurn calls PrepareRun, falling back when the session was deleted.
func prepareChatTurn(line string, state *chatState, ctx context.Context) (*service.RunResult, error) {
	req := chatTurnRequest(line, state)
	result, err := state.svc.PrepareRun(ctx, req, CLIApprover{})
	if err == nil || state.sess == nil || !errors.Is(err, session.ErrNotFound) {
		return result, err
	}
	req.SessionID = ""
	req.Messages = state.sess.GetMessages()
	return state.svc.PrepareRun(ctx, req, CLIApprover{})
}

// handleChatCommand processes a /command input; returns true to exit the REPL.
func handleChatCommand(line string, state *chatState) (exit bool) {
	parts := strings.Fields(line)
	switch parts[0] {
	case "/exit", "/quit":
		fmt.Println(tui.Muted(i18n.T("tui.chat.goodbye")))
		return true
	case "/help":
		tui.PrintHelp()
		return false
	case "/clear":
		return handleClearCommand(state)
	case "/agent":
		return handleAgentCommand(parts, state)
	case "/tools":
		return handleToolsCommand(parts, state)
	default:
		fmt.Println(tui.Muted(i18n.T("tui.chat.unknown_command")))
		return false
	}
}

// handleClearCommand saves the current session and resets for the next turn.
func handleClearCommand(state *chatState) bool {
	if state.svc.Store != nil && state.sess != nil {
		if err := state.svc.Store.Save(state.sess); err != nil {
			fmt.Println(tui.Error(i18n.T("tui.chat.save_session_failed", "error", err.Error())))
		} else {
			fmt.Println(tui.Muted(i18n.T("tui.chat.session_saved", "id", state.sess.ID)))
		}
	}
	state.sess = nil
	fmt.Println(tui.Muted(i18n.T("tui.chat.session_cleared")))
	return false
}

// handleAgentCommand handles the /agent command — list or switch agents.
func handleAgentCommand(parts []string, state *chatState) bool {
	if len(parts) < 2 {
		return listChatAgents(state)
	}
	a, err := state.svc.GetAgent(parts[1])
	if err != nil {
		fmt.Println(tui.Error(err.Error()))
		return false
	}
	state.agent = a.ID
	state.sess = nil
	fmt.Println(tui.Success(i18n.T("tui.chat.agent_switched", "agent", a.Name, "provider", a.Provider, "model", a.Model)))
	return false
}

// listChatAgents prints the available agents with the active one marked.
func listChatAgents(state *chatState) bool {
	agents, err := agent.LoadAll(state.paths.agentsDir)
	if err != nil {
		fmt.Println(tui.Error(err.Error()))
		return false
	}
	if len(agents.Agents) == 0 {
		fmt.Println(tui.Muted(i18n.T("tui.chat.no_agents")))
		return false
	}
	fmt.Println(tui.Muted(i18n.T("tui.chat.agents_header")))
	for _, a := range agents.Agents {
		marker := " "
		if a.ID == state.agent || a.Name == state.agent {
			marker = tui.Success("●")
		}
		fmt.Printf("  %s %s\n", marker, a.Name)
	}
	return false
}

// handleToolsCommand handles the /tools command — list or validate tools.
func handleToolsCommand(parts []string, state *chatState) bool {
	if len(parts) >= 2 {
		a, err := agent.LoadByNameAndValidate(state.paths.agentsDir, parts[1], state.reg.Names())
		if err != nil {
			fmt.Println(tui.Error(err.Error()))
			return false
		}
		fmt.Println(tui.Success(i18n.T("tui.chat.tools_validated", "agent", a.Name, "tools", fmt.Sprint(a.Tools))))
		return false
	}
	names := state.reg.Names()
	if len(names) == 0 {
		fmt.Println(tui.Muted(i18n.T("tui.chat.no_tools")))
		return false
	}
	fmt.Println(tui.Muted(i18n.T("tui.chat.tools_header", "count", len(names))))
	for _, t := range state.reg.List() {
		fmt.Printf("  %-16s %s\n", tui.Muted(t.Name()), t.Description())
	}
	return false
}
