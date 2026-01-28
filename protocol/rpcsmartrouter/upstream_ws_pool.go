package rpcsmartrouter

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	rpcclient "github.com/lavanet/lava/v5/protocol/chainlib/chainproxy/rpcclient"
	"github.com/lavanet/lava/v5/protocol/common"
	"github.com/lavanet/lava/v5/utils"
)

// UpstreamWSConnection wraps an rpcclient.Client with health tracking and subscription count
type UpstreamWSConnection struct {
	client           *rpcclient.Client
	endpoint         string
	sanitizedURL     string // For logging (no auth)
	nodeUrl          *common.NodeUrl
	healthy          atomic.Bool
	lastError        atomic.Value // stores error
	createdAt        time.Time
	subscriptionCount atomic.Int32 // Number of active subscriptions on this connection
	lock             sync.RWMutex
	closed           atomic.Bool
}

// NewUpstreamWSConnection creates a new upstream WebSocket connection
func NewUpstreamWSConnection(ctx context.Context, nodeUrl *common.NodeUrl) (*UpstreamWSConnection, error) {
	conn := &UpstreamWSConnection{
		endpoint:     nodeUrl.Url,
		sanitizedURL: sanitizeEndpointURL(nodeUrl.Url),
		nodeUrl:      nodeUrl,
		createdAt:    time.Now(),
	}
	conn.healthy.Store(true)

	if err := conn.connect(ctx); err != nil {
		return nil, err
	}

	return conn, nil
}

// connect establishes the WebSocket connection using rpcclient
func (c *UpstreamWSConnection) connect(ctx context.Context) error {
	c.lock.Lock()
	defer c.lock.Unlock()

	if c.closed.Load() {
		return fmt.Errorf("connection is closed")
	}

	// Build headers from nodeUrl auth config
	headers := make(map[string]string)
	for k, v := range c.nodeUrl.GetAuthHeaders() {
		headers[k] = v
	}

	// Add auth path if configured
	endpoint := c.nodeUrl.AuthConfig.AddAuthPath(c.endpoint)

	// Use rpcclient's WebSocket dialer
	client, err := rpcclient.DialWebsocket(ctx, endpoint, headers)
	if err != nil {
		c.healthy.Store(false)
		c.lastError.Store(err)
		return fmt.Errorf("failed to dial WebSocket %s: %w", c.sanitizedURL, err)
	}

	c.client = client
	c.healthy.Store(true)
	// Note: Don't store nil in lastError - atomic.Value panics on nil
	// The healthy flag indicates success; lastError is only set on failure

	utils.LavaFormatDebug("WebSocket connection established",
		utils.LogAttr("endpoint", c.sanitizedURL),
	)

	return nil
}

// GetClient returns the underlying rpcclient.Client for subscription operations
func (c *UpstreamWSConnection) GetClient() *rpcclient.Client {
	c.lock.RLock()
	defer c.lock.RUnlock()
	return c.client
}

// IsHealthy returns true if the connection is healthy
func (c *UpstreamWSConnection) IsHealthy() bool {
	return c.healthy.Load() && !c.closed.Load()
}

// MarkUnhealthy marks the connection as unhealthy
func (c *UpstreamWSConnection) MarkUnhealthy(err error) {
	c.healthy.Store(false)
	if err != nil {
		c.lastError.Store(err)
	}
}

// Close closes the WebSocket connection
func (c *UpstreamWSConnection) Close() error {
	if !c.closed.CompareAndSwap(false, true) {
		return nil // Already closed
	}

	c.lock.Lock()
	defer c.lock.Unlock()

	if c.client != nil {
		c.client.Close()
		c.client = nil
	}

	utils.LavaFormatDebug("WebSocket connection closed",
		utils.LogAttr("endpoint", c.sanitizedURL),
	)

	return nil
}

// GetEndpoint returns the endpoint URL (sanitized for logging)
func (c *UpstreamWSConnection) GetEndpoint() string {
	return c.sanitizedURL
}

// GetNodeUrl returns the full nodeUrl configuration
func (c *UpstreamWSConnection) GetNodeUrl() *common.NodeUrl {
	return c.nodeUrl
}

