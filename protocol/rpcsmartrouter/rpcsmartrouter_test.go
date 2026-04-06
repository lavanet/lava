package rpcsmartrouter

import (
	"context"
	"fmt"
	"net"
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/v5/protocol/lavasession"
	"github.com/lavanet/lava/v5/protocol/provideroptimizer"
	"github.com/lavanet/lava/v5/utils/rand"
	"github.com/stretchr/testify/require"
)

func TestUpdateEpoch_FreshSessions(t *testing.T) {
	// 0. Initialize random seed for tests
	rand.InitRandomSeed()

	// 1. Setup RPCSmartRouter
	rpsr := &RPCSmartRouter{
		sessionManagers:        make(map[string]*lavasession.ConsumerSessionManager),
		providerSessions:       make(map[string]map[uint64]*lavasession.ConsumerSessionsWithProvider),
		backupProviderSessions: make(map[string]map[uint64]*lavasession.ConsumerSessionsWithProvider),
	}

	// 2. Setup dependencies for SessionManager
	rpcEndpoint := &lavasession.RPCEndpoint{
		ChainID:        "LAV1",
		ApiInterface:   "tendermintrpc",
		NetworkAddress: "127.0.0.1:3333",
	}
	optimizer := provideroptimizer.NewProviderOptimizer(provideroptimizer.StrategyBalanced, time.Second, uint(1), nil, "LAV1")

	chainKey := rpcEndpoint.Key()
	sessionManager := lavasession.NewConsumerSessionManager(rpcEndpoint, optimizer, nil, nil, "test-router", lavasession.NewActiveSubscriptionProvidersStorage())
	rpsr.sessionManagers[chainKey] = sessionManager

	// 3. Create initial provider session
	providerAddr := "lava@provider1"
	initialEpoch := uint64(1)

	initialSession := lavasession.NewConsumerSessionWithProvider(
		providerAddr,
		[]*lavasession.Endpoint{{NetworkAddress: "http://provider:8080", Enabled: true}},
		100,
		initialEpoch,
		sdk.NewCoin("ulava", sdk.NewInt(1)),
	)
	initialSession.StaticProvider = true

	rpsr.providerSessions[chainKey] = map[uint64]*lavasession.ConsumerSessionsWithProvider{
		0: initialSession,
	}

	// 4. Trigger Epoch Update
	newEpoch := uint64(2)
	rpsr.updateEpoch(newEpoch)

	// 5. Verify results
	// Get the updated session map
	updatedSessionsMap := rpsr.providerSessions[chainKey]
	require.NotNil(t, updatedSessionsMap, "Provider sessions map should not be nil")

	updatedSession := updatedSessionsMap[0]
	require.NotNil(t, updatedSession, "Updated session should not be nil")

	// Verify it's a different object (fresh instance)
	require.False(t, initialSession == updatedSession, "Session object should be replaced with a fresh instance")

	// Verify properties are preserved/updated correctly
	require.Equal(t, providerAddr, updatedSession.PublicLavaAddress)
	require.Equal(t, newEpoch, updatedSession.PairingEpoch)
	require.True(t, updatedSession.StaticProvider)

	// Verify SessionManager was updated (by checking internal state if possible,
	// or at least that no panic occurred and the flow completed)
	// We can't easily check SessionManager internal state as it's private,
	// but the fact that updateEpoch completed means UpdateAllProviders was called.
}

func TestUpdateEpoch_ResetsDisabledEndpoints(t *testing.T) {
	rand.InitRandomSeed()

	rpsr := &RPCSmartRouter{
		sessionManagers:        make(map[string]*lavasession.ConsumerSessionManager),
		providerSessions:       make(map[string]map[uint64]*lavasession.ConsumerSessionsWithProvider),
		backupProviderSessions: make(map[string]map[uint64]*lavasession.ConsumerSessionsWithProvider),
	}

	rpcEndpoint := &lavasession.RPCEndpoint{
		ChainID:        "LAV1",
		ApiInterface:   "tendermintrpc",
		NetworkAddress: "127.0.0.1:3334",
	}
	optimizer := provideroptimizer.NewProviderOptimizer(provideroptimizer.StrategyBalanced, time.Second, uint(1), nil, "LAV1")
	chainKey := rpcEndpoint.Key()
	sessionManager := lavasession.NewConsumerSessionManager(rpcEndpoint, optimizer, nil, nil, "test-router", lavasession.NewActiveSubscriptionProvidersStorage())
	rpsr.sessionManagers[chainKey] = sessionManager

	// Create endpoints that are disabled — simulating 5 consecutive failures.
	disabledEndpoint := &lavasession.Endpoint{
		NetworkAddress:     "http://provider1:8080",
		Enabled:            false,
		ConnectionRefusals: lavasession.MaxConsecutiveConnectionAttempts,
	}
	disabledBackupEndpoint := &lavasession.Endpoint{
		NetworkAddress:     "http://backup1:8080",
		Enabled:            false,
		ConnectionRefusals: lavasession.MaxConsecutiveConnectionAttempts,
	}

	initialEpoch := uint64(1)

	providerSession := lavasession.NewConsumerSessionWithProvider(
		"lava@provider1",
		[]*lavasession.Endpoint{disabledEndpoint},
		100,
		initialEpoch,
		sdk.NewCoin("ulava", sdk.NewInt(1)),
	)
	providerSession.StaticProvider = true

	backupSession := lavasession.NewConsumerSessionWithProvider(
		"lava@backup1",
		[]*lavasession.Endpoint{disabledBackupEndpoint},
		100,
		initialEpoch,
		sdk.NewCoin("ulava", sdk.NewInt(1)),
	)
	backupSession.StaticProvider = true

	rpsr.providerSessions[chainKey] = map[uint64]*lavasession.ConsumerSessionsWithProvider{0: providerSession}
	rpsr.backupProviderSessions[chainKey] = map[uint64]*lavasession.ConsumerSessionsWithProvider{0: backupSession}

	rpsr.updateEpoch(uint64(2))

	// Direct field reads below are safe without mu: updateEpoch is synchronous and
	// has fully returned, so no other goroutine holds or can acquire the endpoint lock.
	require.True(t, disabledEndpoint.Enabled, "provider endpoint should be re-enabled after epoch transition")
	require.Equal(t, uint64(0), disabledEndpoint.ConnectionRefusals, "provider endpoint refusals should be reset")

	require.True(t, disabledBackupEndpoint.Enabled, "backup endpoint should be re-enabled after epoch transition")
	require.Equal(t, uint64(0), disabledBackupEndpoint.ConnectionRefusals, "backup endpoint refusals should be reset")
}

