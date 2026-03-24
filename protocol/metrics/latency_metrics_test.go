package metrics

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
)

var (
	endToEndLatencyLabels = []string{"spec", "apiInterface", "method"}
	providerLatencyLabels = []string{"spec", "apiInterface", "provider_address", "method"}
)

func newConsumerForLatencyTest() *ConsumerMetricsManager {
	return &ConsumerMetricsManager{
		latencyEndToEndHistogram: prometheus.NewHistogramVec(prometheus.HistogramOpts{Name: "t_c_e2e_latency", Buckets: prometheus.DefBuckets}, endToEndLatencyLabels),
		latencyProviderHistogram: prometheus.NewHistogramVec(prometheus.HistogramOpts{Name: "t_c_provider_latency", Buckets: prometheus.DefBuckets}, providerLatencyLabels),
	}
}

func TestConsumerRecordEndToEndLatency_Observed(t *testing.T) {
	cmm := newConsumerForLatencyTest()

	cmm.RecordEndToEndLatency("ETH1", "jsonrpc", "eth_blockNumber", 50.0)

	reg := prometheus.NewRegistry()
	require.NoError(t, reg.Register(cmm.latencyEndToEndHistogram))
	families, err := reg.Gather()
	require.NoError(t, err)
	require.Len(t, families, 1)
	m := families[0].GetMetric()
	require.Len(t, m, 1)
	require.Equal(t, uint64(1), m[0].GetHistogram().GetSampleCount())
	require.InDelta(t, 50.0, m[0].GetHistogram().GetSampleSum(), 0.001)
}

func TestConsumerRecordEndToEndLatency_MultipleObservations(t *testing.T) {
	cmm := newConsumerForLatencyTest()

	cmm.RecordEndToEndLatency("ETH1", "jsonrpc", "eth_blockNumber", 10.0)
	cmm.RecordEndToEndLatency("ETH1", "jsonrpc", "eth_blockNumber", 20.0)
	cmm.RecordEndToEndLatency("ETH1", "jsonrpc", "eth_blockNumber", 30.0)

	reg := prometheus.NewRegistry()
	require.NoError(t, reg.Register(cmm.latencyEndToEndHistogram))
	families, err := reg.Gather()
	require.NoError(t, err)
	m := families[0].GetMetric()
	require.Len(t, m, 1)
	require.Equal(t, uint64(3), m[0].GetHistogram().GetSampleCount())
	require.InDelta(t, 60.0, m[0].GetHistogram().GetSampleSum(), 0.001)
}

func TestConsumerRecordEndToEndLatency_NilManager(t *testing.T) {
	var cmm *ConsumerMetricsManager
	require.NotPanics(t, func() {
		cmm.RecordEndToEndLatency("ETH1", "jsonrpc", "eth_blockNumber", 50.0)
	})
}

func TestConsumerRecordProviderLatency_Observed(t *testing.T) {
	cmm := newConsumerForLatencyTest()

	cmm.RecordProviderLatency("ETH1", "jsonrpc", "provider1", "eth_blockNumber", 25.0)

	reg := prometheus.NewRegistry()
	require.NoError(t, reg.Register(cmm.latencyProviderHistogram))
	families, err := reg.Gather()
	require.NoError(t, err)
	require.Len(t, families, 1)
	m := families[0].GetMetric()
	require.Len(t, m, 1)
	require.Equal(t, uint64(1), m[0].GetHistogram().GetSampleCount())
	require.InDelta(t, 25.0, m[0].GetHistogram().GetSampleSum(), 0.001)
}

func TestConsumerRecordProviderLatency_NilManager(t *testing.T) {
	var cmm *ConsumerMetricsManager
	require.NotPanics(t, func() {
		cmm.RecordProviderLatency("ETH1", "jsonrpc", "provider1", "eth_blockNumber", 25.0)
	})
}
