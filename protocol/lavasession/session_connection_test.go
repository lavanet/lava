package lavasession

import (
	"testing"

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

func TestGetConsumerSessionInstanceFromEndpoint_Integration(t *testing.T) {
	// This test verifies that the Connection field is properly set when creating
	// sessions through GetConsumerSessionInstanceFromEndpoint (the actual production path)

	// Initialize random seed (required by session creation)
	rand.InitRandomSeed()

	// Create a ConsumerSessionsWithProvider
	cswp := &ConsumerSessionsWithProvider{
		Sessions:          make(map[int64]*SingleConsumerSession),
		PairingEpoch:      100,
		StaticProvider:    false,
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

	// Verify GetProviderConnection() works
	providerConn, ok := session.GetProviderConnection()
	assert.True(t, ok, "GetProviderConnection() should return true for provider-relay session")
	assert.NotNil(t, providerConn, "Provider connection should not be nil")
	assert.Equal(t, endpointConnection, providerConn, "Provider connection should match the original")

	// Verify QoS manager is accessible
	sessionQoS := session.GetSessionQoSManager()
	assert.Equal(t, qosManager, sessionQoS, "QoS manager should be accessible via Connection")

	// Verify network address is stored correctly
	assert.Equal(t, networkAddress, session.Connection.GetEndpointAddress())

	// Verify backward compatibility - legacy fields are still set
	assert.Equal(t, endpointConnection, session.EndpointConnection, "Legacy EndpointConnection field should still be set")
	assert.Equal(t, qosManager, session.QoSManager, "Legacy QoSManager field should still be set")
}
