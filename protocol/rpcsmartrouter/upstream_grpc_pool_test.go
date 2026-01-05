package rpcsmartrouter

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/lavanet/lava/v5/protocol/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewUpstreamGRPCPool(t *testing.T) {
	nodeUrl := &common.NodeUrl{
		Url: "grpc://localhost:9090",
	}

	pool := NewUpstreamGRPCPool(nodeUrl)

	require.NotNil(t, pool)
	assert.Equal(t, nodeUrl, pool.nodeUrl)
	// sanitizeEndpointURL extracts just the host:port portion
	assert.Equal(t, "localhost:9090", pool.sanitizedURL)
	assert.Equal(t, 1, pool.minConnections)
	assert.Equal(t, 5, pool.maxConnections)
	assert.Equal(t, 100, pool.streamsPerConn)
	assert.Equal(t, 30*time.Second, pool.connectTimeout)
	assert.NotNil(t, pool.backoff)
	assert.Empty(t, pool.connections)
}

func TestNewUpstreamGRPCPoolWithConfig(t *testing.T) {
	nodeUrl := &common.NodeUrl{
		Url: "grpcs://example.com:443",
	}

	config := &GRPCStreamingConfig{
		PoolMinConnections:   2,
		PoolMaxConnections:   10,
		StreamsPerConnection: 50,
		ConnectionTimeout:    60 * time.Second,
	}

	pool := NewUpstreamGRPCPoolWithConfig(nodeUrl, config)

	require.NotNil(t, pool)
	assert.Equal(t, 2, pool.minConnections)
	assert.Equal(t, 10, pool.maxConnections)
	assert.Equal(t, 50, pool.streamsPerConn)
	assert.Equal(t, 60*time.Second, pool.connectTimeout)
}

func TestNewUpstreamGRPCPoolWithConfig_NilConfig(t *testing.T) {
	nodeUrl := &common.NodeUrl{
		Url: "grpc://localhost:9090",
	}

	pool := NewUpstreamGRPCPoolWithConfig(nodeUrl, nil)

	require.NotNil(t, pool)
	// Should use defaults
	assert.Equal(t, 1, pool.minConnections)
	assert.Equal(t, 5, pool.maxConnections)
}

func TestUpstreamGRPCPool_SetReconnectCallback(t *testing.T) {
	nodeUrl := &common.NodeUrl{
		Url: "grpc://localhost:9090",
	}
	pool := NewUpstreamGRPCPool(nodeUrl)

	callbackCalled := false
	pool.SetReconnectCallback(func() {
		callbackCalled = true
	})

	// Verify callback is stored
	pool.lock.RLock()
	callback := pool.onReconnect
	pool.lock.RUnlock()

	assert.NotNil(t, callback)

	// Call it to verify it works
	callback()
	assert.True(t, callbackCalled)
}

func TestUpstreamGRPCPool_GetEndpoint(t *testing.T) {
	// Test URL sanitization - the sanitizeEndpointURL function extracts host:port only
	tests := []struct {
		name        string
		url         string
		expectedURL string
	}{
		{
			name:        "simple URL",
			url:         "grpc://localhost:9090",
			expectedURL: "localhost:9090", // sanitizeEndpointURL extracts just host:port
		},
		{
			name:        "URL with auth",
			url:         "grpc://user:pass@localhost:9090",
			expectedURL: "localhost:9090", // Auth info is stripped
		},
		{
			name:        "secure gRPC",
			url:         "grpcs://example.com:443",
			expectedURL: "example.com:443", // Same - just host:port
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nodeUrl := &common.NodeUrl{Url: tt.url}
			pool := NewUpstreamGRPCPool(nodeUrl)
			assert.Equal(t, tt.expectedURL, pool.GetEndpoint())
		})
	}
}

func TestUpstreamGRPCPool_ConnectionCount(t *testing.T) {
	nodeUrl := &common.NodeUrl{
		Url: "grpc://localhost:9090",
	}
	pool := NewUpstreamGRPCPool(nodeUrl)

	// Initially should be 0
	assert.Equal(t, 0, pool.ConnectionCount())
}

