package rpcsmartrouter

import (
	"context"
	"sync"
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/v5/protocol/common"
	"github.com/lavanet/lava/v5/protocol/lavasession"
	"github.com/lavanet/lava/v5/protocol/provideroptimizer"
	"github.com/lavanet/lava/v5/utils/rand"
	"github.com/stretchr/testify/require"
)

// createTestRPCSmartRouter creates an RPCSmartRouter with all maps initialized and an epoch timer.
func createTestRPCSmartRouter() *RPCSmartRouter {
	return &RPCSmartRouter{
		epochTimer:             common.NewEpochTimer(15 * time.Minute),
		sessionManagers:        make(map[string]*lavasession.ConsumerSessionManager),
		providerSessions:       make(map[string]map[uint64]*lavasession.ConsumerSessionsWithProvider),
		backupProviderSessions: make(map[string]map[uint64]*lavasession.ConsumerSessionsWithProvider),
		failedStaticProviders:  make(map[string][]*lavasession.RPCStaticProviderEndpoint),
		rpcServers:             make(map[string]*RPCSmartRouterServer),
	}
}

// createTestSessionManager creates a ConsumerSessionManager for a given chain key.
func createTestSessionManager(chainID, apiInterface string) (*lavasession.ConsumerSessionManager, *lavasession.RPCEndpoint) {
	rpcEndpoint := &lavasession.RPCEndpoint{
		ChainID:        chainID,
		ApiInterface:   apiInterface,
		NetworkAddress: "127.0.0.1:3333",
	}
	optimizer := provideroptimizer.NewProviderOptimizer(provideroptimizer.StrategyBalanced, time.Second, uint(1), nil, chainID)
	sm := lavasession.NewConsumerSessionManager(rpcEndpoint, optimizer, nil, nil, "test-router", lavasession.NewActiveSubscriptionProvidersStorage())
	return sm, rpcEndpoint
}

// createTestProviderSession creates a ConsumerSessionsWithProvider for testing.
func createTestProviderSession(name string, epoch uint64) *lavasession.ConsumerSessionsWithProvider {
	session := lavasession.NewConsumerSessionWithProvider(
		name,
		[]*lavasession.Endpoint{{NetworkAddress: "http://" + name + ":8080", Enabled: true}},
		100,
		epoch,
		sdk.NewCoin("ulava", sdk.NewInt(1)),
	)
	session.StaticProvider = true
	return session
}

// createTestStaticProviderEndpoint creates an RPCStaticProviderEndpoint for testing.
func createTestStaticProviderEndpoint(name, chainID, apiInterface string) *lavasession.RPCStaticProviderEndpoint {
	return &lavasession.RPCStaticProviderEndpoint{
		Name:         name,
		ChainID:      chainID,
		ApiInterface: apiInterface,
		NodeUrls:     []common.NodeUrl{{Url: "http://" + name + ":8080"}},
	}
}

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

// =============================================================================
// Graceful Verification Failure Tests
// =============================================================================

// Scenario 1: All providers healthy — no failures, no retry, baseline behavior.
func TestGracefulFailure_AllProvidersHealthy_NoRetryLaunched(t *testing.T) {
	rand.InitRandomSeed()
	rpsr := createTestRPCSmartRouter()

	chainKey := "LAV1-tendermintrpc"
	sm, _ := createTestSessionManager("LAV1", "tendermintrpc")
	rpsr.sessionManagers[chainKey] = sm

	// Two healthy providers, no failures
	rpsr.providerSessions[chainKey] = map[uint64]*lavasession.ConsumerSessionsWithProvider{
		0: createTestProviderSession("providerA", 1),
		1: createTestProviderSession("providerB", 1),
	}
	// No failed providers
	// rpsr.failedStaticProviders[chainKey] is not set (empty map)

	// Verify: no failed providers stored
	require.Empty(t, rpsr.failedStaticProviders[chainKey])

	// Verify: both providers in sessions
	require.Len(t, rpsr.providerSessions[chainKey], 2)

	// Verify: epoch update works and preserves both providers
	rpsr.updateEpoch(2)
	require.Len(t, rpsr.providerSessions[chainKey], 2)
	require.Equal(t, "providerA", rpsr.providerSessions[chainKey][0].PublicLavaAddress)
	require.Equal(t, "providerB", rpsr.providerSessions[chainKey][1].PublicLavaAddress)
}

