package rpcsmartrouter

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/grpcreflect"
	"github.com/lavanet/lava/v5/protocol/chainlib/chainproxy/rpcInterfaceMessages"
	"github.com/lavanet/lava/v5/protocol/common"
	"github.com/lavanet/lava/v5/utils"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	reflectionpb "google.golang.org/grpc/reflection/grpc_reflection_v1alpha"
)

// UpstreamGRPCStreamConnection wraps a grpc.ClientConn with health tracking and stream count.
// Unlike the unary GRPCConnector, this is designed for long-lived streaming connections.
type UpstreamGRPCStreamConnection struct {
	conn             *grpc.ClientConn
	endpoint         string
	sanitizedURL     string // For logging (no auth info)
	nodeUrl          *common.NodeUrl
	healthy          atomic.Bool
	lastError        atomic.Value // stores error
	createdAt        time.Time
	activeStreams    atomic.Int32                                  // Number of active streams on this connection
	descriptorsCache *common.SafeSyncMap[string, *desc.MethodDescriptor] // Cached method descriptors
	lock             sync.RWMutex
	closed           atomic.Bool
}

// NewUpstreamGRPCStreamConnection creates a new upstream gRPC connection for streaming
func NewUpstreamGRPCStreamConnection(ctx context.Context, nodeUrl *common.NodeUrl, timeout time.Duration) (*UpstreamGRPCStreamConnection, error) {
	conn := &UpstreamGRPCStreamConnection{
		endpoint:         nodeUrl.Url,
		sanitizedURL:     sanitizeEndpointURL(nodeUrl.Url),
		nodeUrl:          nodeUrl,
		createdAt:        time.Now(),
		descriptorsCache: &common.SafeSyncMap[string, *desc.MethodDescriptor]{},
	}
	conn.healthy.Store(true)

	if err := conn.connect(ctx, timeout); err != nil {
		return nil, err
	}

	return conn, nil
}

// connect establishes the gRPC connection
func (c *UpstreamGRPCStreamConnection) connect(ctx context.Context, timeout time.Duration) error {
	c.lock.Lock()
	defer c.lock.Unlock()

	if c.closed.Load() {
		return fmt.Errorf("connection is closed")
	}

	// Parse URL to extract host
	parsedURL, err := url.Parse(c.endpoint)
	if err != nil {
		return fmt.Errorf("failed to parse gRPC URL %s: %w", c.sanitizedURL, err)
	}

	// Determine target address (host:port)
	target := parsedURL.Host
	if parsedURL.Path != "" && parsedURL.Path != "/" {
		target += parsedURL.Path
	}

	// Determine TLS configuration
	var dialOpts []grpc.DialOption
	scheme := strings.ToLower(parsedURL.Scheme)

	if scheme == "grpcs" || c.nodeUrl.AuthConfig.UseTLS {
		// Use TLS
		tlsConfig := &tls.Config{}
		if c.nodeUrl.AuthConfig.AllowInsecure {
			tlsConfig.InsecureSkipVerify = true
		}
		dialOpts = append(dialOpts, grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)))
	} else {
		// Insecure connection (for local/dev)
		if !c.nodeUrl.GrpcConfig.AllowInsecure {
			return fmt.Errorf("insecure gRPC (grpc://) requires allow-insecure: true in config")
		}
		dialOpts = append(dialOpts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	// Add standard options
	dialOpts = append(dialOpts,
		grpc.WithBlock(),
		grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(512*1024*1024)), // 512MB for large responses
	)

	// Create connection with timeout
	connectCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	grpcConn, err := grpc.DialContext(connectCtx, target, dialOpts...)
	if err != nil {
		c.healthy.Store(false)
		c.lastError.Store(err)
		return fmt.Errorf("failed to dial gRPC %s: %w", c.sanitizedURL, err)
	}

	c.conn = grpcConn
	c.healthy.Store(true)

	utils.LavaFormatDebug("gRPC streaming connection established",
		utils.LogAttr("endpoint", c.sanitizedURL),
	)

	return nil
}

// GetConn returns the underlying grpc.ClientConn for stream operations
func (c *UpstreamGRPCStreamConnection) GetConn() *grpc.ClientConn {
	c.lock.RLock()
	defer c.lock.RUnlock()
	return c.conn
}

// IsHealthy returns true if the connection is healthy
func (c *UpstreamGRPCStreamConnection) IsHealthy() bool {
	return c.healthy.Load() && !c.closed.Load()
}

