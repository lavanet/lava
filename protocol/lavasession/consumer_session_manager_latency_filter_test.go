package lavasession

import (
	"context"
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/v5/protocol/provideroptimizer"
	pairingtypes "github.com/lavanet/lava/v5/x/pairing/types"
	"github.com/stretchr/testify/require"
)

// stubLatencyOptimizer is a minimal ProviderOptimizer used to drive
// filterHighLatencyProviders with controlled per-provider QoS latency.
// Only GetReputationReportForProvider carries meaningful behavior; the rest
// of the interface is stubbed out as it is not exercised by the filter.
type stubLatencyOptimizer struct {
	reports map[string]*pairingtypes.QualityOfServiceReport
}

func (s *stubLatencyOptimizer) GetReputationReportForProvider(address string) (*pairingtypes.QualityOfServiceReport, time.Time) {
	return s.reports[address], time.Time{}
}

func (s *stubLatencyOptimizer) AppendProbeRelayData(providerAddress string, latency time.Duration, success bool) {
}
func (s *stubLatencyOptimizer) AppendRelayFailure(providerAddress string) {}
func (s *stubLatencyOptimizer) AppendRelayData(providerAddress string, latency time.Duration, cu, syncBlock uint64) {
}

func (s *stubLatencyOptimizer) ChooseProvider(ctx context.Context, allAddresses []string, ignoredProviders map[string]struct{}, cu uint64, requestedBlock int64) []string {
	return nil
}

func (s *stubLatencyOptimizer) ChooseProviderWithStats(ctx context.Context, allAddresses []string, ignoredProviders map[string]struct{}, cu uint64, requestedBlock int64) ([]string, *provideroptimizer.SelectionStats) {
	return nil, nil
}

func (s *stubLatencyOptimizer) ChooseBestProvider(ctx context.Context, allAddresses []string, ignoredProviders map[string]struct{}, cu uint64, requestedBlock int64) []string {
	return nil
}

func (s *stubLatencyOptimizer) ChooseBestProviderWithStats(ctx context.Context, allAddresses []string, ignoredProviders map[string]struct{}, cu uint64, requestedBlock int64) ([]string, *provideroptimizer.SelectionStats) {
	return nil, nil
}

func (s *stubLatencyOptimizer) Strategy() provideroptimizer.Strategy {
	return provideroptimizer.StrategyBalanced
}
func (s *stubLatencyOptimizer) UpdateWeights(map[string]int64, uint64) {}

// qosWithLatency builds a QoS report with the given latency (in seconds) and
// neutral availability/sync, which is all filterHighLatencyProviders reads.
func qosWithLatency(latencySeconds string) *pairingtypes.QualityOfServiceReport {
	return &pairingtypes.QualityOfServiceReport{
		Latency:      sdk.MustNewDecFromStr(latencySeconds),
		Availability: sdk.OneDec(),
		Sync:         sdk.OneDec(),
	}
}

// newLatencyFilterCSM returns a ConsumerSessionManager wired with a stub
// optimizer and the given per-chain latency cutoff.
func newLatencyFilterCSM(maxLatency float64, reports map[string]*pairingtypes.QualityOfServiceReport) *ConsumerSessionManager {
	return &ConsumerSessionManager{
		rpcEndpoint:       &RPCEndpoint{ChainID: "test", ApiInterface: "jsonrpc", MaxProviderLatency: maxLatency},
		providerOptimizer: &stubLatencyOptimizer{reports: reports},
	}
}

func TestFilterHighLatencyProviders_ExcludesSlow(t *testing.T) {
	ctx := context.Background()
	validAddresses := []string{"fast", "slow"}
	csm := newLatencyFilterCSM(1.0, map[string]*pairingtypes.QualityOfServiceReport{
		"fast": qosWithLatency("0.5"),
		"slow": qosWithLatency("1.5"),
	})

	ignored := map[string]struct{}{}
	csm.filterHighLatencyProviders(ctx, validAddresses, ignored)

	_, slowIgnored := ignored["slow"]
	_, fastIgnored := ignored["fast"]
	require.True(t, slowIgnored, "provider above cutoff should be put aside")
	require.False(t, fastIgnored, "provider below cutoff should remain selectable")
}

func TestFilterHighLatencyProviders_KeepsAllFast(t *testing.T) {
	ctx := context.Background()
	validAddresses := []string{"a", "b"}
	csm := newLatencyFilterCSM(1.0, map[string]*pairingtypes.QualityOfServiceReport{
		"a": qosWithLatency("0.2"),
		"b": qosWithLatency("0.99"),
	})

	ignored := map[string]struct{}{}
	csm.filterHighLatencyProviders(ctx, validAddresses, ignored)

	require.Empty(t, ignored, "no provider above cutoff should be filtered")
}

func TestFilterHighLatencyProviders_FallbackWhenAllSlow(t *testing.T) {
	ctx := context.Background()
	validAddresses := []string{"x", "y"}
	csm := newLatencyFilterCSM(1.0, map[string]*pairingtypes.QualityOfServiceReport{
		"x": qosWithLatency("2.0"),
		"y": qosWithLatency("3.0"),
	})

	ignored := map[string]struct{}{}
	csm.filterHighLatencyProviders(ctx, validAddresses, ignored)

	require.Empty(t, ignored, "safety fallback should keep the full pool when every provider is above the cutoff")
}

