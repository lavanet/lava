package metrics

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSanitizeFloat(t *testing.T) {
	assert.Equal(t, 0.0, sanitizeFloat(math.NaN()))
	assert.Equal(t, 0.0, sanitizeFloat(math.Inf(1)))
	assert.Equal(t, 0.0, sanitizeFloat(math.Inf(-1)))
	assert.Equal(t, 0.42, sanitizeFloat(0.42))
	assert.Equal(t, 0.0, sanitizeFloat(0.0))
}

// fakeOptimizer returns a pre-canned list of QoS reports on every call,
// regardless of the addresses the sampler passes in.
type fakeOptimizer struct {
	reports []*OptimizerQoSReport
}

func (f *fakeOptimizer) CalculateQoSScoresForMetrics(_ []string, _ map[string]struct{}) []*OptimizerQoSReport {
	return f.reports
}

// TestSampleAndEmit_EmitsPerChainProvider exercises the full sampler:
//   - one optimizer registered for a chain
//   - a stake map fed in via UpdatePairingListStake (also sets epoch)
//   - sampleAndEmit asks the optimizer for reports, builds one
//     OptimizerQoSReportToSend per (chain, provider), and pushes each
//     through the sink. The latest batch is cached for the Prometheus
//     scrape path.
func TestSampleAndEmit_EmitsPerChainProvider(t *testing.T) {
	sink := &captureSink{}
	client := NewConsumerOptimizerQoSClient("consumer-pub", sink, 7)

	const chainID = "ETH1"
	client.RegisterOptimizer(&fakeOptimizer{
		reports: []*OptimizerQoSReport{
			{
				ProviderAddress:          "lava@a",
				EntryIndex:               0,
				SelectionAvailability:    0.95,
				SelectionLatency:         0.80,
				SelectionSync:            0.70,
				SelectionStake:           0.60,
				SelectionComposite:       0.85,
				AvailabilityContribution: 0.38,
				LatencyContribution:      0.24,
				SyncContribution:         0.14,
				StakeContribution:        0.06,
			},
			{
				ProviderAddress:    "lava@b",
				EntryIndex:         1,
				SelectionComposite: 0.50,
			},
		},
	}, chainID)
	client.UpdatePairingListStake(map[string]int64{
		"lava@a": 1_000,
		"lava@b": 2_000,
	}, chainID, 42)

	client.sampleAndEmit()

	require.Len(t, sink.qosEvents, 2, "one record per (chain, provider) in the stake map")
	assert.Empty(t, sink.events, "QoS path must not produce relay_usage events")

	byProvider := map[string]OptimizerQoSReportToSend{}
	for _, ev := range sink.qosEvents {
		byProvider[ev.ProviderAddress] = ev
	}

	a := byProvider["lava@a"]
	assert.Equal(t, chainID, a.ChainId)
	assert.Equal(t, "consumer-pub", a.ConsumerAddress)
	assert.Equal(t, uint64(7), a.GeoLocation)
	assert.Equal(t, uint64(42), a.Epoch)
	assert.Equal(t, int64(1_000), a.ProviderStake, "provider_stake comes from the pairing-list snapshot")
	assert.Equal(t, 0.85, a.SelectionComposite)
	assert.Equal(t, 0.38, a.AvailabilityContribution)

	b := byProvider["lava@b"]
	assert.Equal(t, int64(2_000), b.ProviderStake)
	assert.Equal(t, 0.50, b.SelectionComposite)

	cached := client.GetReportsToSend()
	require.Len(t, cached, 2, "latest batch is cached for the scrape endpoint")
}

func TestSampleAndEmit_SkipsNilReports(t *testing.T) {
	sink := &captureSink{}
	client := NewConsumerOptimizerQoSClient("c", sink, 1)

	const chainID = "ETH1"
	client.RegisterOptimizer(&fakeOptimizer{
		reports: []*OptimizerQoSReport{
			nil, // optimizer returns a nil entry; sampler must skip
			{ProviderAddress: "lava@real", SelectionComposite: 0.7},
			nil,
		},
	}, chainID)
	client.UpdatePairingListStake(map[string]int64{"lava@real": 500}, chainID, 1)

	client.sampleAndEmit()

	require.Len(t, sink.qosEvents, 1)
	assert.Equal(t, "lava@real", sink.qosEvents[0].ProviderAddress)
	assert.Equal(t, int64(500), sink.qosEvents[0].ProviderStake)

	cached := client.GetReportsToSend()
	require.Len(t, cached, 1, "nil reports must not leak into the scrape cache")
}

// TestSampleAndEmit_NoStakeNoEmit covers the branch where a chain has a
// registered optimizer but UpdatePairingListStake hasn't fired yet
// (empty stake map for that chain). The sampler must not call the
// optimizer at all for that chain.
func TestSampleAndEmit_NoStakeNoEmit(t *testing.T) {
	sink := &captureSink{}
	client := NewConsumerOptimizerQoSClient("c", sink, 1)

	client.RegisterOptimizer(&fakeOptimizer{
		reports: []*OptimizerQoSReport{{ProviderAddress: "lava@a", SelectionComposite: 0.9}},
	}, "ETH1")
	// no UpdatePairingListStake for ETH1

	client.sampleAndEmit()

	assert.Empty(t, sink.qosEvents)
	assert.Empty(t, client.GetReportsToSend())
}

func TestRegisterOptimizer_IgnoresDuplicate(t *testing.T) {
	client := NewConsumerOptimizerQoSClient("c", &captureSink{}, 1)

	first := &fakeOptimizer{reports: []*OptimizerQoSReport{{ProviderAddress: "x", SelectionComposite: 0.1}}}
	second := &fakeOptimizer{reports: []*OptimizerQoSReport{{ProviderAddress: "y", SelectionComposite: 0.9}}}

	client.RegisterOptimizer(first, "ETH1")
	client.RegisterOptimizer(second, "ETH1") // duplicate — must be ignored

	client.UpdatePairingListStake(map[string]int64{"x": 1, "y": 1}, "ETH1", 1)
	client.sampleAndEmit()

	cached := client.GetReportsToSend()
	require.Len(t, cached, 1)
	assert.Equal(t, "x", cached[0].ProviderAddress, "first registration wins")
}
