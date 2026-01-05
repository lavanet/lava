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

func TestProviderRelayConnection_InterfaceCompliance(t *testing.T) {
	qosManager := &qos.QoSManager{}
	endpointConn := &EndpointConnection{}
	
	prc := &ProviderRelayConnection{
		EndpointConnection: endpointConn,
		QoSManager:         qosManager,
		EndpointAddress:    "provider.example.com:443",
	}

	// Verify it implements SessionConnection interface
	var _ SessionConnection = prc

	// Test interface methods
	assert.Equal(t, qosManager, prc.GetQoSManager())
	assert.True(t, prc.IsHealthy())
	assert.Equal(t, "provider.example.com:443", prc.GetEndpointAddress())
}

func TestProviderRelayConnection_IsHealthy(t *testing.T) {
	tests := []struct {
		name     string
		conn     *ProviderRelayConnection
		expected bool
	}{
		{
			name: "healthy with endpoint connection",
			conn: &ProviderRelayConnection{
				EndpointConnection: &EndpointConnection{},
				QoSManager:         &qos.QoSManager{},
			},
			expected: true,
		},
		{
			name: "unhealthy with nil endpoint connection",
			conn: &ProviderRelayConnection{
				EndpointConnection: nil,
				QoSManager:         &qos.QoSManager{},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.conn.IsHealthy())
		})
	}
}

func TestDirectRPCSessionConnection_InterfaceCompliance(t *testing.T) {
	ctx := context.Background()
	nodeUrl := common.NodeUrl{Url: "https://eth-mainnet.g.alchemy.com/v2/test"}
	
	directConn, err := NewDirectRPCConnection(ctx, nodeUrl, 5)
	require.NoError(t, err)
	
	qosManager := &qos.QoSManager{}
	
	drsc := &DirectRPCSessionConnection{
		DirectConnection: directConn,
		QoSManager:       qosManager,
		EndpointAddress:  "eth-mainnet.g.alchemy.com",
	}

	// Verify it implements SessionConnection interface
	var _ SessionConnection = drsc

	// Test interface methods
	assert.Equal(t, qosManager, drsc.GetQoSManager())
	assert.True(t, drsc.IsHealthy())
	assert.Equal(t, "eth-mainnet.g.alchemy.com", drsc.GetEndpointAddress())
}

func TestDirectRPCSessionConnection_IsHealthy(t *testing.T) {
	ctx := context.Background()
	nodeUrl := common.NodeUrl{Url: "https://test.example.com"}
	
	directConn, err := NewDirectRPCConnection(ctx, nodeUrl, 5)
	require.NoError(t, err)

	tests := []struct {
		name     string
		conn     *DirectRPCSessionConnection
		expected bool
	}{
		{
			name: "healthy with direct connection",
			conn: &DirectRPCSessionConnection{
				DirectConnection: directConn,
				QoSManager:       &qos.QoSManager{},
			},
			expected: true,
		},
		{
			name: "unhealthy with nil direct connection",
			conn: &DirectRPCSessionConnection{
				DirectConnection: nil,
				QoSManager:       &qos.QoSManager{},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.conn.IsHealthy())
		})
	}
}

func TestSingleConsumerSession_GetDirectConnection(t *testing.T) {
	ctx := context.Background()
	nodeUrl := common.NodeUrl{Url: "https://test.example.com"}
	
	directConn, err := NewDirectRPCConnection(ctx, nodeUrl, 5)
	require.NoError(t, err)

	// Test with DirectRPCSessionConnection
	drsc := &DirectRPCSessionConnection{
		DirectConnection: directConn,
		QoSManager:       &qos.QoSManager{},
	}

	session := &SingleConsumerSession{
		Connection: drsc,
	}

	conn, ok := session.GetDirectConnection()
	assert.True(t, ok)
	assert.Equal(t, directConn, conn)
}

func TestSingleConsumerSession_GetProviderConnection(t *testing.T) {
	endpointConn := &EndpointConnection{}
	prc := &ProviderRelayConnection{
		EndpointConnection: endpointConn,
		QoSManager:         &qos.QoSManager{},
	}

	session := &SingleConsumerSession{
		Connection: prc,
	}

	conn, ok := session.GetProviderConnection()
	assert.True(t, ok)
	assert.Equal(t, endpointConn, conn)
}

