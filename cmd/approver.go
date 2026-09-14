package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/teexue/common-agent/core/i18n"
	"github.com/teexue/nexakit/loop"
	"github.com/teexue/common-agent/core/tui"
)

// CLIApprover prompts the user in the terminal to approve tool calls.
type CLIApprover struct{}

// Approve prompts the user and returns true if they approve.
func (CLIApprover) Approve(ctx context.Context, req loop.ApprovalRequest) bool {
	fmt.Printf("\n%s%s\n", tui.Muted("⚠"), i18n.T("tui.approval.tool_request", "name", req.Tool))
	fmt.Printf("%s\n", tui.Muted(string(req.Arguments)))
	fmt.Printf("%s%s", tui.Prompt(), i18n.T("tui.approval.prompt"))

	// Read user input in a goroutine so we can respect context cancellation.
	ch := make(chan string, 1)
	go func() {
		var input string
		_, _ = fmt.Scanln(&input)
		ch <- strings.TrimSpace(strings.ToLower(input))
	}()

	select {
	case <-ctx.Done():
		return false
	case input := <-ch:
		return input == "" || input == "y" || input == "yes"
	}
}