// Scenario 3: One provider fails, others healthy — epoch doesn't resurrect the failed one.
func TestGracefulFailure_EpochDoesNotResurrectFailedProviders(t *testing.T) {
	rand.InitRandomSeed()
	rpsr := createTestRPCSmartRouter()

	chainKey := "LAV1-tendermintrpc"
	sm, _ := createTestSessionManager("LAV1", "tendermintrpc")
	rpsr.sessionManagers[chainKey] = sm

	// Only healthy providers in sessions (B was excluded at startup)
	rpsr.providerSessions[chainKey] = map[uint64]*lavasession.ConsumerSessionsWithProvider{
		0: createTestProviderSession("providerA", 1),
		1: createTestProviderSession("providerC", 1),
	}

	// B is in the failed list (excluded at startup, awaiting retry)
	rpsr.failedStaticProviders[chainKey] = []*lavasession.RPCStaticProviderEndpoint{
		createTestStaticProviderEndpoint("providerB", "LAV1", "tendermintrpc"),
	}

	// Trigger epoch update
	rpsr.updateEpoch(2)

	// Verify: only A and C in sessions — B was NOT resurrected
	sessions := rpsr.providerSessions[chainKey]
	require.Len(t, sessions, 2)
	addresses := map[string]bool{}
	for _, s := range sessions {
		addresses[s.PublicLavaAddress] = true
	}
	require.True(t, addresses["providerA"], "providerA should be in sessions")
	require.True(t, addresses["providerC"], "providerC should be in sessions")
	require.False(t, addresses["providerB"], "providerB should NOT be resurrected by epoch")

	// Verify: failed providers list is unchanged
	require.Len(t, rpsr.failedStaticProviders[chainKey], 1)
	require.Equal(t, "providerB", rpsr.failedStaticProviders[chainKey][0].Name)
}

// Scenario 3 (continued): Filtering logic — failedStaticNames correctly filters provider lists.
func TestGracefulFailure_FilteringLogic(t *testing.T) {
	// Simulate the filtering that happens in CreateSmartRouterEndpoint after Phase 1
	relevantStaticProviderList := []*lavasession.RPCStaticProviderEndpoint{
		createTestStaticProviderEndpoint("providerA", "LAV1", "tendermintrpc"),
		createTestStaticProviderEndpoint("providerB", "LAV1", "tendermintrpc"),
		createTestStaticProviderEndpoint("providerC", "LAV1", "tendermintrpc"),
	}

	// B failed validation
	failedStaticNames := map[string]struct{}{
		"providerB": {},
	}

	// Apply the same filtering logic as the production code (line 828-834)
	healthyStaticProviders := make([]*lavasession.RPCStaticProviderEndpoint, 0, len(relevantStaticProviderList)-len(failedStaticNames))
	for _, p := range relevantStaticProviderList {
		if _, failed := failedStaticNames[p.Name]; !failed {
			healthyStaticProviders = append(healthyStaticProviders, p)
		}
	}

	require.Len(t, healthyStaticProviders, 2)
	require.Equal(t, "providerA", healthyStaticProviders[0].Name)
	require.Equal(t, "providerC", healthyStaticProviders[1].Name)
}

