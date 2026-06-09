package metrics

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/require"
)

func newConsumerForCacheTest() *ConsumerMetricsManager {
	cacheLabels := []string{"spec", "apiInterface", "method"}
	return &ConsumerMetricsManager{
		cacheRequestsTotalMetric: prometheus.NewCounterVec(prometheus.CounterOpts{Name: "t_c_cache_req"}, cacheLabels),
		cacheSuccessTotalMetric:  prometheus.NewCounterVec(prometheus.CounterOpts{Name: "t_c_cache_success"}, cacheLabels),
		cacheFailedTotalMetric:   prometheus.NewCounterVec(prometheus.CounterOpts{Name: "t_c_cache_failed"}, cacheLabels),
		cacheLatencyHistogram:    prometheus.NewHistogramVec(prometheus.HistogramOpts{Name: "t_c_cache_latency"}, cacheLabels),
	}
}

// ---- Consumer tests ----

func TestConsumerRecordCacheResult_Hit(t *testing.T) {
	cmm := newConsumerForCacheTest()
	labels := []string{"ETH1", "jsonrpc", "eth_blockNumber"}

	cmm.RecordCacheResult("ETH1", "jsonrpc", "eth_blockNumber", true, 5.0)

	require.Equal(t, float64(1), testutil.ToFloat64(cmm.cacheRequestsTotalMetric.WithLabelValues(labels...)))
	require.Equal(t, float64(1), testutil.ToFloat64(cmm.cacheSuccessTotalMetric.WithLabelValues(labels...)))
	require.Equal(t, float64(0), testutil.ToFloat64(cmm.cacheFailedTotalMetric.WithLabelValues(labels...)))
}

func TestConsumerRecordCacheResult_Miss(t *testing.T) {
	cmm := newConsumerForCacheTest()
	labels := []string{"ETH1", "jsonrpc", "eth_blockNumber"}

	cmm.RecordCacheResult("ETH1", "jsonrpc", "eth_blockNumber", false, 3.0)

	require.Equal(t, float64(1), testutil.ToFloat64(cmm.cacheRequestsTotalMetric.WithLabelValues(labels...)))
	require.Equal(t, float64(0), testutil.ToFloat64(cmm.cacheSuccessTotalMetric.WithLabelValues(labels...)))
	require.Equal(t, float64(1), testutil.ToFloat64(cmm.cacheFailedTotalMetric.WithLabelValues(labels...)))
}

func TestConsumerRecordCacheResult_TotalEqualsSuccessPlusFailedAfterMixed(t *testing.T) {
	cmm := newConsumerForCacheTest()
	labels := []string{"ETH1", "jsonrpc", "eth_blockNumber"}

	cmm.RecordCacheResult("ETH1", "jsonrpc", "eth_blockNumber", true, 5.0)
	cmm.RecordCacheResult("ETH1", "jsonrpc", "eth_blockNumber", true, 4.0)
	cmm.RecordCacheResult("ETH1", "jsonrpc", "eth_blockNumber", false, 2.0)

	total := testutil.ToFloat64(cmm.cacheRequestsTotalMetric.WithLabelValues(labels...))
	success := testutil.ToFloat64(cmm.cacheSuccessTotalMetric.WithLabelValues(labels...))
	failed := testutil.ToFloat64(cmm.cacheFailedTotalMetric.WithLabelValues(labels...))

	require.Equal(t, float64(3), total)
	require.Equal(t, float64(2), success)
	require.Equal(t, float64(1), failed)
	require.Equal(t, total, success+failed)
}

func TestConsumerRecordCacheResult_LatencyObservedOnHitOnly(t *testing.T) {
	cmm := newConsumerForCacheTest()

	// Hit — should be observed in the histogram.
	cmm.RecordCacheResult("ETH1", "jsonrpc", "eth_blockNumber", true, 10.0)
	// Miss — should NOT be observed (different latency population).
	cmm.RecordCacheResult("ETH1", "jsonrpc", "eth_blockNumber", false, 20.0)

	// Exactly one time-series should be populated (the hit observation).
	require.Equal(t, 1, testutil.CollectAndCount(cmm.cacheLatencyHistogram))
}

func TestConsumerRecordCacheResult_NilManager(t *testing.T) {
	var cmm *ConsumerMetricsManager
	require.NotPanics(t, func() {
		cmm.RecordCacheResult("ETH1", "jsonrpc", "eth_blockNumber", true, 5.0)
	})
}