func TestUpstreamGRPCPool_TotalStreamCount(t *testing.T) {
	nodeUrl := &common.NodeUrl{
		Url: "grpc://localhost:9090",
	}
	pool := NewUpstreamGRPCPool(nodeUrl)

	// Initially should be 0
	assert.Equal(t, int32(0), pool.TotalStreamCount())
}

func TestUpstreamGRPCPool_Close_Empty(t *testing.T) {
	nodeUrl := &common.NodeUrl{
		Url: "grpc://localhost:9090",
	}
	pool := NewUpstreamGRPCPool(nodeUrl)

	// Close empty pool should not panic
	err := pool.Close()
	assert.NoError(t, err)

	// Pool should be marked as closed
	assert.True(t, pool.closed.Load())
}

func TestUpstreamGRPCPool_Close_Idempotent(t *testing.T) {
	nodeUrl := &common.NodeUrl{
		Url: "grpc://localhost:9090",
	}
	pool := NewUpstreamGRPCPool(nodeUrl)

	// First close
	err1 := pool.Close()
	assert.NoError(t, err1)

	// Second close should also be fine
	err2 := pool.Close()
	assert.NoError(t, err2)
}

func TestUpstreamGRPCPool_GetConnectionForStream_ClosedPool(t *testing.T) {
	nodeUrl := &common.NodeUrl{
		Url: "grpc://localhost:9090",
	}
	pool := NewUpstreamGRPCPool(nodeUrl)

	pool.Close()

	ctx := context.Background()
	conn, err := pool.GetConnectionForStream(ctx)

	assert.Error(t, err)
	assert.Nil(t, conn)
	assert.Contains(t, err.Error(), "pool is closed")
}

func TestUpstreamGRPCStreamConnection_StreamCount(t *testing.T) {
	// Create a mock connection (without actual gRPC)
	conn := &UpstreamGRPCStreamConnection{
		endpoint:     "grpc://localhost:9090",
		sanitizedURL: "grpc://localhost:9090",
		createdAt:    time.Now(),
	}
	conn.healthy.Store(true)

	// Initial count should be 0
	assert.Equal(t, int32(0), conn.StreamCount())

	// Increment
	count := conn.IncrementStreams()
	assert.Equal(t, int32(1), count)
	assert.Equal(t, int32(1), conn.StreamCount())

	// Increment again
	count = conn.IncrementStreams()
	assert.Equal(t, int32(2), count)

	// Decrement
	count = conn.DecrementStreams()
	assert.Equal(t, int32(1), count)

	// Decrement again
	count = conn.DecrementStreams()
	assert.Equal(t, int32(0), count)
}

func TestUpstreamGRPCStreamConnection_IsHealthy(t *testing.T) {
	conn := &UpstreamGRPCStreamConnection{
		endpoint:     "grpc://localhost:9090",
		sanitizedURL: "grpc://localhost:9090",
		createdAt:    time.Now(),
	}

	// Initially not healthy (default is false)
	assert.False(t, conn.IsHealthy())

	// Mark as healthy
	conn.healthy.Store(true)
	assert.True(t, conn.IsHealthy())

	// Mark unhealthy
	conn.MarkUnhealthy(nil)
	assert.False(t, conn.IsHealthy())
}

func TestUpstreamGRPCStreamConnection_IsHealthy_WhenClosed(t *testing.T) {
	conn := &UpstreamGRPCStreamConnection{
		endpoint:     "grpc://localhost:9090",
		sanitizedURL: "grpc://localhost:9090",
		createdAt:    time.Now(),
	}
	conn.healthy.Store(true)
	assert.True(t, conn.IsHealthy())

	// Close the connection
	conn.closed.Store(true)

	// Should be unhealthy when closed even if healthy flag is true
	assert.False(t, conn.IsHealthy())
}

func TestUpstreamGRPCStreamConnection_MarkUnhealthy_WithError(t *testing.T) {
	conn := &UpstreamGRPCStreamConnection{
		endpoint:     "grpc://localhost:9090",
		sanitizedURL: "grpc://localhost:9090",
		createdAt:    time.Now(),
	}
	conn.healthy.Store(true)

	testErr := context.DeadlineExceeded
	conn.MarkUnhealthy(testErr)

	assert.False(t, conn.IsHealthy())

	// Check error was stored
	storedErr := conn.lastError.Load()
	assert.Equal(t, testErr, storedErr)
}

