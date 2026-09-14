package grpcapi

import (
	"context"
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/teexue/nexa/core/auth"
	"github.com/teexue/nexa/core/i18n"
	"github.com/teexue/nexakit/session"
	nexav1 "github.com/teexue/nexa/proto"
)

// ListSessions returns all persisted sessions.
func (s *GRPCServer) ListSessions(ctx context.Context, _ *nexav1.ListSessionsRequest) (*nexav1.ListSessionsResponse, error) {
	ctx = withRequestLocale(ctx)
	if err := s.checkAuth(ctx); err != nil {
		return nil, err
	}
	metas, err := s.svc.ListSessions(auth.IdentityFromContext(ctx).UserID)
	if err != nil {
		return nil, status.Error(codes.FailedPrecondition, i18n.TCtx(ctx, "api.grpc.error.failed_precondition", "error", err.Error()))
	}

	items := make([]*nexav1.SessionMeta, len(metas))
	for i, m := range metas {
		items[i] = &nexav1.SessionMeta{
			Id:        m.ID,
			AgentName: m.Agent,
			UpdatedAt: m.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
		}
	}
	return &nexav1.ListSessionsResponse{Sessions: items}, nil
}

// GetSession returns a specific session with its messages.
func (s *GRPCServer) GetSession(ctx context.Context, req *nexav1.GetSessionRequest) (*nexav1.GetSessionResponse, error) {
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
	protoMsgs := make([]*nexav1.Message, len(msgs))
	for i, m := range msgs {
		protoMsgs[i] = &nexav1.Message{
			Role:    string(m.Role),
			Content: m.Content,
		}
	}

	return &nexav1.GetSessionResponse{
		Id:        sess.ID,
		AgentName: sess.Agent,
		Messages:  protoMsgs,
	}, nil
}

// DeleteSession deletes a persisted session.
func (s *GRPCServer) DeleteSession(ctx context.Context, req *nexav1.DeleteSessionRequest) (*nexav1.DeleteSessionResponse, error) {
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
	return &nexav1.DeleteSessionResponse{}, nil
}
