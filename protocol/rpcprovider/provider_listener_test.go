package rpcprovider

import (
	"context"
	"testing"

	"github.com/lavanet/lava/v3/protocol/lavasession"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health/grpc_health_v1"
)

func TestHealthCheck(t *testing.T) {
	address := "0.0.0.0:2220"

	networkAddressData := lavasession.NetworkAddressData{
		Address:    address,
		DisableTLS: true,
	}

	providerListener := NewProviderListener(context.Background(), networkAddressData, "/lava/health")

	defer providerListener.Shutdown(context.Background())

	conn, err := grpc.NewClient(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("GRPC new client failed: %v", err)
	}

	client := grpc_health_v1.NewHealthClient(conn)
	resp, err := client.Check(context.Background(), &grpc_health_v1.HealthCheckRequest{})
	if err != nil {
		t.Fatalf("Health check failed: %v", err)
	}

	if resp.Status != grpc_health_v1.HealthCheckResponse_SERVING {
		t.Errorf("Unexpected health status: got %v, want %v", resp.Status, grpc_health_v1.HealthCheckResponse_SERVING)
	}
}
