package tui

import (
	"context"

	"github.com/teexue/nexakit/event"
	"github.com/teexue/nexakit/loop"
	"github.com/teexue/nexakit/provider"
	"github.com/teexue/nexakit/session"
)

// ModelOption is one selectable provider+model pair for the /model picker.
type ModelOption struct {
	Provider string
	Label    string
	Model    string
}

// TurnRequest is one user prompt handed to ChatRunner.
type TurnRequest struct {
	Agent     string
	Prompt    string
	SessionID string
	Messages  []provider.Message
	// Model/Provider override the agent defaults for this turn (session lock
	// still wins inside PrepareRun).
	Model    string
	Provider string
}

// TurnResult is a live event stream plus the session used for the turn.
type TurnResult struct {
	Session *session.Session
	Events  <-chan event.Event
	Cleanup func()
}

// ChatRunner abstracts PrepareRun + loop.Run so core/tui does not import
// core/service (which would cycle through config → tui).
type ChatRunner interface {
	RunTurn(ctx context.Context, req TurnRequest, approver loop.Approver) (TurnResult, error)
	ListSessions() ([]session.Meta, error)
	LoadSession(id string) (*session.Session, error)
	// ListModels returns enabled provider/model pairs for the /model picker.
	ListModels() []ModelOption
}
