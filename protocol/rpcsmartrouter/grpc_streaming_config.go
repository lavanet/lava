package rpcsmartrouter

import (
	"time"

	"golang.org/x/time/rate"
)

// GRPCStreamingConfig holds all configurable parameters for gRPC streaming subscriptions.
// This follows the same pattern as WebsocketConfig for consistency.
type GRPCStreamingConfig struct {
	// Subscription limits
	MaxSubscriptionsPerClient int    // Max subscriptions per client connection (default: 25)
	PerClientLimitEnforcement string // "warn" or "reject" (default: "warn")
	MaxTotalSubscriptions     int    // Global subscription warning/limit threshold (default: 10000)
	TotalLimitEnforcement     string // "warn" or "reject" (default: "warn")

	// Subscription sharing (deduplication)
	SubscriptionSharingEnabled bool // Enable subscription deduplication (default: true)

	// Rate limiting
	SubscriptionsPerMinutePerClient int // Max subscription creates per minute per client (default: 30)
	UnsubscribesPerMinutePerClient  int // Max unsubscribe requests per minute per client (default: 60)

	// Timeouts
	ConnectionTimeout         time.Duration // gRPC connection establishment timeout (default: 30s)
	StreamIdleTimeout         time.Duration // Close streams with no messages for this duration (default: 5m)
	SubscriptionFirstReplyTimeout time.Duration // Timeout waiting for first streaming message (default: 30s)

	// Cleanup
	CleanupInterval time.Duration // Interval for stale subscription cleanup (default: 1 minute)

	// Upstream connection pool
	PoolMinConnections   int // Minimum gRPC connections per endpoint (default: 1)
	PoolMaxConnections   int // Maximum gRPC connections per endpoint (default: 5)
	StreamsPerConnection int // Target concurrent streams per connection (default: 100)
}

// DefaultGRPCStreamingConfig returns a GRPCStreamingConfig with sensible defaults
func DefaultGRPCStreamingConfig() *GRPCStreamingConfig {
	return &GRPCStreamingConfig{
		// Subscription limits
		MaxSubscriptionsPerClient: 25,
		PerClientLimitEnforcement: "warn",
		MaxTotalSubscriptions:     10000,
		TotalLimitEnforcement:     "warn",

		// Subscription sharing
		SubscriptionSharingEnabled: true,

		// Rate limiting - gRPC streaming tends to be more efficient, allow higher rates
		SubscriptionsPerMinutePerClient: 30,
		UnsubscribesPerMinutePerClient:  60,

		// Timeouts - gRPC streaming may take longer to establish
		ConnectionTimeout:         30 * time.Second,
		StreamIdleTimeout:         5 * time.Minute,
		SubscriptionFirstReplyTimeout: 30 * time.Second,

		// Cleanup
		CleanupInterval: 1 * time.Minute,

		// Upstream connection pool - gRPC multiplexes streams over fewer connections
		PoolMinConnections:   1,
		PoolMaxConnections:   5,
		StreamsPerConnection: 100,
	}
}

// GRPCClientRateLimiter manages per-client rate limiting for gRPC streaming operations.
// This is similar to ClientRateLimiter but can be extended for gRPC-specific needs.
type GRPCClientRateLimiter struct {
	// subscribeLimiters tracks subscription creation rate per client
	subscribeLimiters map[string]*rate.Limiter
	// unsubscribeLimiters tracks unsubscription rate per client
	unsubscribeLimiters map[string]*rate.Limiter

	subscribeRate    rate.Limit // subscriptions per second
	unsubscribeRate  rate.Limit // unsubscribes per second
	subscribeBurst   int        // max burst for subscriptions
	unsubscribeBurst int        // max burst for unsubscribes
}

// NewGRPCClientRateLimiter creates a new rate limiter based on gRPC streaming config
func NewGRPCClientRateLimiter(config *GRPCStreamingConfig) *GRPCClientRateLimiter {
	// Convert per-minute limits to per-second rate
	subscribeRate := rate.Limit(float64(config.SubscriptionsPerMinutePerClient) / 60.0)
	unsubscribeRate := rate.Limit(float64(config.UnsubscribesPerMinutePerClient) / 60.0)

	return &GRPCClientRateLimiter{
		subscribeLimiters:   make(map[string]*rate.Limiter),
		unsubscribeLimiters: make(map[string]*rate.Limiter),
		subscribeRate:       subscribeRate,
		unsubscribeRate:     unsubscribeRate,
		subscribeBurst:      config.SubscriptionsPerMinutePerClient,
		unsubscribeBurst:    config.UnsubscribesPerMinutePerClient,
	}
}

// AllowSubscribe checks if the client is allowed to create a gRPC subscription
func (crl *GRPCClientRateLimiter) AllowSubscribe(clientKey string) bool {
	limiter, exists := crl.subscribeLimiters[clientKey]
	if !exists {
		limiter = rate.NewLimiter(crl.subscribeRate, crl.subscribeBurst)
		crl.subscribeLimiters[clientKey] = limiter
	}
	return limiter.Allow()
}

// AllowUnsubscribe checks if the client is allowed to unsubscribe from a gRPC stream
func (crl *GRPCClientRateLimiter) AllowUnsubscribe(clientKey string) bool {
	limiter, exists := crl.unsubscribeLimiters[clientKey]
	if !exists {
		limiter = rate.NewLimiter(crl.unsubscribeRate, crl.unsubscribeBurst)
		crl.unsubscribeLimiters[clientKey] = limiter
	}
	return limiter.Allow()
}

// CleanupClient removes rate limiters for a disconnected client
func (crl *GRPCClientRateLimiter) CleanupClient(clientKey string) {
	delete(crl.subscribeLimiters, clientKey)
	delete(crl.unsubscribeLimiters, clientKey)
}

// ShouldRejectOnClientLimit returns true if the enforcement mode is "reject"
func (config *GRPCStreamingConfig) ShouldRejectOnClientLimit() bool {
	return config.PerClientLimitEnforcement == string(EnforcementModeReject)
}

// ShouldRejectOnTotalLimit returns true if total limit enforcement is "reject"
func (config *GRPCStreamingConfig) ShouldRejectOnTotalLimit() bool {
	return config.TotalLimitEnforcement == string(EnforcementModeReject)
}

// gRPC streaming metadata header keys for subscription routing
const (
	// MetadataGRPCSubscriptionID is the header key for router-assigned subscription ID
	MetadataGRPCSubscriptionID = "x-lava-grpc-sub-id"
	// MetadataGRPCStreamSeq is the header key for stream message sequence number
	MetadataGRPCStreamSeq = "x-lava-grpc-stream-seq"
	// MetadataGRPCClientID is the header key for client identification
	MetadataGRPCClientID = "x-lava-grpc-client-id"
)
