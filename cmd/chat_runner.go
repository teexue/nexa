package main

import (
	"context"
	"fmt"
	"sort"

	"github.com/teexue/nexakit/loop"
	"github.com/teexue/common-agent/core/service"
	"github.com/teexue/nexakit/session"
	"github.com/teexue/common-agent/core/tui"
)

// serviceChatRunner adapts service.Service to tui.ChatRunner.
type serviceChatRunner struct {
	svc *service.Service
}

func (r serviceChatRunner) RunTurn(ctx context.Context, req tui.TurnRequest, approver loop.Approver) (tui.TurnResult, error) {
	result, err := r.svc.PrepareRun(ctx, service.RunRequest{
		Agent: req.Agent, Prompt: req.Prompt, SessionID: req.SessionID,
		Messages: req.Messages, Model: req.Model, Provider: req.Provider,
		Source: "chat",
	}, approver)
	if err != nil {
		return tui.TurnResult{}, err
	}
	events, err := loop.Run(ctx, result.Config)
	if err != nil {
		result.Cleanup(r.svc.Registry)
		return tui.TurnResult{}, err
	}
	return tui.TurnResult{
		Session: result.Session,
		Events:  events,
		Cleanup: func() { result.Cleanup(r.svc.Registry) },
	}, nil
}

func (r serviceChatRunner) ListSessions() ([]session.SessionMeta, error) {
	if r.svc.Store == nil {
		return nil, nil
	}
	return r.svc.ListSessions("")
}

func (r serviceChatRunner) LoadSession(id string) (*session.Session, error) {
	if r.svc.Store == nil {
		return nil, fmt.Errorf("session persistence not configured")
	}
	return r.svc.LoadSession(id, "")
}

func (r serviceChatRunner) ListModels() []tui.ModelOption {
	if r.svc.Catalog == nil {
		return nil
	}
	entries := r.svc.Catalog.Entries()
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name < entries[j].Name })
	out := make([]tui.ModelOption, 0, len(entries)*2)
	for _, e := range entries {
		label := e.DisplayName
		if label == "" {
			label = e.Name
		}
		models := e.Models
		if len(models) == 0 && e.DefaultModel != "" {
			models = []string{e.DefaultModel}
		}
		for _, model := range models {
			out = append(out, tui.ModelOption{
				Provider: e.Name, Label: label, Model: model,
			})
		}
	}
	return out
}