func TestSingleConsumerSession_IsDirectRPC(t *testing.T) {
	ctx := context.Background()
	nodeUrl := common.NodeUrl{Url: "https://test.example.com"}
	
	directConn, err := NewDirectRPCConnection(ctx, nodeUrl, 5)
	require.NoError(t, err)

	tests := []struct {
		name       string
		connection SessionConnection
		expected   bool
	}{
		{
			name: "direct RPC session",
			connection: &DirectRPCSessionConnection{
				DirectConnection: directConn,
				QoSManager:       &qos.QoSManager{},
			},
			expected: true,
		},
		{
			name: "provider relay session",
			connection: &ProviderRelayConnection{
				EndpointConnection: &EndpointConnection{},
				QoSManager:         &qos.QoSManager{},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			session := &SingleConsumerSession{
				Connection: tt.connection,
			}
			assert.Equal(t, tt.expected, session.IsDirectRPC())
		})
	}
}

func TestSingleConsumerSession_GetSessionQoSManager(t *testing.T) {
	qosManager := &qos.QoSManager{}

	tests := []struct {
		name       string
		session    *SingleConsumerSession
		expected   *qos.QoSManager
	}{
		{
			name: "gets QoS from connection",
			session: &SingleConsumerSession{
				Connection: &DirectRPCSessionConnection{
					QoSManager: qosManager,
				},
			},
			expected: qosManager,
		},
		{
			name: "fallback to legacy QoSManager field",
			session: &SingleConsumerSession{
				Connection: nil,
				QoSManager: qosManager,
			},
			expected: qosManager,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.session.GetSessionQoSManager()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestEndpoint_IsDirectRPC(t *testing.T) {
	ctx := context.Background()
	nodeUrl := common.NodeUrl{Url: "https://test.example.com"}
	
	directConn, err := NewDirectRPCConnection(ctx, nodeUrl, 5)
	require.NoError(t, err)

	tests := []struct {
		name     string
		endpoint *Endpoint
		expected bool
	}{
		{
			name: "endpoint with direct connections",
			endpoint: &Endpoint{
				DirectConnections: []DirectRPCConnection{directConn},
			},
			expected: true,
		},
		{
			name: "endpoint with provider connections",
			endpoint: &Endpoint{
				Connections: []*EndpointConnection{{}},
			},
			expected: false,
		},
		{
			name: "endpoint with no connections",
			endpoint: &Endpoint{},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.endpoint.IsDirectRPC())
		})
	}
}

func TestEndpoint_IsProviderRelay(t *testing.T) {
	ctx := context.Background()
	nodeUrl := common.NodeUrl{Url: "https://test.example.com"}
	
	directConn, err := NewDirectRPCConnection(ctx, nodeUrl, 5)
	require.NoError(t, err)

	tests := []struct {
		name     string
		endpoint *Endpoint
		expected bool
	}{
		{
			name: "endpoint with provider connections",
			endpoint: &Endpoint{
				Connections: []*EndpointConnection{{}},
			},
			expected: true,
		},
		{
			name: "endpoint with direct connections",
			endpoint: &Endpoint{
				DirectConnections: []DirectRPCConnection{directConn},
			},
			expected: false,
		},
		{
			name: "endpoint with no connections",
			endpoint: &Endpoint{},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.endpoint.IsProviderRelay())
		})
	}
}

func TestSessionConnection_TypeSafety(t *testing.T) {
	// This test verifies that you cannot accidentally mix connection types
	ctx := context.Background()
	nodeUrl := common.NodeUrl{Url: "https://test.example.com"}
	
	directConn, err := NewDirectRPCConnection(ctx, nodeUrl, 5)
	require.NoError(t, err)

	// Create a session with DirectRPCSessionConnection
	directSession := &SingleConsumerSession{
		Connection: &DirectRPCSessionConnection{
			DirectConnection: directConn,
			QoSManager:       &qos.QoSManager{},
		},
	}

	// Should NOT be able to get provider connection
	_, ok := directSession.GetProviderConnection()
	assert.False(t, ok, "DirectRPC session should not return provider connection")

	// Create a session with ProviderRelayConnection
	providerSession := &SingleConsumerSession{
		Connection: &ProviderRelayConnection{
			EndpointConnection: &EndpointConnection{},
			QoSManager:         &qos.QoSManager{},
		},
	}

	// Should NOT be able to get direct connection
	_, ok = providerSession.GetDirectConnection()
	assert.False(t, ok, "Provider relay session should not return direct connection")
}