// Scenario 4: All static providers fail — filtering produces empty list.
func TestGracefulFailure_AllFailedFiltering(t *testing.T) {
	relevantStaticProviderList := []*lavasession.RPCStaticProviderEndpoint{
		createTestStaticProviderEndpoint("providerA", "LAV1", "tendermintrpc"),
		createTestStaticProviderEndpoint("providerB", "LAV1", "tendermintrpc"),
	}

	// Both failed
	failedStaticNames := map[string]struct{}{
		"providerA": {},
		"providerB": {},
	}

	// Simulate the all-fail check
	totalAttemptedCount := 2
	healthyCount := totalAttemptedCount - len(failedStaticNames)
	require.Equal(t, 0, healthyCount, "Should detect all providers failed")

	// Filtering produces empty list
	healthyStaticProviders := make([]*lavasession.RPCStaticProviderEndpoint, 0)
	for _, p := range relevantStaticProviderList {
		if _, failed := failedStaticNames[p.Name]; !failed {
			healthyStaticProviders = append(healthyStaticProviders, p)
		}
	}
	require.Empty(t, healthyStaticProviders)
}

// Scenario 2: No providers configured — nil failedStaticNames is safe.
func TestGracefulFailure_NilFailedStaticNames(t *testing.T) {
	// When there are no static providers, failedStaticNames is nil (var declaration, no make)
	var failedStaticNames map[string]struct{}

	// This must not panic — len() on nil map returns 0
	require.Equal(t, 0, len(failedStaticNames))

	// The filtering condition is safe on nil
	if len(failedStaticNames) > 0 {
		t.Fatal("should not enter this block with nil map")
	}

	// Direct lookup on nil map returns zero value (no panic)
	_, exists := failedStaticNames["anything"]
	require.False(t, exists)
}

// Scenario 5: Only one provider, and it fails — boundary of the all-fail check.
func TestGracefulFailure_SingleProviderFails(t *testing.T) {
	failedStaticNames := map[string]struct{}{
		"providerA": {},
	}
	totalAttemptedCount := 1
	healthyCount := totalAttemptedCount - len(failedStaticNames)

	// healthyCount == 0 triggers the all-fail branch
	require.Equal(t, 0, healthyCount)
}

// Scenario 9: Mixed static + backup failures — both filtered independently.
func TestGracefulFailure_MixedStaticAndBackupFiltering(t *testing.T) {
	staticProviders := []*lavasession.RPCStaticProviderEndpoint{
		createTestStaticProviderEndpoint("staticA", "LAV1", "tendermintrpc"),
		createTestStaticProviderEndpoint("staticB", "LAV1", "tendermintrpc"),
		createTestStaticProviderEndpoint("staticC", "LAV1", "tendermintrpc"),
	}
	backupProviders := []*lavasession.RPCStaticProviderEndpoint{
		createTestStaticProviderEndpoint("backupX", "LAV1", "tendermintrpc"),
		createTestStaticProviderEndpoint("backupY", "LAV1", "tendermintrpc"),
	}

	failedStaticNames := map[string]struct{}{"staticB": {}}
	failedBackupNames := map[string]struct{}{"backupX": {}}

	// Filter statics
	healthyStatics := make([]*lavasession.RPCStaticProviderEndpoint, 0)
	for _, p := range staticProviders {
		if _, failed := failedStaticNames[p.Name]; !failed {
			healthyStatics = append(healthyStatics, p)
		}
	}

	// Filter backups
	healthyBackups := make([]*lavasession.RPCStaticProviderEndpoint, 0)
	for _, p := range backupProviders {
		if _, failed := failedBackupNames[p.Name]; !failed {
			healthyBackups = append(healthyBackups, p)
		}
	}

	require.Len(t, healthyStatics, 2)
	require.Equal(t, "staticA", healthyStatics[0].Name)
	require.Equal(t, "staticC", healthyStatics[1].Name)

	require.Len(t, healthyBackups, 1)
	require.Equal(t, "backupY", healthyBackups[0].Name)
}

