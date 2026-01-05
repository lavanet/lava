package lavasession

import (
	"context"
	"testing"

	"github.com/lavanet/lava/v5/protocol/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDetectProtocol(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected DirectRPCProtocol
		wantErr  bool
	}{
		{
			name:     "HTTP protocol",
			url:      "http://localhost:8545",
			expected: DirectRPCProtocolHTTP,
			wantErr:  false,
		},
		{
			name:     "HTTPS protocol",
			url:      "https://mainnet.infura.io",
			expected: DirectRPCProtocolHTTPS,
			wantErr:  false,
		},
		{
			name:     "WebSocket protocol",
			url:      "ws://localhost:8546",
			expected: DirectRPCProtocolWS,
			wantErr:  false,
		},
		{
			name:     "WebSocket Secure protocol",
			url:      "wss://eth-mainnet.g.alchemy.com/v2/KEY",
			expected: DirectRPCProtocolWSS,
			wantErr:  false,
		},
		{
			name:     "gRPC protocol",
			url:      "grpc://localhost:9090",
			expected: DirectRPCProtocolGRPC,
			wantErr:  false,
		},
		{
			name:     "gRPCs protocol",
			url:      "grpcs://localhost:9090",
			expected: DirectRPCProtocolGRPC,
			wantErr:  false,
		},
		{
			name:     "No scheme defaults to HTTPS",
			url:      "mainnet.infura.io",
			expected: DirectRPCProtocolHTTPS,
			wantErr:  false,
		},
		{
			name:     "Unsupported protocol",
			url:      "ftp://example.com",
			expected: "",
			wantErr:  true,
		},
		{
			name:     "Invalid URL",
			url:      "://invalid",
			expected: "",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			protocol, err := DetectProtocol(tt.url)
			
			if tt.wantErr {
				require.Error(t, err)
				assert.Equal(t, DirectRPCProtocol(""), protocol)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, protocol)
			}
		})
	}
}

func TestHTTPConnectionCreation(t *testing.T) {
	ctx := context.Background()
	nodeUrl := common.NodeUrl{Url: "http://localhost:8545"}

	conn, err := NewDirectRPCConnection(ctx, nodeUrl, 5)
	require.NoError(t, err)
	require.NotNil(t, conn)

	assert.Equal(t, DirectRPCProtocolHTTP, conn.GetProtocol())
	assert.True(t, conn.IsHealthy())
	assert.Equal(t, "http://localhost:8545", conn.GetURL())

	err = conn.Close()
	assert.NoError(t, err)
}

func TestHTTPSConnectionCreation(t *testing.T) {
	ctx := context.Background()
	nodeUrl := common.NodeUrl{Url: "https://eth-mainnet.g.alchemy.com/v2/test"}

	conn, err := NewDirectRPCConnection(ctx, nodeUrl, 5)
	require.NoError(t, err)
	require.NotNil(t, conn)

	assert.Equal(t, DirectRPCProtocolHTTPS, conn.GetProtocol())
	assert.True(t, conn.IsHealthy())
	assert.Equal(t, "https://eth-mainnet.g.alchemy.com/v2/test", conn.GetURL())

	err = conn.Close()
	assert.NoError(t, err)
}

func TestWebSocketConnectionCreation(t *testing.T) {
	ctx := context.Background()
	nodeUrl := common.NodeUrl{Url: "wss://eth-mainnet.g.alchemy.com/v2/test"}

	conn, err := NewDirectRPCConnection(ctx, nodeUrl, 5)
	require.NoError(t, err)
	require.NotNil(t, conn)

	assert.Equal(t, DirectRPCProtocolWSS, conn.GetProtocol())
	assert.True(t, conn.IsHealthy())
	assert.Equal(t, "wss://eth-mainnet.g.alchemy.com/v2/test", conn.GetURL())

	err = conn.Close()
	assert.NoError(t, err)
}

func TestGRPCConnectionCreation(t *testing.T) {
	ctx := context.Background()
	nodeUrl := common.NodeUrl{Url: "grpc://localhost:9090"}

	conn, err := NewDirectRPCConnection(ctx, nodeUrl, 5)
	require.NoError(t, err)
	require.NotNil(t, conn)

	assert.Equal(t, DirectRPCProtocolGRPC, conn.GetProtocol())
	assert.True(t, conn.IsHealthy())
	assert.Equal(t, "grpc://localhost:9090", conn.GetURL())

	err = conn.Close()
	assert.NoError(t, err)
}

func TestConnectionCreationWithInvalidURL(t *testing.T) {
	ctx := context.Background()
	nodeUrl := common.NodeUrl{Url: "://invalid"}

	conn, err := NewDirectRPCConnection(ctx, nodeUrl, 5)
	require.Error(t, err)
	assert.Nil(t, conn)
	assert.Contains(t, err.Error(), "failed to detect protocol")
}

func TestConnectionCreationWithUnsupportedProtocol(t *testing.T) {
	ctx := context.Background()
	nodeUrl := common.NodeUrl{Url: "ftp://example.com"}

	conn, err := NewDirectRPCConnection(ctx, nodeUrl, 5)
	require.Error(t, err)
	assert.Nil(t, conn)
	assert.Contains(t, err.Error(), "failed to detect protocol")
}

func TestHTTPConnectionInterface(t *testing.T) {
	ctx := context.Background()
	nodeUrl := common.NodeUrl{Url: "https://test.example.com"}

	conn, err := NewDirectRPCConnection(ctx, nodeUrl, 5)
	require.NoError(t, err)

	// Verify it implements DirectRPCConnection interface
	var _ DirectRPCConnection = conn

	// Test interface methods
	assert.Equal(t, DirectRPCProtocolHTTPS, conn.GetProtocol())
	assert.True(t, conn.IsHealthy())
	assert.Equal(t, "https://test.example.com", conn.GetURL())
	assert.NoError(t, conn.Close())
}

func TestWebSocketSendRequestNotImplemented(t *testing.T) {
	ctx := context.Background()
	nodeUrl := common.NodeUrl{Url: "wss://test.example.com"}

	conn, err := NewDirectRPCConnection(ctx, nodeUrl, 5)
	require.NoError(t, err)

	// WebSocket SendRequest should return not implemented error
	_, err = conn.SendRequest(ctx, []byte("test"), nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "WebSocket SendRequest not implemented")
}

func TestGRPCSendRequestNotImplemented(t *testing.T) {
	ctx := context.Background()
	nodeUrl := common.NodeUrl{Url: "grpc://test.example.com"}

	conn, err := NewDirectRPCConnection(ctx, nodeUrl, 5)
	require.NoError(t, err)

	// gRPC SendRequest should return not implemented error
	_, err = conn.SendRequest(ctx, []byte("test"), nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "gRPC direct connections not yet implemented")
}