// IncrementSubscriptions increments the subscription count and returns the new count
func (c *UpstreamWSConnection) IncrementSubscriptions() int32 {
	return c.subscriptionCount.Add(1)
}

// DecrementSubscriptions decrements the subscription count and returns the new count
func (c *UpstreamWSConnection) DecrementSubscriptions() int32 {
	return c.subscriptionCount.Add(-1)
}

// SubscriptionCount returns the current number of subscriptions on this connection
func (c *UpstreamWSConnection) SubscriptionCount() int32 {
	return c.subscriptionCount.Load()
}

// UpstreamWSPool manages a pool of WebSocket connections to upstream endpoints.
// The pool automatically scales between minConnections and maxConnections based on
// subscription load, targeting approximately subscriptionsPerConn subscriptions per connection.
type UpstreamWSPool struct {
	nodeUrl          *common.NodeUrl
	sanitizedURL     string
	connections      []*UpstreamWSConnection // Slice of connections for auto-scaling
	backoff          *ExponentialBackoff
	reconnectMu      sync.Mutex // Prevents concurrent reconnection attempts
	lock             sync.RWMutex
	closed           atomic.Bool
	reconnecting     atomic.Bool
	onReconnect      func() // Callback when reconnected (for subscription restoration)

	// Pool configuration (from WebsocketConfig)
	minConnections       int // Minimum connections to maintain (default: 1)
	maxConnections       int // Maximum connections allowed (default: 10)
	subscriptionsPerConn int // Target subscriptions per connection (default: 100)
}

// NewUpstreamWSPool creates a new WebSocket connection pool for an endpoint
func NewUpstreamWSPool(nodeUrl *common.NodeUrl) *UpstreamWSPool {
	return &UpstreamWSPool{
		nodeUrl:              nodeUrl,
		sanitizedURL:         sanitizeEndpointURL(nodeUrl.Url),
		connections:          make([]*UpstreamWSConnection, 0),
		backoff:              NewWebSocketBackoff(),
		minConnections:       1,   // Default minimum
		maxConnections:       10,  // Default maximum
		subscriptionsPerConn: 100, // Default target
	}
}

// NewUpstreamWSPoolWithConfig creates a pool with custom configuration
func NewUpstreamWSPoolWithConfig(nodeUrl *common.NodeUrl, config *WebsocketConfig) *UpstreamWSPool {
	pool := NewUpstreamWSPool(nodeUrl)
	if config != nil {
		pool.minConnections = config.UpstreamPoolMinConnections
		pool.maxConnections = config.UpstreamPoolMaxConnections
		pool.subscriptionsPerConn = config.UpstreamPoolSubscriptionsPerConn
	}
	return pool
}

// SetReconnectCallback sets a callback to be called after successful reconnection
// This is used to restore subscriptions after reconnection
func (p *UpstreamWSPool) SetReconnectCallback(callback func()) {
	p.lock.Lock()
	defer p.lock.Unlock()
	p.onReconnect = callback
}

// GetConnection returns a healthy connection, creating one if needed.
// This is the legacy API - prefer GetConnectionForSubscription for new code.
func (p *UpstreamWSPool) GetConnection(ctx context.Context) (*UpstreamWSConnection, error) {
	return p.GetConnectionForSubscription(ctx)
}