func TestUpstreamGRPCStreamConnection_GetEndpoint(t *testing.T) {
	conn := &UpstreamGRPCStreamConnection{
		endpoint:     "grpc://user:pass@localhost:9090",
		sanitizedURL: "grpc://[REDACTED]@localhost:9090",
	}

	// GetEndpoint should return sanitized URL
	assert.Equal(t, "grpc://[REDACTED]@localhost:9090", conn.GetEndpoint())
}

func TestUpstreamGRPCStreamConnection_GetNodeUrl(t *testing.T) {
	nodeUrl := &common.NodeUrl{
		Url: "grpc://localhost:9090",
	}
	conn := &UpstreamGRPCStreamConnection{
		nodeUrl: nodeUrl,
	}

	assert.Equal(t, nodeUrl, conn.GetNodeUrl())
}

func TestUpstreamGRPCStreamConnection_Close_Idempotent(t *testing.T) {
	conn := &UpstreamGRPCStreamConnection{
		endpoint:     "grpc://localhost:9090",
		sanitizedURL: "grpc://localhost:9090",
		createdAt:    time.Now(),
	}

	// First close
	err1 := conn.Close()
	assert.NoError(t, err1)
	assert.True(t, conn.closed.Load())

	// Second close should be no-op
	err2 := conn.Close()
	assert.NoError(t, err2)
}

func TestUpstreamGRPCStreamConnection_ConcurrentStreamOperations(t *testing.T) {
	conn := &UpstreamGRPCStreamConnection{
		endpoint:     "grpc://localhost:9090",
		sanitizedURL: "grpc://localhost:9090",
		createdAt:    time.Now(),
	}
	conn.healthy.Store(true)

	var wg sync.WaitGroup
	numGoroutines := 100
	numOperations := 100

	// Concurrently increment and decrement streams
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				conn.IncrementStreams()
				conn.StreamCount()
				conn.DecrementStreams()
			}
		}()
	}

	wg.Wait()

	// Final count should be 0
	assert.Equal(t, int32(0), conn.StreamCount())
}

func TestUpstreamGRPCPool_NotifyStreamRemoved_NilConn(t *testing.T) {
	nodeUrl := &common.NodeUrl{
		Url: "grpc://localhost:9090",
	}
	pool := NewUpstreamGRPCPool(nodeUrl)

	// Should not panic with nil connection
	pool.NotifyStreamRemoved(nil)
}

func TestUpstreamGRPCPool_MaybeScaleDown_BelowMinConnections(t *testing.T) {
	nodeUrl := &common.NodeUrl{
		Url: "grpc://localhost:9090",
	}
	pool := NewUpstreamGRPCPool(nodeUrl)
	pool.minConnections = 2

	// Add one mock connection
	mockConn := &UpstreamGRPCStreamConnection{
		endpoint:     "grpc://localhost:9090",
		sanitizedURL: "grpc://localhost:9090",
		createdAt:    time.Now(),
	}
	mockConn.healthy.Store(true)

	pool.lock.Lock()
	pool.connections = []*UpstreamGRPCStreamConnection{mockConn}
	pool.lock.Unlock()

	// Scale down should not remove connections below minConnections
	pool.maybeScaleDown()

	assert.Equal(t, 1, pool.ConnectionCount())
}

func TestUpstreamGRPCPool_ReconnectWithBackoff_AlreadyReconnecting(t *testing.T) {
	nodeUrl := &common.NodeUrl{
		Url: "grpc://localhost:9090",
	}
	pool := NewUpstreamGRPCPool(nodeUrl)

	// Mark as already reconnecting
	pool.reconnecting.Store(true)

	ctx := context.Background()
	err := pool.ReconnectWithBackoff(ctx)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "reconnection already in progress")
}

func TestUpstreamGRPCPool_ReconnectWithBackoff_ContextCanceled(t *testing.T) {
	nodeUrl := &common.NodeUrl{
		Url: "grpc://localhost:9090",
	}
	pool := NewUpstreamGRPCPool(nodeUrl)

	// Cancel context immediately
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := pool.ReconnectWithBackoff(ctx)

	assert.Error(t, err)
	assert.Equal(t, context.Canceled, err)

	// Reconnecting flag should be cleared
	assert.False(t, pool.reconnecting.Load())
}