// MarkUnhealthy marks the connection as unhealthy
func (c *UpstreamGRPCStreamConnection) MarkUnhealthy(err error) {
	c.healthy.Store(false)
	if err != nil {
		c.lastError.Store(err)
	}
}

// Close closes the gRPC connection
func (c *UpstreamGRPCStreamConnection) Close() error {
	if !c.closed.CompareAndSwap(false, true) {
		return nil // Already closed
	}

	c.lock.Lock()
	defer c.lock.Unlock()

	if c.conn != nil {
		err := c.conn.Close()
		c.conn = nil
		utils.LavaFormatDebug("gRPC streaming connection closed",
			utils.LogAttr("endpoint", c.sanitizedURL),
		)
		return err
	}

	return nil
}

// GetEndpoint returns the endpoint URL (sanitized for logging)
func (c *UpstreamGRPCStreamConnection) GetEndpoint() string {
	return c.sanitizedURL
}

// GetNodeUrl returns the full nodeUrl configuration
func (c *UpstreamGRPCStreamConnection) GetNodeUrl() *common.NodeUrl {
	return c.nodeUrl
}

// IncrementStreams increments the active stream count and returns the new count
func (c *UpstreamGRPCStreamConnection) IncrementStreams() int32 {
	return c.activeStreams.Add(1)
}

// DecrementStreams decrements the active stream count and returns the new count
func (c *UpstreamGRPCStreamConnection) DecrementStreams() int32 {
	return c.activeStreams.Add(-1)
}

// StreamCount returns the current number of active streams on this connection
func (c *UpstreamGRPCStreamConnection) StreamCount() int32 {
	return c.activeStreams.Load()
}

// GetMethodDescriptor retrieves a method descriptor, using cache or reflection
func (c *UpstreamGRPCStreamConnection) GetMethodDescriptor(
	ctx context.Context,
	service, methodName string,
) (*desc.MethodDescriptor, error) {
	fullMethodName := service + "." + methodName

	// Check cache first
	if methodDesc, found, _ := c.descriptorsCache.Load(fullMethodName); found {
		return methodDesc, nil
	}

	c.lock.RLock()
	conn := c.conn
	c.lock.RUnlock()

	if conn == nil {
		return nil, fmt.Errorf("connection not available")
	}

	// Use reflection to get descriptor
	cl := grpcreflect.NewClient(ctx, reflectionpb.NewServerReflectionClient(conn))
	defer cl.Reset()

	descriptorSource := rpcInterfaceMessages.DescriptorSourceFromServer(cl)

	descriptor, err := descriptorSource.FindSymbol(service)
	if err != nil {
		return nil, utils.LavaFormatError("failed to find service via reflection", err,
			utils.LogAttr("service", service))
	}

	serviceDescriptor, ok := descriptor.(*desc.ServiceDescriptor)
	if !ok {
		return nil, utils.LavaFormatError("descriptor is not a ServiceDescriptor", nil,
			utils.LogAttr("service", service))
	}

	methodDescriptor := serviceDescriptor.FindMethodByName(methodName)
	if methodDescriptor == nil {
		return nil, utils.LavaFormatError("method not found in service", nil,
			utils.LogAttr("service", service),
			utils.LogAttr("method", methodName))
	}

	// Cache the descriptor
	c.descriptorsCache.Store(fullMethodName, methodDescriptor)

	return methodDescriptor, nil
}

// UpstreamGRPCPool manages a pool of gRPC connections for streaming.
// The pool automatically scales between minConnections and maxConnections based on
// stream load, targeting approximately streamsPerConn streams per connection.
type UpstreamGRPCPool struct {
	nodeUrl       *common.NodeUrl
	sanitizedURL  string
	connections   []*UpstreamGRPCStreamConnection
	backoff       *ExponentialBackoff
	reconnectMu   sync.Mutex // Prevents concurrent reconnection attempts
	lock          sync.RWMutex
	closed        atomic.Bool
	reconnecting  atomic.Bool
	onReconnect   func() // Callback when reconnected (for stream restoration)

	// Pool configuration
	minConnections   int           // Minimum connections to maintain (default: 1)
	maxConnections   int           // Maximum connections allowed (default: 5)
	streamsPerConn   int           // Target streams per connection (default: 100)
	connectTimeout   time.Duration // Connection establishment timeout
}

