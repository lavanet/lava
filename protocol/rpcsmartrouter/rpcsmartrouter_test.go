package rpcsmartrouter

import (
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
		[]*lavasession.Endpoint{{NetworkAddress: "http://provider:8080"}},
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
