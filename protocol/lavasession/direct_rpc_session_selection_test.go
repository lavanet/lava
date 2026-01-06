package lavasession

import (
	"context"
	"testing"

	"github.com/lavanet/lava/v5/protocol/common"
	"github.com/lavanet/lava/v5/protocol/qos"
	"github.com/lavanet/lava/v5/utils/rand"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetConsumerSessionInstanceFromEndpoint_DirectRPC(t *testing.T) {
	// This test verifies that direct RPC sessions are created when endpointConnection is nil
	
	// Initialize random seed
	rand.InitRandomSeed()
	
	ctx := context.Background()
	nodeUrl := common.NodeUrl{Url: "https://eth-mainnet.g.alchemy.com/v2/test-key"}
	
	// Create direct RPC connection
	directConn, err := NewDirectRPCConnection(ctx, nodeUrl, 5)
	require.NoError(t, err)
	
	// Create endpoint with DirectConnections (smart router mode)
	endpoint := &Endpoint{
		NetworkAddress:    nodeUrl.Url,
		Enabled:           true,
		DirectConnections: []DirectRPCConnection{directConn},
		Connections:       nil, // No provider-relay connections
	}
	
	// Create ConsumerSessionsWithProvider with direct RPC endpoint
	cswp := &ConsumerSessionsWithProvider{
		Sessions:          make(map[int64]*SingleConsumerSession),
		PairingEpoch:      100,
		StaticProvider:    true,
		PublicLavaAddress: "ethereum-alchemy",
		Endpoints:         []*Endpoint{endpoint},
	}
	
	qosManager := &qos.QoSManager{}
	
	// Call GetConsumerSessionInstanceFromEndpoint with nil endpointConnection (direct RPC mode)
	session, epoch, err := cswp.GetConsumerSessionInstanceFromEndpoint(
		nil,                 // endpointConnection = nil triggers direct RPC mode
		0,                   // numberOfResets
		qosManager,          // qosManager
		nodeUrl.Url,         // networkAddress
	)
	
	// Verify session creation succeeded
	require.NoError(t, err)
	require.NotNil(t, session)
	assert.Equal(t, uint64(100), epoch)
	
	// CRITICAL: Verify this is a direct RPC session
	assert.True(t, session.IsDirectRPC(), "Session should be direct RPC")
	
	// Verify Connection field is DirectRPCSessionConnection
	assert.NotNil(t, session.Connection, "Connection field must be set")
	_, ok := session.Connection.(*DirectRPCSessionConnection)
	assert.True(t, ok, "Connection should be DirectRPCSessionConnection type")
	
	// Verify GetDirectConnection works
	conn, ok := session.GetDirectConnection()
	assert.True(t, ok, "GetDirectConnection should return true")
	assert.Equal(t, directConn, conn, "Should return the same DirectRPCConnection")
	
	// Verify legacy EndpointConnection is nil (no provider-relay)
	assert.Nil(t, session.EndpointConnection, "EndpointConnection should be nil for direct RPC")
	
	// Verify network address matches
	assert.Equal(t, nodeUrl.Url, session.Connection.GetEndpointAddress())
}

func TestGetConsumerSessionInstanceFromEndpoint_ProviderRelay(t *testing.T) {
	// This test verifies that provider-relay sessions still work (backward compatibility)
	
	// Initialize random seed
	rand.InitRandomSeed()
	
	// Create endpoint connection (provider-relay mode)
	endpointConn := &EndpointConnection{}
	networkAddress := "provider.example.com:443"
	
	// Create endpoint with Connections (consumer mode)
	endpoint := &Endpoint{
		NetworkAddress:    networkAddress,
		Enabled:           true,
		Connections:       []*EndpointConnection{endpointConn},
		DirectConnections: nil, // No direct connections
	}
	
	// Create ConsumerSessionsWithProvider with provider-relay endpoint
	cswp := &ConsumerSessionsWithProvider{
		Sessions:          make(map[int64]*SingleConsumerSession),
		PairingEpoch:      100,
		StaticProvider:    false,
		PublicLavaAddress: "lava@provider123",
		Endpoints:         []*Endpoint{endpoint},
	}
	
	qosManager := &qos.QoSManager{}
	
	// Call GetConsumerSessionInstanceFromEndpoint with actual endpointConnection (provider-relay mode)
	session, epoch, err := cswp.GetConsumerSessionInstanceFromEndpoint(
		endpointConn,   // endpointConnection != nil triggers provider-relay mode
		0,              // numberOfResets
		qosManager,     // qosManager
		networkAddress, // networkAddress
	)
	
	// Verify session creation succeeded
	require.NoError(t, err)
	require.NotNil(t, session)
	assert.Equal(t, uint64(100), epoch)
	
	// CRITICAL: Verify this is a provider-relay session
	assert.False(t, session.IsDirectRPC(), "Session should be provider-relay, not direct RPC")
	
	// Verify Connection field is ProviderRelayConnection
	assert.NotNil(t, session.Connection, "Connection field must be set")
	_, ok := session.Connection.(*ProviderRelayConnection)
	assert.True(t, ok, "Connection should be ProviderRelayConnection type")
	
	// Verify GetProviderConnection works
	conn, ok := session.GetProviderConnection()
	assert.True(t, ok, "GetProviderConnection should return true")
	assert.Equal(t, endpointConn, conn, "Should return the same EndpointConnection")
	
	// Verify legacy EndpointConnection is set (backward compatibility)
	assert.Equal(t, endpointConn, session.EndpointConnection, "EndpointConnection should be set for provider-relay")
}

func TestFetchEndpointConnection_DirectRPC(t *testing.T) {
	// This test verifies that fetchEndpointConnectionFromConsumerSessionWithProvider
	// properly handles direct RPC endpoints
	
	ctx := context.Background()
	nodeUrl := common.NodeUrl{Url: "https://test.example.com"}
	
	// Create direct RPC connection
	directConn, err := NewDirectRPCConnection(ctx, nodeUrl, 5)
	require.NoError(t, err)
	
	// Create endpoint with DirectConnections
	endpoint := &Endpoint{
		NetworkAddress:    nodeUrl.Url,
		Enabled:           true,
		DirectConnections: []DirectRPCConnection{directConn},
		Connections:       nil,
	}
	
	// Create ConsumerSessionsWithProvider
	cswp := &ConsumerSessionsWithProvider{
		Sessions:          make(map[int64]*SingleConsumerSession),
		PairingEpoch:      100,
		PublicLavaAddress: "test-provider",
		Endpoints:         []*Endpoint{endpoint},
	}
	
	// Call fetchEndpointConnectionFromConsumerSessionWithProvider
	connected, endpoints, providerAddr, err := cswp.fetchEndpointConnectionFromConsumerSessionWithProvider(
		ctx,
		false, // retryDisabledEndpoints
		false, // getAllEndpoints (get one endpoint)
		"",    // addon
		nil,   // extensionNames
	)
	
	// Verify connection succeeded
	require.NoError(t, err)
	assert.True(t, connected, "Should be connected for direct RPC")
	assert.Equal(t, "test-provider", providerAddr)
	require.Len(t, endpoints, 1, "Should return one endpoint")
	
	// For direct RPC, chosenEndpointConnection should be nil
	assert.Nil(t, endpoints[0].chosenEndpointConnection, "Direct RPC should have nil chosenEndpointConnection")
	assert.NotNil(t, endpoints[0].endpoint, "Should have endpoint reference")
	assert.True(t, endpoints[0].endpoint.IsDirectRPC(), "Endpoint should be direct RPC")
}
