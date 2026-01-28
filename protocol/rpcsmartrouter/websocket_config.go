package rpcsmartrouter

import (
	"time"

	"golang.org/x/time/rate"
)

// WebsocketConfig holds all configurable WebSocket parameters for the smart router.
// This aligns with the configuration schema in websocket-design-decisions.md
type WebsocketConfig struct {
	// Subscription limits
	MaxSubscriptionsPerClient int    // Max subscriptions per client connection (default: 25)
	PerClientLimitEnforcement string // "warn" or "reject" (default: "warn")
	MaxTotalSubscriptions     int    // Global subscription warning/limit threshold (default: 5000)
	TotalLimitEnforcement     string // "warn" or "reject" (default: "warn")

	// Subscription sharing
	SubscriptionSharingEnabled bool // Enable subscription deduplication (default: true)

	// Rate limiting
	SubscriptionsPerMinutePerClient int // Max subscription creates per minute per client (default: 10)
	UnsubscribesPerMinutePerClient  int // Max unsubscribe requests per minute per client (default: 20)

	// Message limits
	MaxMessageSize int64 // Maximum message size in bytes (default: 1MB = 1048576)

	// Timeouts
	HandshakeTimeout         time.Duration // WebSocket handshake timeout (default: 10s)
	WriteTimeout             time.Duration // Write timeout for sending messages (default: 10s)
	SubscriptionFirstReplyTimeout time.Duration // Timeout for first subscription reply (default: 10s)

	// Cleanup
	CleanupInterval time.Duration // Interval for stale subscription cleanup (default: 1 minute)

	// Upstream connection pool
	UpstreamPoolMinConnections       int // Minimum connections per endpoint (default: 1)
	UpstreamPoolMaxConnections       int // Maximum connections per endpoint (default: 10)
	UpstreamPoolSubscriptionsPerConn int // Target subscriptions per connection (default: 100)
}

// DefaultWebsocketConfig returns a WebsocketConfig with sensible defaults
// aligned with the design decisions document
func DefaultWebsocketConfig() *WebsocketConfig {
	return &WebsocketConfig{
		// Subscription limits
		MaxSubscriptionsPerClient: 25,
		PerClientLimitEnforcement: "warn",
		MaxTotalSubscriptions:     5000,
		TotalLimitEnforcement:     "warn",

		// Subscription sharing
		SubscriptionSharingEnabled: true,

		// Rate limiting
		SubscriptionsPerMinutePerClient: 10,
		UnsubscribesPerMinutePerClient:  20,

		// Message limits
		MaxMessageSize: 1048576, // 1 MB

		// Timeouts
		HandshakeTimeout:         10 * time.Second,
		WriteTimeout:             10 * time.Second,
		SubscriptionFirstReplyTimeout: 10 * time.Second,

		// Cleanup
		CleanupInterval: 1 * time.Minute,

		// Upstream connection pool
		UpstreamPoolMinConnections:       1,
		UpstreamPoolMaxConnections:       10,
		UpstreamPoolSubscriptionsPerConn: 100,
	}
}

// ClientRateLimiter manages per-client rate limiting for subscription operations
type ClientRateLimiter struct {
	// subscribeLimiters tracks subscription creation rate per client
	subscribeLimiters map[string]*rate.Limiter
	// unsubscribeLimiters tracks unsubscription rate per client
	unsubscribeLimiters map[string]*rate.Limiter

	subscribeRate   rate.Limit // subscriptions per second
	unsubscribeRate rate.Limit // unsubscribes per second
	subscribeBurst  int        // max burst for subscriptions
	unsubscribeBurst int       // max burst for unsubscribes
}

// NewClientRateLimiter creates a new rate limiter based on config
func NewClientRateLimiter(config *WebsocketConfig) *ClientRateLimiter {
	// Convert per-minute limits to per-second rate
	subscribeRate := rate.Limit(float64(config.SubscriptionsPerMinutePerClient) / 60.0)
	unsubscribeRate := rate.Limit(float64(config.UnsubscribesPerMinutePerClient) / 60.0)

	return &ClientRateLimiter{
		subscribeLimiters:   make(map[string]*rate.Limiter),
		unsubscribeLimiters: make(map[string]*rate.Limiter),
		subscribeRate:       subscribeRate,
		unsubscribeRate:     unsubscribeRate,
		subscribeBurst:      config.SubscriptionsPerMinutePerClient, // Allow burst up to full minute's worth
		unsubscribeBurst:    config.UnsubscribesPerMinutePerClient,
	}
}

// AllowSubscribe checks if the client is allowed to create a subscription
func (crl *ClientRateLimiter) AllowSubscribe(clientKey string) bool {
	limiter, exists := crl.subscribeLimiters[clientKey]
	if !exists {
		limiter = rate.NewLimiter(crl.subscribeRate, crl.subscribeBurst)
		crl.subscribeLimiters[clientKey] = limiter
	}
	return limiter.Allow()
}

// AllowUnsubscribe checks if the client is allowed to unsubscribe
func (crl *ClientRateLimiter) AllowUnsubscribe(clientKey string) bool {
	limiter, exists := crl.unsubscribeLimiters[clientKey]
	if !exists {
		limiter = rate.NewLimiter(crl.unsubscribeRate, crl.unsubscribeBurst)
		crl.unsubscribeLimiters[clientKey] = limiter
	}
	return limiter.Allow()
}

// CleanupClient removes rate limiters for a disconnected client
func (crl *ClientRateLimiter) CleanupClient(clientKey string) {
	delete(crl.subscribeLimiters, clientKey)
	delete(crl.unsubscribeLimiters, clientKey)
}

// EnforcementMode represents how limit violations are handled
type EnforcementMode string

const (
	EnforcementModeWarn   EnforcementMode = "warn"
	EnforcementModeReject EnforcementMode = "reject"
)

// ShouldReject returns true if the enforcement mode is "reject"
func (config *WebsocketConfig) ShouldRejectOnClientLimit() bool {
	return config.PerClientLimitEnforcement == string(EnforcementModeReject)
}

// ShouldRejectOnTotalLimit returns true if total limit enforcement is "reject"
func (config *WebsocketConfig) ShouldRejectOnTotalLimit() bool {
	return config.TotalLimitEnforcement == string(EnforcementModeReject)
}