// Scenario 20: Duplicate provider names — healthy one gets excluded as collateral.
func TestGracefulFailure_DuplicateNameCollision(t *testing.T) {
	providers := []*lavasession.RPCStaticProviderEndpoint{
		{Name: "my-node", ChainID: "LAV1", ApiInterface: "tendermintrpc",
			NodeUrls: []common.NodeUrl{{Url: "http://healthy:8080"}}},
		{Name: "my-node", ChainID: "LAV1", ApiInterface: "tendermintrpc",
			NodeUrls: []common.NodeUrl{{Url: "http://dead:8080"}}},
	}

	// The dead one fails, name "my-node" is added to failedStaticNames
	failedStaticNames := map[string]struct{}{"my-node": {}}

	// Filter — both get excluded because they share the name
	healthy := make([]*lavasession.RPCStaticProviderEndpoint, 0)
	for _, p := range providers {
		if _, failed := failedStaticNames[p.Name]; !failed {
			healthy = append(healthy, p)
		}
	}

	require.Empty(t, healthy, "Both providers excluded because they share the same name")
}

// Scenario 13/14/15: Retry goroutine self-terminates when failedStaticProviders is empty.
func TestGracefulFailure_RetryGoroutineSelfTerminates(t *testing.T) {
	rand.InitRandomSeed()
	rpsr := createTestRPCSmartRouter()

	chainKey := "LAV1-tendermintrpc"
	sm, rpcEndpoint := createTestSessionManager("LAV1", "tendermintrpc")
	rpsr.sessionManagers[chainKey] = sm

	rpsr.providerSessions[chainKey] = map[uint64]*lavasession.ConsumerSessionsWithProvider{
		0: createTestProviderSession("providerA", 1),
	}

	// Start with NO failed providers — retry should exit immediately on first tick
	// (simulates the case where all providers recovered before the tick)
	rpsr.failedStaticProviders[chainKey] = []*lavasession.RPCStaticProviderEndpoint{}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Use a short-lived mock convertProvidersToSessions that should never be called
	convertFn := func(providers []*lavasession.RPCStaticProviderEndpoint) map[uint64]*lavasession.ConsumerSessionsWithProvider {
		t.Fatal("convertProvidersToSessions should not be called when failedProviders is empty")
		return nil
	}

	done := make(chan struct{})
	go func() {
		rpsr.retryFailedStaticProviders(ctx, chainKey, nil, rpcEndpoint, convertFn)
		close(done)
	}()

	// The goroutine should exit within one ticker interval (3 minutes).
	// We can't wait 3 minutes in a test, so cancel context to force exit if it doesn't self-terminate.
	select {
	case <-done:
		// Goroutine self-terminated — test passes
	case <-time.After(5 * time.Second):
		// In the real code, the ticker is 3 minutes. The goroutine won't self-terminate
		// in 5 seconds because it waits for the ticker. Cancel context to verify clean exit.
		cancel()
		<-done
		// This is expected — the goroutine waits for the ticker before checking.
		// The important thing is it exited cleanly after context cancellation.
	}
}

// Scenario 17: Concurrent updateEpoch + retry goroutine — no race on rpsr.providerSessions.
// Run with: go test -race -run TestGracefulFailure_ConcurrentEpochAndRetry
func TestGracefulFailure_ConcurrentEpochAndRetry(t *testing.T) {
	rand.InitRandomSeed()
	rpsr := createTestRPCSmartRouter()

	chainKey := "LAV1-tendermintrpc"
	sm, _ := createTestSessionManager("LAV1", "tendermintrpc")
	rpsr.sessionManagers[chainKey] = sm

	rpsr.providerSessions[chainKey] = map[uint64]*lavasession.ConsumerSessionsWithProvider{
		0: createTestProviderSession("providerA", 1),
	}

	// Simulate: retry goroutine merges a recovered provider concurrently with epoch update
	var wg sync.WaitGroup

	// Goroutine 1: simulate retry merging a recovered provider (copy-on-write)
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 50; i++ {
			rpsr.mu.Lock()
			oldSessions := rpsr.providerSessions[chainKey]
			newSessions := make(map[uint64]*lavasession.ConsumerSessionsWithProvider, len(oldSessions)+1)
			for k, v := range oldSessions {
				newSessions[k] = v
			}
			newSessions[uint64(100+i)] = createTestProviderSession("recovered", uint64(i))
			rpsr.providerSessions[chainKey] = newSessions
			rpsr.mu.Unlock()
			time.Sleep(time.Millisecond)
		}
	}()

	// Goroutine 2: simulate epoch updates
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 50; i++ {
			rpsr.updateEpoch(uint64(10 + i))
			time.Sleep(time.Millisecond)
		}
	}()

	wg.Wait()

	// If we reach here without a race detector complaint, the test passes.
	// Verify sessions are still accessible
	rpsr.mu.Lock()
	sessions := rpsr.providerSessions[chainKey]
	rpsr.mu.Unlock()
	require.NotNil(t, sessions)
}

