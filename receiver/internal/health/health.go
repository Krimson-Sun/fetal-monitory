package health

import (
	"context"
	"sync"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/status"
)

type HealthServer struct {
	grpc_health_v1.UnimplementedHealthServer
	mu       sync.RWMutex
	services map[string]grpc_health_v1.HealthCheckResponse_ServingStatus
}

func NewHealthServer() *HealthServer {
	return &HealthServer{
		services: make(map[string]grpc_health_v1.HealthCheckResponse_ServingStatus),
	}
}

func (h *HealthServer) Check(ctx context.Context, req *grpc_health_v1.HealthCheckRequest) (*grpc_health_v1.HealthCheckResponse, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	service := req.GetService()

	if service == "" {
		return &grpc_health_v1.HealthCheckResponse{
			Status: grpc_health_v1.HealthCheckResponse_SERVING,
		}, nil
	}

	servingStatus, exists := h.services[service]
	if !exists {
		return nil, status.Error(codes.NotFound, "service not found")
	}

	return &grpc_health_v1.HealthCheckResponse{
		Status: servingStatus,
	}, nil
}

func (h *HealthServer) Watch(req *grpc_health_v1.HealthCheckRequest, stream grpc_health_v1.Health_WatchServer) error {
	response, err := h.Check(stream.Context(), req)
	if err != nil {
		return err
	}

	if err := stream.Send(response); err != nil {
		return err
	}

	<-stream.Context().Done()
	return stream.Context().Err()
}

func (h *HealthServer) SetServingStatus(service string) {
	h.setStatus(service, grpc_health_v1.HealthCheckResponse_SERVING)
}

func (h *HealthServer) SetNotServingStatus(service string) {
	h.setStatus(service, grpc_health_v1.HealthCheckResponse_NOT_SERVING)
}

func (h *HealthServer) setStatus(service string, status grpc_health_v1.HealthCheckResponse_ServingStatus) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.services[service] = status
}
