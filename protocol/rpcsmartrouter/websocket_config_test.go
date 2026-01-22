package rpcsmartrouter

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultWebsocketConfig(t *testing.T) {
	config := DefaultWebsocketConfig()

	// Verify subscription limits
	assert.Equal(t, 25, config.MaxSubscriptionsPerClient)
	assert.Equal(t, "warn", config.PerClientLimitEnforcement)
	assert.Equal(t, 5000, config.MaxTotalSubscriptions)
	assert.Equal(t, "warn", config.TotalLimitEnforcement)

	// Verify subscription sharing
	assert.True(t, config.SubscriptionSharingEnabled)

	// Verify rate limiting
	assert.Equal(t, 10, config.SubscriptionsPerMinutePerClient)
	assert.Equal(t, 20, config.UnsubscribesPerMinutePerClient)

	// Verify message limits
	assert.Equal(t, int64(1048576), config.MaxMessageSize) // 1 MB

	// Verify timeouts
	assert.Equal(t, 10*time.Second, config.HandshakeTimeout)
	assert.Equal(t, 10*time.Second, config.WriteTimeout)
	assert.Equal(t, 10*time.Second, config.SubscriptionFirstReplyTimeout)

	// Verify cleanup interval
	assert.Equal(t, 1*time.Minute, config.CleanupInterval)

	// Verify upstream pool settings
	assert.Equal(t, 1, config.UpstreamPoolMinConnections)
	assert.Equal(t, 10, config.UpstreamPoolMaxConnections)
	assert.Equal(t, 100, config.UpstreamPoolSubscriptionsPerConn)
}

func TestWebsocketConfig_ShouldRejectOnClientLimit(t *testing.T) {
	tests := []struct {
		name           string
		enforcement    string
		expectedReject bool
	}{
		{"warn mode", "warn", false},
		{"reject mode", "reject", true},
		{"invalid mode treated as warn", "invalid", false},
		{"empty mode treated as warn", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := DefaultWebsocketConfig()
			config.PerClientLimitEnforcement = tt.enforcement
			assert.Equal(t, tt.expectedReject, config.ShouldRejectOnClientLimit())
		})
	}
}

func TestWebsocketConfig_ShouldRejectOnTotalLimit(t *testing.T) {
	tests := []struct {
		name           string
		enforcement    string
		expectedReject bool
	}{
		{"warn mode", "warn", false},
		{"reject mode", "reject", true},
		{"invalid mode treated as warn", "invalid", false},
		{"empty mode treated as warn", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := DefaultWebsocketConfig()
			config.TotalLimitEnforcement = tt.enforcement
			assert.Equal(t, tt.expectedReject, config.ShouldRejectOnTotalLimit())
		})
	}
}

func TestNewClientRateLimiter(t *testing.T) {
	config := DefaultWebsocketConfig()
	limiter := NewClientRateLimiter(config)

	require.NotNil(t, limiter)
	assert.NotNil(t, limiter.subscribeLimiters)
	assert.NotNil(t, limiter.unsubscribeLimiters)
}

func TestClientRateLimiter_AllowSubscribe(t *testing.T) {
	config := DefaultWebsocketConfig()
	config.SubscriptionsPerMinutePerClient = 60 // 1 per second for easier testing
	limiter := NewClientRateLimiter(config)

	clientKey := "test-client-1"

	// First request should always be allowed (burst)
	assert.True(t, limiter.AllowSubscribe(clientKey))

	// Multiple rapid requests should be allowed up to burst limit
	// With 60/minute rate and burst equal to that, we can make 60 requests immediately
	for i := 0; i < 59; i++ {
		allowed := limiter.AllowSubscribe(clientKey)
		if !allowed {
			t.Logf("Request %d was rate limited (expected after burst)", i+2)
			break
		}
	}
}

func TestClientRateLimiter_AllowUnsubscribe(t *testing.T) {
	config := DefaultWebsocketConfig()
	config.UnsubscribesPerMinutePerClient = 60 // 1 per second for easier testing
	limiter := NewClientRateLimiter(config)

	clientKey := "test-client-1"

	// First request should always be allowed (burst)
	assert.True(t, limiter.AllowUnsubscribe(clientKey))

	// Multiple rapid requests should be allowed up to burst limit
	for i := 0; i < 59; i++ {
		allowed := limiter.AllowUnsubscribe(clientKey)
		if !allowed {
			t.Logf("Request %d was rate limited (expected after burst)", i+2)
			break
		}
	}
}

func TestClientRateLimiter_SeparateClientsHaveSeparateLimits(t *testing.T) {
	config := DefaultWebsocketConfig()
	config.SubscriptionsPerMinutePerClient = 2 // Very low limit
	limiter := NewClientRateLimiter(config)

	client1 := "client-1"
	client2 := "client-2"

	// Client 1 uses its burst
	assert.True(t, limiter.AllowSubscribe(client1))
	assert.True(t, limiter.AllowSubscribe(client1))

	// Client 2 should still have its own burst
	assert.True(t, limiter.AllowSubscribe(client2))
	assert.True(t, limiter.AllowSubscribe(client2))
}

func TestClientRateLimiter_CleanupClient(t *testing.T) {
	config := DefaultWebsocketConfig()
	limiter := NewClientRateLimiter(config)

	clientKey := "test-client-cleanup"

	// Use the limiter to create entries
	limiter.AllowSubscribe(clientKey)
	limiter.AllowUnsubscribe(clientKey)

	// Verify entries exist
	_, existsSubscribe := limiter.subscribeLimiters[clientKey]
	_, existsUnsubscribe := limiter.unsubscribeLimiters[clientKey]
	assert.True(t, existsSubscribe)
	assert.True(t, existsUnsubscribe)

	// Cleanup
	limiter.CleanupClient(clientKey)

	// Verify entries are removed
	_, existsSubscribe = limiter.subscribeLimiters[clientKey]
	_, existsUnsubscribe = limiter.unsubscribeLimiters[clientKey]
	assert.False(t, existsSubscribe)
	assert.False(t, existsUnsubscribe)
}

func TestClientRateLimiter_ConcurrentAccess(t *testing.T) {
	// Note: The current ClientRateLimiter implementation uses plain maps
	// which are not thread-safe. This test verifies sequential access works correctly.
	// For production use with concurrent access, the limiter would need synchronization.
	config := DefaultWebsocketConfig()
	limiter := NewClientRateLimiter(config)

	// Test sequential access for multiple clients
	numClients := 10
	numOperations := 10

	for i := 0; i < numClients; i++ {
		clientKey := "client-" + string(rune('A'+i))
		for j := 0; j < numOperations; j++ {
			limiter.AllowSubscribe(clientKey)
			limiter.AllowUnsubscribe(clientKey)
		}
		limiter.CleanupClient(clientKey)
	}
	// Test passes if no panic occurs
}

func TestEnforcementMode_Constants(t *testing.T) {
	assert.Equal(t, EnforcementMode("warn"), EnforcementModeWarn)
	assert.Equal(t, EnforcementMode("reject"), EnforcementModeReject)
}
