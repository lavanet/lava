package rpcprovider

import (
	"context"
	"testing"

	"github.com/lavanet/lava/v3/protocol/lavasession"
	"github.com/stretchr/testify/require"
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

	healthcheckPath := "/lava/health"

	providerListener := NewProviderListener(context.Background(), networkAddressData, healthcheckPath)

	defer providerListener.Shutdown(context.Background())

	conn, err := grpc.NewClient(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)

	client := grpc_health_v1.NewHealthClient(conn)
	resp, err := client.Check(context.Background(), &grpc_health_v1.HealthCheckRequest{})
	require.NoError(t, err)

	require.Equal(t, grpc_health_v1.HealthCheckResponse_SERVING, resp.Status, "Unexpected health status")
}
