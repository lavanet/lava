package badgegenerator

import (
	"golang.org/x/net/context"
	health "google.golang.org/grpc/health/grpc_health_v1"
)

type HealthServer struct {
	health.UnimplementedHealthServer
}

func (s *HealthServer) Check(ctx context.Context, in *health.HealthCheckRequest) (*health.HealthCheckResponse, error) {
	return &health.HealthCheckResponse{Status: health.HealthCheckResponse_SERVING}, nil
}

func (s *HealthServer) Watch(in *health.HealthCheckRequest, _ health.Health_WatchServer) error {
	return nil
}