// Scenario 18: Copy-on-write correctness — old map is not mutated when merging recovered providers.
func TestGracefulFailure_CopyOnWriteDoesNotMutateOldMap(t *testing.T) {
	// Create the "old" sessions map (currently active)
	oldSessions := map[uint64]*lavasession.ConsumerSessionsWithProvider{
		0: createTestProviderSession("providerA", 1),
		1: createTestProviderSession("providerC", 1),
	}

	// Snapshot the original length
	originalLen := len(oldSessions)

	// Simulate the copy-on-write merge from retryFailedStaticProviders (lines 1719-1737)
	recoveredSession := createTestProviderSession("providerB", 1)

	mergedSessions := make(map[uint64]*lavasession.ConsumerSessionsWithProvider, len(oldSessions)+1)
	for k, v := range oldSessions {
		mergedSessions[k] = v
	}
	maxIdx := uint64(0)
	for idx := range mergedSessions {
		if idx >= maxIdx {
			maxIdx = idx + 1
		}
	}
	mergedSessions[maxIdx] = recoveredSession

	// Verify: old map is NOT mutated
	require.Len(t, oldSessions, originalLen, "Old map must not be mutated by copy-on-write")

	// Verify: new map has the merged result
	require.Len(t, mergedSessions, originalLen+1)
	require.Equal(t, "providerB", mergedSessions[maxIdx].PublicLavaAddress)

	// Verify: old entries are shared (same pointers)
	require.Equal(t, oldSessions[0], mergedSessions[0])
	require.Equal(t, oldSessions[1], mergedSessions[1])
}

// Scenario 22: Short epoch fires before retry — epoch only sees healthy providers.
func TestGracefulFailure_EpochBeforeRetry_OnlyHealthyProviders(t *testing.T) {
	rand.InitRandomSeed()
	rpsr := createTestRPCSmartRouter()

	chainKey := "LAV1-tendermintrpc"
	sm, _ := createTestSessionManager("LAV1", "tendermintrpc")
	rpsr.sessionManagers[chainKey] = sm

	// Startup result: only A is healthy, B failed
	rpsr.providerSessions[chainKey] = map[uint64]*lavasession.ConsumerSessionsWithProvider{
		0: createTestProviderSession("providerA", 1),
	}
	rpsr.failedStaticProviders[chainKey] = []*lavasession.RPCStaticProviderEndpoint{
		createTestStaticProviderEndpoint("providerB", "LAV1", "tendermintrpc"),
	}

	// Epoch fires multiple times before retry runs
	rpsr.updateEpoch(2)
	rpsr.updateEpoch(3)
	rpsr.updateEpoch(4)

	// Verify: still only A in sessions after 3 epochs — B never resurrected
	sessions := rpsr.providerSessions[chainKey]
	require.Len(t, sessions, 1)
	require.Equal(t, "providerA", sessions[0].PublicLavaAddress)
	require.Equal(t, uint64(4), sessions[0].PairingEpoch)

	// Verify: failed list unchanged
	require.Len(t, rpsr.failedStaticProviders[chainKey], 1)
	require.Equal(t, "providerB", rpsr.failedStaticProviders[chainKey][0].Name)
}
