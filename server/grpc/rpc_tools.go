package grpcapi

import (
	"context"
	"encoding/json"

	nexav1 "github.com/teexue/nexa/proto"
)

// ListTools returns all registered tools.
func (s *GRPCServer) ListTools(ctx context.Context, _ *nexav1.ListToolsRequest) (*nexav1.ListToolsResponse, error) {
	ctx = withRequestLocale(ctx)
	if err := s.checkAuth(ctx); err != nil {
		return nil, err
	}
	tools := s.registry.List()
	result := make([]*nexav1.ToolInfo, len(tools))
	for i, t := range tools {
		params, _ := json.Marshal(t.InputSchema())
		result[i] = &nexav1.ToolInfo{
			Name:        t.Name(),
			Description: t.Description(),
			Parameters:  params,
		}
	}
	return &nexav1.ListToolsResponse{Tools: result}, nil
}
