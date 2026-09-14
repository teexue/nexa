package grpcapi

import (
	"context"
	"errors"
	"os"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/teexue/nexa/core/i18n"
	"github.com/teexue/nexa/core/service"
	nexav1 "github.com/teexue/nexa/proto"
)

// ListAgents returns all loaded agents.
func (s *GRPCServer) ListAgents(ctx context.Context, _ *nexav1.ListAgentsRequest) (*nexav1.ListAgentsResponse, error) {
	ctx = withRequestLocale(ctx)
	if err := s.checkAuth(ctx); err != nil {
		return nil, err
	}
	summaries := s.svc.ListAgents()
	items := make([]*nexav1.AgentListItem, len(summaries))
	for i, a := range summaries {
		items[i] = &nexav1.AgentListItem{
			Name:     a.Name,
			Provider: a.Provider,
			Model:    a.Model,
			Tools:    a.Tools,
			MaxTurns: int32(a.MaxTurns),
		}
	}
	return &nexav1.ListAgentsResponse{Agents: items}, nil
}

// GetAgent returns details for a specific agent.
func (s *GRPCServer) GetAgent(ctx context.Context, req *nexav1.GetAgentRequest) (*nexav1.GetAgentResponse, error) {
	ctx = withRequestLocale(ctx)
	if err := s.checkAuth(ctx); err != nil {
		return nil, err
	}
	a, err := s.svc.GetAgent(req.Name)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, status.Error(codes.NotFound, i18n.TCtx(ctx, "api.grpc.error.agent_not_found", "name", req.Name))
		}
		return nil, status.Error(codes.InvalidArgument, i18n.TCtx(ctx, "api.grpc.error.load_agent", "error", err.Error()))
	}

	return &nexav1.GetAgentResponse{
		Name:         a.Name,
		Provider:     a.Provider,
		Model:        a.Model,
		SystemPrompt: a.SystemPrompt,
		Tools:        a.Tools,
		MaxTurns:     int32(a.MaxTurns),
		MaxTokens:    int32(a.MaxTokens),
	}, nil
}

// UpdateAgent creates or updates an agent YAML.
func (s *GRPCServer) UpdateAgent(ctx context.Context, req *nexav1.UpdateAgentRequest) (*nexav1.UpdateAgentResponse, error) {
	ctx = withRequestLocale(ctx)
	if err := s.checkAuth(ctx); err != nil {
		return nil, err
	}
	if err := s.svc.SaveAgent(req.Name, req.YamlContent); err != nil {
		return nil, status.Error(codes.InvalidArgument, i18n.TCtx(ctx, "api.grpc.error.save_agent", "error", err.Error()))
	}
	return &nexav1.UpdateAgentResponse{Name: service.NormalizeAgentName(req.Name)}, nil
}

// DeleteAgent deletes an agent YAML.
func (s *GRPCServer) DeleteAgent(ctx context.Context, req *nexav1.DeleteAgentRequest) (*nexav1.DeleteAgentResponse, error) {
	ctx = withRequestLocale(ctx)
	if err := s.checkAuth(ctx); err != nil {
		return nil, err
	}
	if err := s.svc.DeleteAgent(req.Name); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, status.Error(codes.NotFound, i18n.TCtx(ctx, "api.grpc.error.agent_not_found", "name", req.Name))
		}
		return nil, status.Error(codes.Internal, i18n.TCtx(ctx, "api.grpc.error.delete_agent", "error", err.Error()))
	}
	return &nexav1.DeleteAgentResponse{}, nil
}
