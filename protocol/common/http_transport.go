package common

import (
	"net"
	"net/http"
	"time"
)

// HTTP Connection Pool Configuration
// These values are optimized for high-concurrency scenarios where providers
// handle many simultaneous requests to blockchain nodes.
const (
	// MaxIdleConns controls the maximum number of idle (keep-alive) connections across all hosts.
	// Setting this higher prevents constantly creating new TCP connections.
	// Go default: 100
	DefaultMaxIdleConns = 200

	// MaxIdleConnsPerHost controls the maximum idle (keep-alive) connections to keep per-host.
	// This is critical for blockchain node connections where we repeatedly connect to the same node.
	// Go default: 2 (way too low for high-concurrency!)
	DefaultMaxIdleConnsPerHost = 50

	// MaxConnsPerHost limits the total number of connections per host, including those in active use.
	// This prevents overwhelming a single blockchain node with too many connections.
	// Go default: 0 (unlimited - can cause node overload)
	DefaultMaxConnsPerHost = 100

	// IdleConnTimeout is the maximum amount of time an idle connection will remain idle before closing.
	// Keeps connections alive for reuse but eventually closes them to avoid resource leaks.
	// Go default: 90s
	DefaultIdleConnTimeout = 90 * time.Second

	// TLSHandshakeTimeout is the maximum amount of time waiting to perform a TLS handshake.
	// Go default: 10s
	DefaultTLSHandshakeTimeout = 10 * time.Second

	// ExpectContinueTimeout limits the time the client will wait between sending the request headers
	// and receiving the go-ahead to send the request body.
	// Go default: 1s
	DefaultExpectContinueTimeout = 1 * time.Second

	// ResponseHeaderTimeout is the amount of time to wait for a server's response headers.
	// This should be relatively high for blockchain nodes as they may be slow under load.
	// Go default: 0 (no timeout - can hang forever)
	DefaultResponseHeaderTimeout = 0

	// DialTimeout is the maximum amount of time a dial will wait for a connect to complete.
	// Go default: 30s
	DefaultDialTimeout = 10 * time.Second

	// KeepAlive specifies the interval between keep-alive probes for an active network connection.
	// Go default: 30s
	DefaultKeepAlive = 30 * time.Second

	// DefaultHTTPTimeout is the default timeout for the entire request/response cycle.
	// Set to 5 minutes to handle slow blockchain node operations like trace_block.
	// Go default: 0 (no timeout - requests can hang forever)
	DefaultHTTPTimeout = 5 * time.Minute
)

// OptimizedHttpTransport creates an HTTP transport optimized for provider-to-node communication.
// This transport is configured to:
// 1. Reuse connections efficiently (high MaxIdleConnsPerHost)
// 2. Limit total connections per host (prevents overwhelming nodes)
// 3. Handle high concurrency scenarios (200+ simultaneous requests)
// 4. Close idle connections appropriately to avoid leaks
//
// Benefits:
// - Reduces TCP connection overhead
// - Prevents connection exhaustion on blockchain nodes
// - Improves latency through connection reuse
// - Handles heavy load without creating thousands of connections
func OptimizedHttpTransport() *http.Transport {
	return &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   DefaultDialTimeout,
			KeepAlive: DefaultKeepAlive,
		}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          DefaultMaxIdleConns,
		MaxIdleConnsPerHost:   DefaultMaxIdleConnsPerHost,
		MaxConnsPerHost:       DefaultMaxConnsPerHost,
		IdleConnTimeout:       DefaultIdleConnTimeout,
		TLSHandshakeTimeout:   DefaultTLSHandshakeTimeout,
		ExpectContinueTimeout: DefaultExpectContinueTimeout,
		ResponseHeaderTimeout: DefaultResponseHeaderTimeout,
	}
}

// OptimizedHttpClient creates an HTTP client with optimized transport settings
// and a default 5-minute timeout suitable for blockchain node operations.
// The client uses a custom transport configured for high-concurrency scenarios.
//
// Returns:
//   - *http.Client: A client with optimized connection pooling and default timeout
func OptimizedHttpClient() *http.Client {
	return &http.Client{
		Timeout:   DefaultHTTPTimeout,
		Transport: OptimizedHttpTransport(),
	}
}
