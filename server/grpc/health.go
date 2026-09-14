package grpcapi

import (
	"context"
	"sync"

	"google.golang.org/grpc/health/grpc_health_v1"

	nexav1 "github.com/teexue/nexa/proto"
)

var _ grpc_health_v1.HealthServer = (*grpcHealthService)(nil)

// grpcHealthService implements grpc_health_v1.HealthServer with
// component-level readiness checks from telemetry.HealthServer.
type grpcHealthService struct {
	grpc_health_v1.UnimplementedHealthServer
	grpcSrv  *GRPCServer
	mu       sync.RWMutex
	statuses map[string]grpc_health_v1.HealthCheckResponse_ServingStatus
}

// Check returns the serving status for the requested service.
// Before checking, it refreshes status based on registered component checkers.
func (h *grpcHealthService) Check(ctx context.Context, req *grpc_health_v1.HealthCheckRequest) (*grpc_health_v1.HealthCheckResponse, error) {
	h.refreshStatus(ctx)

	h.mu.RLock()
	status, ok := h.statuses[req.Service]
	h.mu.RUnlock()

	if !ok {
		return &grpc_health_v1.HealthCheckResponse{
			Status: grpc_health_v1.HealthCheckResponse_SERVICE_UNKNOWN,
		}, nil
	}
	return &grpc_health_v1.HealthCheckResponse{Status: status}, nil
}

// Watch streams health status changes. For simplicity, it sends the current
// status and then waits for context cancellation (no change notifications).
func (h *grpcHealthService) Watch(req *grpc_health_v1.HealthCheckRequest, stream grpc_health_v1.Health_WatchServer) error {
	resp, err := h.Check(stream.Context(), req)
	if err != nil {
		return err
	}
	if err := stream.Send(resp); err != nil {
		return err
	}
	// Block until client disconnects (no change notifications implemented).
	<-stream.Context().Done()
	return stream.Context().Err()
}

// refreshStatus checks all registered telemetry components and updates
// the gRPC health serving status accordingly.
func (h *grpcHealthService) refreshStatus(ctx context.Context) {
	h.grpcSrv.healthMu.RLock()
	health := h.grpcSrv.health
	h.grpcSrv.healthMu.RUnlock()

	if health == nil {
		return
	}

	status := grpc_health_v1.HealthCheckResponse_SERVING
	if err := health.CheckAll(ctx); err != nil {
		status = grpc_health_v1.HealthCheckResponse_NOT_SERVING
	}

	h.mu.Lock()
	h.statuses[""] = status
	h.statuses[nexav1.AgentService_ServiceDesc.ServiceName] = status
	h.mu.Unlock()
}
