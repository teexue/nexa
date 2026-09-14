package grpcapi

import (
	"context"
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/teexue/common-agent/core/auth"
	"github.com/teexue/common-agent/core/i18n"
	"github.com/teexue/nexakit/session"
	commonagentv1 "github.com/teexue/common-agent/proto"
)

// ListSessions returns all persisted sessions.
func (s *GRPCServer) ListSessions(ctx context.Context, _ *commonagentv1.ListSessionsRequest) (*commonagentv1.ListSessionsResponse, error) {
	ctx = withRequestLocale(ctx)
	if err := s.checkAuth(ctx); err != nil {
		return nil, err
	}
	metas, err := s.svc.ListSessions(auth.IdentityFromContext(ctx).UserID)
	if err != nil {
		return nil, status.Error(codes.FailedPrecondition, i18n.TCtx(ctx, "api.grpc.error.failed_precondition", "error", err.Error()))
	}

	items := make([]*commonagentv1.SessionMeta, len(metas))
	for i, m := range metas {
		items[i] = &commonagentv1.SessionMeta{
			Id:        m.ID,
			AgentName: m.Agent,
			UpdatedAt: m.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
		}
	}
	return &commonagentv1.ListSessionsResponse{Sessions: items}, nil
}

// GetSession returns a specific session with its messages.
func (s *GRPCServer) GetSession(ctx context.Context, req *commonagentv1.GetSessionRequest) (*commonagentv1.GetSessionResponse, error) {
	ctx = withRequestLocale(ctx)
	if err := s.checkAuth(ctx); err != nil {
		return nil, err
	}
	sess, err := s.svc.LoadSession(req.Id, auth.IdentityFromContext(ctx).UserID)
	if err != nil {
		if errors.Is(err, session.ErrNotFound) {
			return nil, status.Error(codes.NotFound, i18n.TCtx(ctx, "api.grpc.error.session_not_found", "id", req.Id))
		}
		return nil, status.Error(codes.InvalidArgument, i18n.TCtx(ctx, "api.grpc.error.invalid_argument", "error", err.Error()))
	}

	msgs := sess.GetMessages()
	protoMsgs := make([]*commonagentv1.Message, len(msgs))
	for i, m := range msgs {
		protoMsgs[i] = &commonagentv1.Message{
			Role:    string(m.Role),
			Content: m.Content,
		}
	}

	return &commonagentv1.GetSessionResponse{
		Id:        sess.ID,
		AgentName: sess.Agent,
		Messages:  protoMsgs,
	}, nil
}

// DeleteSession deletes a persisted session.
func (s *GRPCServer) DeleteSession(ctx context.Context, req *commonagentv1.DeleteSessionRequest) (*commonagentv1.DeleteSessionResponse, error) {
	ctx = withRequestLocale(ctx)
	if err := s.checkAuth(ctx); err != nil {
		return nil, err
	}
	if err := s.svc.DeleteSession(req.Id, auth.IdentityFromContext(ctx).UserID); err != nil {
		if errors.Is(err, session.ErrNotFound) {
			return nil, status.Error(codes.NotFound, i18n.TCtx(ctx, "api.grpc.error.session_not_found", "id", req.Id))
		}
		return nil, status.Error(codes.InvalidArgument, i18n.TCtx(ctx, "api.grpc.error.invalid_argument", "error", err.Error()))
	}
	return &commonagentv1.DeleteSessionResponse{}, nil
}