// GetConnectionForSubscription returns the best connection for a new subscription.
// It prefers connections with lower subscription counts and will create new connections
// if all existing ones are near capacity (and we haven't hit maxConnections).
func (p *UpstreamWSPool) GetConnectionForSubscription(ctx context.Context) (*UpstreamWSConnection, error) {
	if p.closed.Load() {
		return nil, fmt.Errorf("pool is closed")
	}

	p.lock.Lock()
	defer p.lock.Unlock()

	// Find the best connection (healthy with lowest subscription count)
	var bestConn *UpstreamWSConnection
	var lowestSubs int32 = int32(p.subscriptionsPerConn + 1) // Start higher than target

	for _, conn := range p.connections {
		if conn.IsHealthy() {
			subs := conn.SubscriptionCount()
			if subs < lowestSubs {
				lowestSubs = subs
				bestConn = conn
			}
		}
	}

	// If we have a connection with capacity, use it
	if bestConn != nil && lowestSubs < int32(p.subscriptionsPerConn) {
		return bestConn, nil
	}

	// Need to scale up if possible
	if len(p.connections) < p.maxConnections {
		newConn, err := p.createConnectionLocked(ctx)
		if err != nil {
			// If we can't create a new connection but have an existing one, use it
			if bestConn != nil {
				utils.LavaFormatWarning("WebSocket pool: failed to scale up, using existing connection", err,
					utils.LogAttr("endpoint", p.sanitizedURL),
					utils.LogAttr("currentConnections", len(p.connections)),
					utils.LogAttr("existingSubCount", lowestSubs),
				)
				return bestConn, nil
			}
			return nil, fmt.Errorf("failed to create connection: %w", err)
		}

		utils.LavaFormatInfo("WebSocket pool: scaled up",
			utils.LogAttr("endpoint", p.sanitizedURL),
			utils.LogAttr("totalConnections", len(p.connections)),
			utils.LogAttr("reason", "all connections near capacity"),
		)
		return newConn, nil
	}

	// At max connections - use the one with lowest subs even if over target
	if bestConn != nil {
		return bestConn, nil
	}

	// No healthy connections - try to create one
	return p.createConnectionLocked(ctx)
}

// createConnectionLocked creates a new connection (caller must hold lock)
func (p *UpstreamWSPool) createConnectionLocked(ctx context.Context) (*UpstreamWSConnection, error) {
	conn, err := NewUpstreamWSConnection(ctx, p.nodeUrl)
	if err != nil {
		return nil, err
	}

	p.connections = append(p.connections, conn)
	p.backoff.Reset()

	utils.LavaFormatDebug("WebSocket pool: connection added",
		utils.LogAttr("endpoint", p.sanitizedURL),
		utils.LogAttr("totalConnections", len(p.connections)),
	)

	return conn, nil
}

// NotifySubscriptionRemoved should be called when a subscription is removed from a connection.
// This allows the pool to potentially scale down if connections are underutilized.
func (p *UpstreamWSPool) NotifySubscriptionRemoved(conn *UpstreamWSConnection) {
	if conn == nil {
		return
	}

	conn.DecrementSubscriptions()

	// Consider scaling down if we have excess connections with no subscriptions
	p.maybeScaleDown()
}