// NewUpstreamGRPCPool creates a new gRPC connection pool for streaming
func NewUpstreamGRPCPool(nodeUrl *common.NodeUrl) *UpstreamGRPCPool {
	return &UpstreamGRPCPool{
		nodeUrl:          nodeUrl,
		sanitizedURL:     sanitizeEndpointURL(nodeUrl.Url),
		connections:      make([]*UpstreamGRPCStreamConnection, 0),
		backoff:          NewWebSocketBackoff(), // Reuse the same backoff logic
		minConnections:   1,
		maxConnections:   5,
		streamsPerConn:   100,
		connectTimeout:   30 * time.Second,
	}
}

// NewUpstreamGRPCPoolWithConfig creates a pool with custom configuration
func NewUpstreamGRPCPoolWithConfig(nodeUrl *common.NodeUrl, config *GRPCStreamingConfig) *UpstreamGRPCPool {
	pool := NewUpstreamGRPCPool(nodeUrl)
	if config != nil {
		pool.minConnections = config.PoolMinConnections
		pool.maxConnections = config.PoolMaxConnections
		pool.streamsPerConn = config.StreamsPerConnection
		pool.connectTimeout = config.ConnectionTimeout
	}
	return pool
}

// SetReconnectCallback sets a callback to be called after successful reconnection
// This is used to restore streams after reconnection
func (p *UpstreamGRPCPool) SetReconnectCallback(callback func()) {
	p.lock.Lock()
	defer p.lock.Unlock()
	p.onReconnect = callback
}

// GetConnectionForStream returns the best connection for a new stream.
// It prefers connections with lower stream counts and will create new connections
// if all existing ones are near capacity (and we haven't hit maxConnections).
func (p *UpstreamGRPCPool) GetConnectionForStream(ctx context.Context) (*UpstreamGRPCStreamConnection, error) {
	if p.closed.Load() {
		return nil, fmt.Errorf("pool is closed")
	}

	p.lock.Lock()
	defer p.lock.Unlock()

	// Find the best connection (healthy with lowest stream count)
	var bestConn *UpstreamGRPCStreamConnection
	var lowestStreams int32 = int32(p.streamsPerConn + 1)

	for _, conn := range p.connections {
		if conn.IsHealthy() {
			streams := conn.StreamCount()
			if streams < lowestStreams {
				lowestStreams = streams
				bestConn = conn
			}
		}
	}

	// If we have a connection with capacity, use it
	if bestConn != nil && lowestStreams < int32(p.streamsPerConn) {
		return bestConn, nil
	}

	// Need to scale up if possible
	if len(p.connections) < p.maxConnections {
		newConn, err := p.createConnectionLocked(ctx)
		if err != nil {
			// If we can't create a new connection but have an existing one, use it
			if bestConn != nil {
				utils.LavaFormatWarning("gRPC pool: failed to scale up, using existing connection", err,
					utils.LogAttr("endpoint", p.sanitizedURL),
					utils.LogAttr("currentConnections", len(p.connections)),
					utils.LogAttr("existingStreamCount", lowestStreams),
				)
				return bestConn, nil
			}
			return nil, fmt.Errorf("failed to create connection: %w", err)
		}

		utils.LavaFormatInfo("gRPC pool: scaled up",
			utils.LogAttr("endpoint", p.sanitizedURL),
			utils.LogAttr("totalConnections", len(p.connections)),
			utils.LogAttr("reason", "all connections near capacity"),
		)
		return newConn, nil
	}

	// At max connections - use the one with lowest streams even if over target
	if bestConn != nil {
		return bestConn, nil
	}

	// No healthy connections - try to create one
	return p.createConnectionLocked(ctx)
}

// createConnectionLocked creates a new connection (caller must hold lock)
func (p *UpstreamGRPCPool) createConnectionLocked(ctx context.Context) (*UpstreamGRPCStreamConnection, error) {
	conn, err := NewUpstreamGRPCStreamConnection(ctx, p.nodeUrl, p.connectTimeout)
	if err != nil {
		return nil, err
	}

	p.connections = append(p.connections, conn)
	p.backoff.Reset()

	utils.LavaFormatDebug("gRPC pool: connection added",
		utils.LogAttr("endpoint", p.sanitizedURL),
		utils.LogAttr("totalConnections", len(p.connections)),
	)

	return conn, nil
}

// NotifyStreamRemoved should be called when a stream is closed on a connection.
// This allows the pool to potentially scale down if connections are underutilized.
func (p *UpstreamGRPCPool) NotifyStreamRemoved(conn *UpstreamGRPCStreamConnection) {
	if conn == nil {
		return
	}

	conn.DecrementStreams()

	// Consider scaling down if we have excess connections with no streams
	p.maybeScaleDown()
}

