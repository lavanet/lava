package rpcsmartrouter

import (
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/v5/protocol/lavasession"
	"github.com/lavanet/lava/v5/protocol/metrics"
	"github.com/lavanet/lava/v5/protocol/provideroptimizer"
	"github.com/lavanet/lava/v5/utils/rand"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
)

// gatherHealthGauge reads the lava_rpc_endpoint_overall_health gauge directly from the
// default Prometheus gatherer for a specific (spec, apiInterface, endpoint_id) tuple.
// Returns (value, true) if found, (0, false) otherwise.
func gatherHealthGauge(t *testing.T, spec, apiInterface, endpointID string) (float64, bool) {
	t.Helper()
	families, err := prometheus.DefaultGatherer.Gather()
	require.NoError(t, err)
	for _, mf := range families {
		if mf.GetName() != "lava_rpc_endpoint_overall_health" {
			continue
		}
		for _, m := range mf.GetMetric() {
			labels := map[string]string{}
			for _, lp := range m.GetLabel() {
				labels[lp.GetName()] = lp.GetValue()
			}
			if labels["spec"] == spec && labels["apiInterface"] == apiInterface && labels["endpoint_id"] == endpointID {
				return m.GetGauge().GetValue(), true
			}
		}
	}
	return 0, false
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

// TestUpdateEpoch_ResetsHealthMetric is the companion to the struct-level reset above:
// it verifies that updateEpoch also resets the Prometheus lava_rpc_endpoint_overall_health
// gauge back to 1 for both primary and backup providers. Prior to this fix, #2256 reset
// the in-memory endpoint struct but left the metric stuck at 0, so operators saw 0% uptime
// on backups even after the router considered them healthy again.
func TestUpdateEpoch_ResetsHealthMetric(t *testing.T) {
	rand.InitRandomSeed()

	// Use unique chain/apiInterface labels per test run so we don't collide with
	// metric values set by other tests sharing the process-global Prometheus registry.
	const (
		testChainID      = "LAV1_METRIC_RESET_TEST"
		testApiInterface = "tendermintrpc"
		primaryProvider  = "lava@primary-metric-test"
		backupProvider   = "lava@backup-metric-test"
	)

	rpsr := &RPCSmartRouter{
		sessionManagers:        make(map[string]*lavasession.ConsumerSessionManager),
		providerSessions:       make(map[string]map[uint64]*lavasession.ConsumerSessionsWithProvider),
		backupProviderSessions: make(map[string]map[uint64]*lavasession.ConsumerSessionsWithProvider),
		rpcServers:             make(map[string]*RPCSmartRouterServer),
	}

	rpcEndpoint := &lavasession.RPCEndpoint{
		ChainID:        testChainID,
		ApiInterface:   testApiInterface,
		NetworkAddress: "127.0.0.1:3335",
	}
	optimizer := provideroptimizer.NewProviderOptimizer(provideroptimizer.StrategyBalanced, time.Second, uint(1), nil, testChainID)
	chainKey := rpcEndpoint.Key()
	rpsr.sessionManagers[chainKey] = lavasession.NewConsumerSessionManager(
		rpcEndpoint, optimizer, nil, nil, "test-router", lavasession.NewActiveSubscriptionProvidersStorage(),
	)

	// Wire a real SmartRouterMetricsManager into a minimal RPCSmartRouterServer so
	// updateEpoch can find it via rpsr.rpcServers[chainKey].
	metricsManager := metrics.NewSmartRouterMetricsManager(metrics.SmartRouterMetricsManagerOptions{
		NetworkAddress:  "disabled-but-nonempty", // non-DisabledFlagOption value so the manager is created
		StartHTTPServer: false,
	})
	require.NotNil(t, metricsManager)
	rpsr.rpcServers[chainKey] = &RPCSmartRouterServer{
		listenEndpoint:             rpcEndpoint,
		smartRouterEndpointMetrics: metricsManager,
	}

	// Seed the health metric to 0 for both providers, simulating the stuck-unhealthy
	// state that triggers the bug in production (an earlier relay error marked them
	// unhealthy and no successful relay has reset the gauge since).
	metricsManager.SetEndpointOverallHealth(testChainID, testApiInterface, primaryProvider, false)
	metricsManager.SetEndpointOverallHealth(testChainID, testApiInterface, backupProvider, false)

	v, ok := gatherHealthGauge(t, testChainID, testApiInterface, primaryProvider)
	require.True(t, ok && v == 0, "precondition: primary health gauge should be 0 before updateEpoch, got ok=%v v=%v", ok, v)
	v, ok = gatherHealthGauge(t, testChainID, testApiInterface, backupProvider)
	require.True(t, ok && v == 0, "precondition: backup health gauge should be 0 before updateEpoch, got ok=%v v=%v", ok, v)

	// Disabled endpoints matching the struct-level ResetHealth path from #2256.
	disabledPrimaryEndpoint := &lavasession.Endpoint{
		NetworkAddress:     "http://primary-metric:8080",
		Enabled:            false,
		ConnectionRefusals: lavasession.MaxConsecutiveConnectionAttempts,
	}
	disabledBackupEndpoint := &lavasession.Endpoint{
		NetworkAddress:     "http://backup-metric:8080",
		Enabled:            false,
		ConnectionRefusals: lavasession.MaxConsecutiveConnectionAttempts,
	}

	primarySession := lavasession.NewConsumerSessionWithProvider(
		primaryProvider,
		[]*lavasession.Endpoint{disabledPrimaryEndpoint},
		100, uint64(1), sdk.NewCoin("ulava", sdk.NewInt(1)),
	)
	primarySession.StaticProvider = true

	backupSession := lavasession.NewConsumerSessionWithProvider(
		backupProvider,
		[]*lavasession.Endpoint{disabledBackupEndpoint},
		100, uint64(1), sdk.NewCoin("ulava", sdk.NewInt(1)),
	)
	backupSession.StaticProvider = true

	rpsr.providerSessions[chainKey] = map[uint64]*lavasession.ConsumerSessionsWithProvider{0: primarySession}
	rpsr.backupProviderSessions[chainKey] = map[uint64]*lavasession.ConsumerSessionsWithProvider{0: backupSession}

	rpsr.updateEpoch(uint64(2))

	// Struct-level reset still holds (regression guard for #2256).
	require.True(t, disabledPrimaryEndpoint.Enabled, "primary endpoint should be re-enabled")
	require.True(t, disabledBackupEndpoint.Enabled, "backup endpoint should be re-enabled")

	// The new behavior this test is specifically for: metric gauge also reset to 1.
	v, ok = gatherHealthGauge(t, testChainID, testApiInterface, primaryProvider)
	require.True(t, ok, "primary health gauge must be present after updateEpoch")
	require.Equal(t, float64(1), v, "primary health gauge must be reset to 1 (healthy) on epoch transition")

	v, ok = gatherHealthGauge(t, testChainID, testApiInterface, backupProvider)
	require.True(t, ok, "backup health gauge must be present after updateEpoch")
	require.Equal(t, float64(1), v, "backup health gauge must be reset to 1 (healthy) on epoch transition")
}

// TestUpdateEpoch_NilListenEndpointDoesNotPanic guards the nil-deref flagged in
// review of commit 555448be2: rpcsmartrouter.updateEpoch read
// server.listenEndpoint.ChainID / .ApiInterface without verifying listenEndpoint
// (a *lavasession.RPCEndpoint pointer) is non-nil. If a server is registered with
// a nil listenEndpoint, the whole epoch transition would panic and every
// endpoint.ResetHealth() in that chain would be left undone. The metric reset is
// optional — skipping it for a server with a nil listenEndpoint is preferable to
// crashing the epoch handler.
func TestUpdateEpoch_NilListenEndpointDoesNotPanic(t *testing.T) {
	rand.InitRandomSeed()

	rpcEndpoint := &lavasession.RPCEndpoint{
		ChainID:        "LAV1",
		ApiInterface:   "tendermintrpc",
		NetworkAddress: "127.0.0.1:3336",
	}
	optimizer := provideroptimizer.NewProviderOptimizer(provideroptimizer.StrategyBalanced, time.Second, uint(1), nil, rpcEndpoint.ChainID)
	chainKey := rpcEndpoint.Key()

	rpsr := &RPCSmartRouter{
		sessionManagers:        make(map[string]*lavasession.ConsumerSessionManager),
		providerSessions:       make(map[string]map[uint64]*lavasession.ConsumerSessionsWithProvider),
		backupProviderSessions: make(map[string]map[uint64]*lavasession.ConsumerSessionsWithProvider),
		rpcServers:             make(map[string]*RPCSmartRouterServer),
	}
	rpsr.sessionManagers[chainKey] = lavasession.NewConsumerSessionManager(
		rpcEndpoint, optimizer, nil, nil, "test-router", lavasession.NewActiveSubscriptionProvidersStorage(),
	)

	// Server registered with nil listenEndpoint — the scenario the guard protects against.
	rpsr.rpcServers[chainKey] = &RPCSmartRouterServer{
		listenEndpoint: nil,
	}

	disabledEndpoint := &lavasession.Endpoint{
		NetworkAddress:     "http://whatever:8080",
		Enabled:            false,
		ConnectionRefusals: lavasession.MaxConsecutiveConnectionAttempts,
	}
	session := lavasession.NewConsumerSessionWithProvider(
		"lava@provider-nil-listen",
		[]*lavasession.Endpoint{disabledEndpoint},
		100, uint64(1), sdk.NewCoin("ulava", sdk.NewInt(1)),
	)
	session.StaticProvider = true
	rpsr.providerSessions[chainKey] = map[uint64]*lavasession.ConsumerSessionsWithProvider{0: session}

	require.NotPanics(t, func() { rpsr.updateEpoch(uint64(2)) },
		"updateEpoch must tolerate a server with nil listenEndpoint rather than nil-deref during metric resolution")

	// Even without the metric reset, the in-memory struct reset (from commit 1559d6b29) must still run.
	require.True(t, disabledEndpoint.Enabled,
		"endpoint.ResetHealth() must still fire even when listenEndpoint is nil — the metric reset is optional but the struct reset is load-bearing")
}