func TestClassifyRelayError(t *testing.T) {
	tests := []struct {
		name              string
		err               error
		ctx               context.Context
		wantMarkUnhealthy bool
		wantNeedsBackoff  bool
	}{
		{
			name:              "HTTP 500 marks unhealthy and backs off",
			err:               &lavasession.HTTPStatusError{StatusCode: 500},
			ctx:               context.Background(),
			wantMarkUnhealthy: true,
			wantNeedsBackoff:  true,
		},
		{
			name:              "HTTP 502 marks unhealthy and backs off",
			err:               &lavasession.HTTPStatusError{StatusCode: 502},
			ctx:               context.Background(),
			wantMarkUnhealthy: true,
			wantNeedsBackoff:  true,
		},
		{
			name:              "HTTP 429 backs off but does not mark unhealthy",
			err:               &lavasession.HTTPStatusError{StatusCode: 429},
			ctx:               context.Background(),
			wantMarkUnhealthy: false,
			wantNeedsBackoff:  true,
		},
		{
			name:              "HTTP 400 client error: no action",
			err:               &lavasession.HTTPStatusError{StatusCode: 400},
			ctx:               context.Background(),
			wantMarkUnhealthy: false,
			wantNeedsBackoff:  false,
		},
		{
			name:              "HTTP 404 client error: no action",
			err:               &lavasession.HTTPStatusError{StatusCode: 404},
			ctx:               context.Background(),
			wantMarkUnhealthy: false,
			wantNeedsBackoff:  false,
		},
		{
			name:              "non-HTTP error (connection refused) marks unhealthy",
			err:               fmt.Errorf("connection refused"),
			ctx:               context.Background(),
			wantMarkUnhealthy: true,
			wantNeedsBackoff:  true,
		},
		{
			name: "context.Canceled with canceled ctx skips unhealthy",
			err:  context.Canceled,
			ctx: func() context.Context {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				return ctx
			}(),
			wantMarkUnhealthy: false,
			wantNeedsBackoff:  false,
		},
		{
			name: "context.DeadlineExceeded with expired ctx still marks unhealthy",
			err:  context.DeadlineExceeded,
			ctx: func() context.Context {
				ctx, cancel := context.WithTimeout(context.Background(), time.Nanosecond)
				defer cancel()
				// Ensure deadline fires
				time.Sleep(time.Millisecond)
				return ctx
			}(),
			wantMarkUnhealthy: true,
			wantNeedsBackoff:  true,
		},
		{
			name: "wrapped context.Canceled with canceled ctx skips unhealthy",
			err:  fmt.Errorf("rpc failed: %w", context.Canceled),
			ctx: func() context.Context {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				return ctx
			}(),
			wantMarkUnhealthy: false,
			wantNeedsBackoff:  false,
		},
		{
			name:              "context.Canceled error but live ctx marks unhealthy (server-side cancel)",
			err:               context.Canceled,
			ctx:               context.Background(),
			wantMarkUnhealthy: true,
			wantNeedsBackoff:  true,
		},
		{
			name:              "network dial error marks unhealthy",
			err:               &net.OpError{Op: "dial", Err: fmt.Errorf("no route to host")},
			ctx:               context.Background(),
			wantMarkUnhealthy: true,
			wantNeedsBackoff:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			unhealthy, backoff := classifyRelayError(tt.err, tt.ctx)
			require.Equal(t, tt.wantMarkUnhealthy, unhealthy, "shouldMarkUnhealthy")
			require.Equal(t, tt.wantNeedsBackoff, backoff, "needsBackoff")
		})
	}
}
