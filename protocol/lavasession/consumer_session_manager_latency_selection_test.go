package lavasession

import (
	"context"
	"testing"
	"time"

	"github.com/lavanet/lava/v5/protocol/common"
	"github.com/lavanet/lava/v5/protocol/provideroptimizer"
	pairingtypes "github.com/lavanet/lava/v5/x/pairing/types"
	"github.com/stretchr/testify/require"
)

// smartStubOptimizer is a ProviderOptimizer that drives getValidProviderAddresses
// without the real weighted-selection machinery: it returns the first non-ignored
// address (so the ignored set fully controls which providers can be picked) and
// reports per-provider latency from a configurable map. Providers without an
// explicit entry default to a fast latency, so only explicitly-slow providers are
// candidates for the cutoff.
type smartStubOptimizer struct {
	reports map[string]*pairingtypes.QualityOfServiceReport
}

func newSmartStubOptimizer() *smartStubOptimizer {
	return &smartStubOptimizer{reports: map[string]*pairingtypes.QualityOfServiceReport{}}
}

func (s *smartStubOptimizer) setLatency(address, latencySeconds string) {
	s.reports[address] = qosWithLatency(latencySeconds)
}

func (s *smartStubOptimizer) GetReputationReportForProvider(address string) (*pairingtypes.QualityOfServiceReport, time.Time) {
	if r, ok := s.reports[address]; ok {
		return r, time.Time{}
	}
	// default: fast, well under any sane cutoff
	return qosWithLatency("0.05"), time.Time{}
}

func firstNotIgnored(allAddresses []string, ignoredProviders map[string]struct{}) []string {
	for _, addr := range allAddresses {
		if _, ok := ignoredProviders[addr]; ok {
			continue
		}
		return []string{addr}
	}
	return []string{}
}

func (s *smartStubOptimizer) ChooseProviderWithStats(ctx context.Context, allAddresses []string, ignoredProviders map[string]struct{}, cu uint64, requestedBlock int64) ([]string, *provideroptimizer.SelectionStats) {
	return firstNotIgnored(allAddresses, ignoredProviders), nil
}

func (s *smartStubOptimizer) ChooseBestProviderWithStats(ctx context.Context, allAddresses []string, ignoredProviders map[string]struct{}, cu uint64, requestedBlock int64) ([]string, *provideroptimizer.SelectionStats) {
	return firstNotIgnored(allAddresses, ignoredProviders), nil
}

func (s *smartStubOptimizer) ChooseProvider(ctx context.Context, allAddresses []string, ignoredProviders map[string]struct{}, cu uint64, requestedBlock int64) []string {
	return firstNotIgnored(allAddresses, ignoredProviders)
}

func (s *smartStubOptimizer) ChooseBestProvider(ctx context.Context, allAddresses []string, ignoredProviders map[string]struct{}, cu uint64, requestedBlock int64) []string {
	return firstNotIgnored(allAddresses, ignoredProviders)
}

func (s *smartStubOptimizer) Strategy() provideroptimizer.Strategy { return provideroptimizer.StrategyBalanced }
func (s *smartStubOptimizer) UpdateWeights(map[string]int64, uint64) {}
func (s *smartStubOptimizer) AppendProbeRelayData(string, time.Duration, bool) {}
func (s *smartStubOptimizer) AppendRelayFailure(string) {}
func (s *smartStubOptimizer) AppendRelayData(string, time.Duration, uint64, uint64) {}

// setupCSMForSelection builds a fully-wired ConsumerSessionManager with the smart
// stub optimizer and a real pairing list, returning the valid provider addresses.
func setupCSMForSelection(t *testing.T, maxLatency float64) (*ConsumerSessionManager, *smartStubOptimizer, []string) {
	ctx := context.Background()
	csm := CreateConsumerSessionManager()
	stub := newSmartStubOptimizer()
	csm.providerOptimizer = stub
	csm.rpcEndpoint.MaxProviderLatency = maxLatency

	pairingList := createPairingList("", true)
	require.NoError(t, csm.UpdateAllProviders(firstEpochHeight, pairingList, nil))

	csm.lock.RLock()
	valid := csm.getValidAddresses("", nil, ctx)
	csm.lock.RUnlock()
	require.GreaterOrEqual(t, len(valid), 2, "need at least two valid providers for the test")
	return csm, stub, valid
}

// callGetValidProviderAddresses runs the selection under the read lock the function expects.
func callGetValidProviderAddresses(csm *ConsumerSessionManager, ctx context.Context, wanted int, stateful uint32, stickiness, selectedProvider string) ([]string, error) {
	csm.lock.RLock()
	defer csm.lock.RUnlock()
	return csm.getValidProviderAddresses(ctx, wanted, map[string]struct{}{}, cuForFirstRequest, servicedBlockNumber, "", nil, stateful, stickiness, selectedProvider)
}

// Test 1: end-to-end general random selection excludes the slow provider and still
// fills the batch from the remaining (fast) providers.
func TestLatencySelection_GeneralPathExcludesSlowProvider(t *testing.T) {
	ctx := context.Background()
	csm, stub, valid := setupCSMForSelection(t, 1.0)
	slow := valid[0]
	stub.setLatency(slow, "1.5")

	addresses, err := callGetValidProviderAddresses(csm, ctx, len(valid), common.NO_STATE, "", "")
	require.NoError(t, err)
	require.NotEmpty(t, addresses)
	require.NotContains(t, addresses, slow, "slow provider must not be selected in the general path")
	require.Len(t, addresses, len(valid)-1, "all fast providers should remain selectable")
}

// Test 3: stickiness path is unaffected by the cutoff - a sticky session pinned to a
// slow provider still returns it.
func TestLatencySelection_StickinessUnaffected(t *testing.T) {
	ctx := context.Background()
	csm, stub, valid := setupCSMForSelection(t, 1.0)
	slow := valid[0]
	stub.setLatency(slow, "5.0")
	csm.stickySessions.Set("sticky-id", &StickySession{Provider: slow, Epoch: firstEpochHeight})

	addresses, err := callGetValidProviderAddresses(csm, ctx, 1, common.NO_STATE, "sticky-id", "")
	require.NoError(t, err)
	require.Equal(t, []string{slow}, addresses, "sticky session must return its pinned provider regardless of latency")
}

// Test 4: stateful (select-all) path is unaffected by the cutoff - the slow provider
// is still included in the returned set.
func TestLatencySelection_StatefulUnaffected(t *testing.T) {
	ctx := context.Background()
	csm, stub, valid := setupCSMForSelection(t, 1.0)
	slow := valid[0]
	stub.setLatency(slow, "5.0")

	addresses, err := callGetValidProviderAddresses(csm, ctx, len(valid), common.CONSISTENCY_SELECT_ALL_PROVIDERS, "", "")
	require.NoError(t, err)
	require.Contains(t, addresses, slow, "stateful select-all must include the slow provider")
}

// Test 5: header/static selected-provider path is unaffected by the cutoff - the
// explicitly selected slow provider is still returned.
func TestLatencySelection_SelectedProviderUnaffected(t *testing.T) {
	ctx := context.Background()
	csm, stub, valid := setupCSMForSelection(t, 1.0)
	slow := valid[0]
	stub.setLatency(slow, "5.0")

	addresses, err := callGetValidProviderAddresses(csm, ctx, 1, common.NO_STATE, "", slow)
	require.NoError(t, err)
	require.Equal(t, []string{slow}, addresses, "explicitly selected provider must be returned regardless of latency")
}