// maybeScaleDown removes empty connections if we have more than minConnections
func (p *UpstreamGRPCPool) maybeScaleDown() {
	p.lock.Lock()
	defer p.lock.Unlock()

	if len(p.connections) <= p.minConnections {
		return
	}

	// Find connections with no streams (excluding the first minConnections)
	var toRemove []*UpstreamGRPCStreamConnection
	kept := make([]*UpstreamGRPCStreamConnection, 0, len(p.connections))

	for i, conn := range p.connections {
		if i < p.minConnections {
			// Keep minimum connections
			kept = append(kept, conn)
		} else if conn.StreamCount() == 0 && !conn.IsHealthy() {
			// Remove unhealthy empty connections
			toRemove = append(toRemove, conn)
		} else if conn.StreamCount() == 0 {
			// Mark for potential removal (only if we have excess)
			if len(kept)+len(p.connections)-i-1 >= p.minConnections {
				toRemove = append(toRemove, conn)
			} else {
				kept = append(kept, conn)
			}
		} else {
			kept = append(kept, conn)
		}
	}

	if len(toRemove) > 0 {
		p.connections = kept

		// Close removed connections asynchronously
		go func(conns []*UpstreamGRPCStreamConnection) {
			for _, conn := range conns {
				conn.Close()
			}
			utils.LavaFormatDebug("gRPC pool: scaled down",
				utils.LogAttr("endpoint", p.sanitizedURL),
				utils.LogAttr("removedConnections", len(conns)),
				utils.LogAttr("remainingConnections", len(kept)),
			)
		}(toRemove)
	}
}

// ReconnectWithBackoff attempts to reconnect the pool with exponential backoff
func (p *UpstreamGRPCPool) ReconnectWithBackoff(ctx context.Context) error {
	if !p.reconnecting.CompareAndSwap(false, true) {
		return fmt.Errorf("reconnection already in progress")
	}
	defer p.reconnecting.Store(false)

	p.reconnectMu.Lock()
	defer p.reconnectMu.Unlock()

	// Get backoff duration
	backoffDuration, _ := p.backoff.NextBackoff()
	utils.LavaFormatDebug("gRPC pool: waiting before reconnect",
		utils.LogAttr("endpoint", p.sanitizedURL),
		utils.LogAttr("backoff", backoffDuration),
	)

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(backoffDuration):
	}

	// Try to create a new connection
	p.lock.Lock()
	newConn, err := NewUpstreamGRPCStreamConnection(ctx, p.nodeUrl, p.connectTimeout)
	if err != nil {
		p.lock.Unlock()
		return fmt.Errorf("failed to reconnect: %w", err)
	}

	p.connections = append(p.connections, newConn)
	p.backoff.Reset()
	callback := p.onReconnect
	p.lock.Unlock()

	utils.LavaFormatInfo("gRPC pool: reconnected successfully",
		utils.LogAttr("endpoint", p.sanitizedURL),
	)

	// Call reconnect callback if set (for stream restoration)
	if callback != nil {
		callback()
	}

	return nil
}

// Close closes all connections in the pool
func (p *UpstreamGRPCPool) Close() error {
	if !p.closed.CompareAndSwap(false, true) {
		return nil
	}

	p.lock.Lock()
	conns := p.connections
	p.connections = nil
	p.lock.Unlock()

	for _, conn := range conns {
		conn.Close()
	}

	utils.LavaFormatDebug("gRPC pool: closed",
		utils.LogAttr("endpoint", p.sanitizedURL),
		utils.LogAttr("closedConnections", len(conns)),
	)

	return nil
}

// GetEndpoint returns the sanitized endpoint URL
func (p *UpstreamGRPCPool) GetEndpoint() string {
	return p.sanitizedURL
}

// ConnectionCount returns the current number of connections in the pool
func (p *UpstreamGRPCPool) ConnectionCount() int {
	p.lock.RLock()
	defer p.lock.RUnlock()
	return len(p.connections)
}

// TotalStreamCount returns the total number of active streams across all connections
func (p *UpstreamGRPCPool) TotalStreamCount() int32 {
	p.lock.RLock()
	defer p.lock.RUnlock()

	var total int32
	for _, conn := range p.connections {
		total += conn.StreamCount()
	}
	return total
}