func TestGetConsumerSessionInstanceFromEndpoint_Integration(t *testing.T) {
	// This test verifies that the Connection field is properly set when creating
	// sessions through GetConsumerSessionInstanceFromEndpoint (the actual production path)
	
	// Initialize random seed (required by session creation)
	rand.InitRandomSeed()
	
	// Create a ConsumerSessionsWithProvider
	cswp := &ConsumerSessionsWithProvider{
		Sessions:      make(map[int64]*SingleConsumerSession),
		PairingEpoch:  100,
		StaticProvider: false,
		PublicLavaAddress: "lava@test123",
	}

	// Create a mock endpoint connection
	endpointConnection := &EndpointConnection{}
	qosManager := &qos.QoSManager{}
	networkAddress := "provider.example.com:443"

	// Get/Create a session (this is the real production path)
	session, epoch, err := cswp.GetConsumerSessionInstanceFromEndpoint(
		endpointConnection,
		0, // numberOfResets
		qosManager,
		networkAddress,
	)

	// Verify session creation succeeded
	require.NoError(t, err)
	require.NotNil(t, session)
	assert.Equal(t, uint64(100), epoch)

	// CRITICAL: Verify Connection field is populated (not nil)
	assert.NotNil(t, session.Connection, "Connection field must be set in production session creation")

	// Verify it's a provider-relay connection
	assert.False(t, session.IsDirectRPC(), "Session should be provider-relay, not direct RPC")

	// Verify GetProviderConnection() works
	providerConn, ok := session.GetProviderConnection()
	assert.True(t, ok, "GetProviderConnection() should return true for provider-relay session")
	assert.NotNil(t, providerConn, "Provider connection should not be nil")
	assert.Equal(t, endpointConnection, providerConn, "Provider connection should match the original")

	// Verify GetDirectConnection() returns false (type safety)
	_, ok = session.GetDirectConnection()
	assert.False(t, ok, "GetDirectConnection() should return false for provider-relay session")

	// Verify QoS manager is accessible
	sessionQoS := session.GetSessionQoSManager()
	assert.Equal(t, qosManager, sessionQoS, "QoS manager should be accessible via Connection")

	// Verify network address is stored correctly
	assert.Equal(t, networkAddress, session.Connection.GetEndpointAddress())

	// Verify backward compatibility - legacy fields are still set
	assert.Equal(t, endpointConnection, session.EndpointConnection, "Legacy EndpointConnection field should still be set")
	assert.Equal(t, qosManager, session.QoSManager, "Legacy QoSManager field should still be set")
}

func TestSingleConsumerSession_Free_WithNilEndpointConnection(t *testing.T) {
	// Test that Free() doesn't panic when EndpointConnection is nil (direct RPC session)
	session := &SingleConsumerSession{
		EndpointConnection: nil, // Direct RPC session won't have this
		Connection: &DirectRPCSessionConnection{
			QoSManager: &qos.QoSManager{},
		},
	}

	// This should not panic
	session.lock.Lock()
	assert.NotPanics(t, func() {
		session.Free(nil)
	})
}

func TestSingleConsumerSession_TryUseSession_WithNilEndpointConnection(t *testing.T) {
	// Test that TryUseSession() doesn't panic when EndpointConnection is nil (direct RPC session)
	session := &SingleConsumerSession{
		EndpointConnection: nil, // Direct RPC session won't have this
		Connection: &DirectRPCSessionConnection{
			QoSManager: &qos.QoSManager{},
		},
		BlockListed: false,
	}

	// This should not panic and should return success
	blocked, ok := session.TryUseSession()
	assert.False(t, blocked)
	assert.True(t, ok)
	
	// Clean up
	session.Free(nil)
}
