package grpcapi

import (
	"context"
	"errors"
	"os"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/teexue/nexa/core/i18n"
	"github.com/teexue/nexakit/loop"
	"github.com/teexue/nexa/core/service"
	"github.com/teexue/nexakit/session"
	nexav1 "github.com/teexue/nexa/proto"
)

// Run executes an agent and streams events back to the client.
func (s *GRPCServer) Run(req *nexav1.RunRequest, stream grpc.ServerStreamingServer[nexav1.AgentEvent]) error {
	ctx := withRequestLocale(stream.Context())

	if err := s.checkAuth(ctx); err != nil {
		return err
	}

	if req.Agent == "" || req.Prompt == "" {
		return status.Error(codes.InvalidArgument, i18n.TCtx(ctx, "api.grpc.error.agent_prompt_required"))
	}

	result, err := s.svc.PrepareRun(ctx, service.RunRequest{
		Agent:     req.Agent,
		Prompt:    req.Prompt,
		SessionID: req.SessionId,
		Messages:  ProtoMessagesToProvider(req.Messages),
		Source:    "grpc",
	}, s.approver)
	if err != nil {
		return mapGRPCRunError(ctx, err)
	}
	defer result.Cleanup(s.registry)

	events, err := loop.Run(ctx, result.Config)
	if err != nil {
		return status.Error(codes.Internal, i18n.TCtx(ctx, "api.grpc.error.run", "error", err.Error()))
	}

	s.logger.Info("log.grpc.agent_run_started",
		"session_id", result.Session.ID,
		"agent", result.Config.Agent.Name,
		"provider", result.Config.Agent.Provider,
		"model", result.Config.Agent.Model)

	for ev := range events {
		if err := stream.Send(EventToProto(ev)); err != nil {
			return err
		}
	}

	return nil
}

func mapGRPCRunError(ctx context.Context, err error) error {
	var arg *service.ArgError
	if errors.As(err, &arg) {
		if arg.Field == "session_id" {
			return status.Error(codes.FailedPrecondition, i18n.TCtx(ctx, "api.grpc.error.session_not_configured"))
		}
		return status.Error(codes.InvalidArgument, err.Error())
	}
	if errors.Is(err, session.ErrNotFound) {
		return status.Error(codes.NotFound, err.Error())
	}
	if errors.Is(err, os.ErrNotExist) {
		return status.Error(codes.InvalidArgument, i18n.TCtx(ctx, "api.grpc.error.load_agent", "error", err.Error()))
	}
	var sev *service.ServerError
	if errors.As(err, &sev) {
		return status.Error(codes.Internal, i18n.TCtx(ctx, "api.grpc.error.create_provider", "error", sev.Message))
	}
	return status.Error(codes.Internal, i18n.TCtx(ctx, "api.grpc.error.run", "error", err.Error()))
}

// Approve resolves a pending tool approval.
func (s *GRPCServer) Approve(ctx context.Context, req *nexav1.ApproveRequest) (*nexav1.ApproveResponse, error) {
	ctx = withRequestLocale(ctx)
	if err := s.checkAuth(ctx); err != nil {
		return nil, err
	}
	if req.ApprovalId == "" {
		return nil, status.Error(codes.InvalidArgument, i18n.TCtx(ctx, "api.grpc.error.approval_id_required"))
	}

	resolved := s.approver.ResolveApproval(req.ApprovalId, req.Approved)
	if !resolved {
		return nil, status.Error(codes.NotFound, i18n.TCtx(ctx, "api.grpc.error.approval_not_found", "id", req.ApprovalId))
	}

	return &nexav1.ApproveResponse{
		Resolved:   true,
		ApprovalId: req.ApprovalId,
		Approved:   req.Approved,
	}, nil
}
