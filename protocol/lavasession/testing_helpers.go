package lavasession

import (
	"context"
	"strconv"

	"github.com/lavanet/lava/v5/utils"
	pairingtypes "github.com/lavanet/lava/v5/x/pairing/types"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// NewMockEndpointConnection creates an EndpointConnection for testing purposes with a mock grpc connection
func NewMockEndpointConnection(client pairingtypes.RelayerClient) (*EndpointConnection, error) {
	// Create a non-blocking connection that won't actually try to connect
	mockConn, err := grpc.DialContext(
		context.Background(),
		"mock-address:443",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithReturnConnectionError(),
	)
	if err != nil {
		return nil, err
	}

	return &EndpointConnection{
		Client:       client,
		connection:   mockConn,
		lbUniqueId:   strconv.FormatUint(utils.GenerateUniqueIdentifier(), 10),
		disconnected: false,
	}, nil
}