func TestFilterHighLatencyProviders_Disabled(t *testing.T) {
	ctx := context.Background()
	validAddresses := []string{"slow"}
	csm := newLatencyFilterCSM(0, map[string]*pairingtypes.QualityOfServiceReport{
		"slow": qosWithLatency("5.0"),
	})

	ignored := map[string]struct{}{}
	csm.filterHighLatencyProviders(ctx, validAddresses, ignored)

	require.Empty(t, ignored, "cutoff of 0 must disable filtering")
}

func TestFilterHighLatencyProviders_ColdStartNotFiltered(t *testing.T) {
	ctx := context.Background()
	validAddresses := []string{"known", "coldstart"}
	csm := newLatencyFilterCSM(1.0, map[string]*pairingtypes.QualityOfServiceReport{
		"known": qosWithLatency("0.5"),
		// "coldstart" has no QoS report -> GetReputationReportForProvider returns nil
	})

	ignored := map[string]struct{}{}
	csm.filterHighLatencyProviders(ctx, validAddresses, ignored)

	require.Empty(t, ignored, "providers with no QoS data must be treated as under threshold")
}

func TestFilterHighLatencyProviders_PreservesExistingIgnored(t *testing.T) {
	ctx := context.Background()
	validAddresses := []string{"fast", "slow", "alreadyIgnored"}
	csm := newLatencyFilterCSM(1.0, map[string]*pairingtypes.QualityOfServiceReport{
		"fast":           qosWithLatency("0.3"),
		"slow":           qosWithLatency("2.0"),
		"alreadyIgnored": qosWithLatency("0.1"),
	})

	ignored := map[string]struct{}{"alreadyIgnored": {}}
	csm.filterHighLatencyProviders(ctx, validAddresses, ignored)

	_, alreadyIgnored := ignored["alreadyIgnored"]
	_, slowIgnored := ignored["slow"]
	require.True(t, alreadyIgnored, "pre-existing ignored providers must remain ignored")
	require.True(t, slowIgnored, "slow provider should be added on top of existing ignored set")
	require.Len(t, ignored, 2)
}

// Test 6: latency exactly equal to the cutoff is NOT filtered (comparison is strict ">").
func TestFilterHighLatencyProviders_ThresholdBoundaryExclusiveGreater(t *testing.T) {
	ctx := context.Background()
	validAddresses := []string{"exact", "justOver"}
	csm := newLatencyFilterCSM(1.0, map[string]*pairingtypes.QualityOfServiceReport{
		"exact":    qosWithLatency("1.0"),   // == cutoff -> kept
		"justOver": qosWithLatency("1.001"), // > cutoff -> filtered
	})

	ignored := map[string]struct{}{}
	csm.filterHighLatencyProviders(ctx, validAddresses, ignored)

	_, exactIgnored := ignored["exact"]
	_, justOverIgnored := ignored["justOver"]
	require.False(t, exactIgnored, "latency equal to the cutoff must not be filtered (strict >)")
	require.True(t, justOverIgnored, "latency just above the cutoff must be filtered")
}

// Test 7: with several providers, EVERY slow one is put aside while all fast ones remain,
// so a multi-provider batch can be filled exclusively from fast providers.
func TestFilterHighLatencyProviders_MultipleProvidersFiltersAllSlow(t *testing.T) {
	ctx := context.Background()
	validAddresses := []string{"fast1", "slow1", "fast2", "slow2", "fast3"}
	csm := newLatencyFilterCSM(1.0, map[string]*pairingtypes.QualityOfServiceReport{
		"fast1": qosWithLatency("0.10"),
		"slow1": qosWithLatency("1.50"),
		"fast2": qosWithLatency("0.40"),
		"slow2": qosWithLatency("2.20"),
		"fast3": qosWithLatency("0.90"),
	})

	ignored := map[string]struct{}{}
	csm.filterHighLatencyProviders(ctx, validAddresses, ignored)

	require.Len(t, ignored, 2, "both slow providers should be filtered")
	for _, slow := range []string{"slow1", "slow2"} {
		_, ok := ignored[slow]
		require.True(t, ok, "slow provider %s should be filtered", slow)
	}
	for _, fast := range []string{"fast1", "fast2", "fast3"} {
		_, ok := ignored[fast]
		require.False(t, ok, "fast provider %s should remain selectable", fast)
	}
}

// Test 8: mixed cold-start + slow + fast. The cold-start (no QoS) and fast providers
// count as "remaining", so the fallback does NOT trigger and the slow one is filtered.
func TestFilterHighLatencyProviders_MixedColdStartSlowFast(t *testing.T) {
	ctx := context.Background()
	validAddresses := []string{"coldstart", "slow", "fast"}
	csm := newLatencyFilterCSM(1.0, map[string]*pairingtypes.QualityOfServiceReport{
		// "coldstart" has no QoS report -> treated as under threshold
		"slow": qosWithLatency("2.0"),
		"fast": qosWithLatency("0.3"),
	})

	ignored := map[string]struct{}{}
	csm.filterHighLatencyProviders(ctx, validAddresses, ignored)

	_, slowIgnored := ignored["slow"]
	_, coldIgnored := ignored["coldstart"]
	_, fastIgnored := ignored["fast"]
	require.True(t, slowIgnored, "slow provider should be filtered when fast/cold-start providers remain")
	require.False(t, coldIgnored, "cold-start provider must not be filtered")
	require.False(t, fastIgnored, "fast provider must not be filtered")
	require.Len(t, ignored, 1)
}