// maybeScaleDown removes empty connections if we have more than minConnections
func (p *UpstreamWSPool) maybeScaleDown() {
	p.lock.Lock()
	defer p.lock.Unlock()

	if len(p.connections) <= p.minConnections {
		return
	}

	// Find connections with no subscriptions (excluding the first minConnections)
	var toRemove []*UpstreamWSConnection
	kept := make([]*UpstreamWSConnection, 0, len(p.connections))

	for i, conn := range p.connections {
		if i < p.minConnections {
			// Keep minimum connections
			kept = append(kept, conn)
		} else if conn.SubscriptionCount() == 0 && !conn.IsHealthy() {
			// Remove unhealthy empty connections
			toRemove = append(toRemove, conn)
		} else if conn.SubscriptionCount() == 0 {
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
		go func(conns []*UpstreamWSConnection) {
			for _, conn := range conns {
				conn.Close()
			}
			utils.LavaFormatDebug("WebSocket pool: scaled down",
				utils.LogAttr("endpoint", p.sanitizedURL),
				utils.LogAttr("removedConnections", len(conns)),
				utils.LogAttr("remainingConnections", len(kept)),
			)
		}(toRemove)
	}
}

// ReconnectWithBackoff attempts to reconnect all unhealthy connections using exponential backoff.
// This should be called when a connection error is detected.
// Returns nil on success, or an error if max retries exceeded or context cancelled.
func (p *UpstreamWSPool) ReconnectWithBackoff(ctx context.Context) error {
	// Prevent concurrent reconnection attempts
	if !p.reconnecting.CompareAndSwap(false, true) {
		// Another goroutine is already reconnecting, wait for it
		return p.waitForReconnect(ctx)
	}
	defer p.reconnecting.Store(false)

	p.reconnectMu.Lock()
	defer p.reconnectMu.Unlock()

	// Check if we have any healthy connections
	p.lock.RLock()
	hasHealthy := false
	for _, conn := range p.connections {
		if conn.IsHealthy() {
			hasHealthy = true
			break
		}
	}
	p.lock.RUnlock()

	if hasHealthy {
		return nil // At least one connection is healthy
	}

	backoff := p.backoff.Clone() // Use fresh backoff for this reconnection attempt

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Attempt to connect
		conn, err := NewUpstreamWSConnection(ctx, p.nodeUrl)
		if err == nil {
			p.lock.Lock()
			// Close all unhealthy connections
			for _, oldConn := range p.connections {
				if !oldConn.IsHealthy() {
					oldConn.Close()
				}
			}
			// Keep healthy connections and add new one
			healthyConns := make([]*UpstreamWSConnection, 0)
			for _, oldConn := range p.connections {
				if oldConn.IsHealthy() {
					healthyConns = append(healthyConns, oldConn)
				}
			}
			healthyConns = append(healthyConns, conn)
			p.connections = healthyConns

			callback := p.onReconnect
			p.lock.Unlock()

			utils.LavaFormatInfo("WebSocket pool: reconnected successfully",
				utils.LogAttr("endpoint", p.sanitizedURL),
				utils.LogAttr("attempts", backoff.Attempt()),
				utils.LogAttr("totalConnections", len(healthyConns)),
			)

			// Call reconnect callback (e.g., to restore subscriptions)
			if callback != nil {
				callback()
			}

			return nil
		}

		// Get next backoff delay
		delay, shouldRetry := backoff.NextBackoff()
		if !shouldRetry {
			return fmt.Errorf("max retries (%d) exceeded for endpoint %s: %w",
				WebSocketMaxRetries, p.sanitizedURL, err)
		}

		utils.LavaFormatDebug("WebSocket pool: reconnect failed, retrying",
			utils.LogAttr("endpoint", p.sanitizedURL),
			utils.LogAttr("attempt", backoff.Attempt()),
			utils.LogAttr("next_delay", delay),
			utils.LogAttr("error", err.Error()),
		)

		// Wait before retry
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
			// Continue to next attempt
		}
	}
}

// waitForReconnect waits for an ongoing reconnection to complete
func (p *UpstreamWSPool) waitForReconnect(ctx context.Context) error {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if !p.reconnecting.Load() {
				// Reconnection finished, check result
				p.lock.RLock()
				hasHealthy := false
				for _, conn := range p.connections {
					if conn.IsHealthy() {
						hasHealthy = true
						break
					}
				}
				p.lock.RUnlock()

				if hasHealthy {
					return nil
				}
				return fmt.Errorf("reconnection failed")
			}
		}
	}
}

// Close closes the pool and all connections
func (p *UpstreamWSPool) Close() error {
	if !p.closed.CompareAndSwap(false, true) {
		return nil // Already closed
	}

	p.lock.Lock()
	defer p.lock.Unlock()

	for _, conn := range p.connections {
		conn.Close()
	}
	p.connections = nil

	utils.LavaFormatDebug("WebSocket pool closed",
		utils.LogAttr("endpoint", p.sanitizedURL),
	)

	return nil
}

// IsHealthy returns true if the pool has at least one healthy connection
func (p *UpstreamWSPool) IsHealthy() bool {
	if p.closed.Load() {
		return false
	}

	p.lock.RLock()
	defer p.lock.RUnlock()

	for _, conn := range p.connections {
		if conn.IsHealthy() {
			return true
		}
	}
	return false
}

// GetEndpoint returns the sanitized endpoint URL
func (p *UpstreamWSPool) GetEndpoint() string {
	return p.sanitizedURL
}

// ConnectionCount returns the current number of connections in the pool
func (p *UpstreamWSPool) ConnectionCount() int {
	p.lock.RLock()
	defer p.lock.RUnlock()
	return len(p.connections)
}

// TotalSubscriptions returns the total number of subscriptions across all connections
func (p *UpstreamWSPool) TotalSubscriptions() int32 {
	p.lock.RLock()
	defer p.lock.RUnlock()

	var total int32
	for _, conn := range p.connections {
		total += conn.SubscriptionCount()
	}
	return total
}

// Note: sanitizeEndpointURL is defined in direct_rpc_relay.go
