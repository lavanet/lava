package metrics

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/require"
)

var hedgeLabels = []string{"spec", "apiInterface", "method"}

func newConsumerForHedgeTest() *ConsumerMetricsManager {
	return &ConsumerMetricsManager{
		incidentHedgeTotalMetric:       prometheus.NewCounterVec(prometheus.CounterOpts{Name: "t_c_hedge_total"}, hedgeLabels),
		incidentHedgeSuccessMetric:     prometheus.NewCounterVec(prometheus.CounterOpts{Name: "t_c_hedge_success"}, hedgeLabels),
		incidentHedgeFailedMetric:      prometheus.NewCounterVec(prometheus.CounterOpts{Name: "t_c_hedge_failed"}, hedgeLabels),
		incidentHedgeAttemptsHistogram: prometheus.NewHistogramVec(prometheus.HistogramOpts{Name: "t_c_hedge_attempts", Buckets: []float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}}, hedgeLabels),
	}
}

// ---- Consumer tests ----

func TestConsumerRecordIncidentHedgeResult_Success(t *testing.T) {
	cmm := newConsumerForHedgeTest()
	labels := []string{"ETH1", "jsonrpc", "eth_blockNumber"}

	cmm.RecordIncidentHedgeResult("ETH1", "jsonrpc", "eth_blockNumber", 2, true)

	require.Equal(t, float64(1), testutil.ToFloat64(cmm.incidentHedgeTotalMetric.WithLabelValues(labels...)))
	require.Equal(t, float64(1), testutil.ToFloat64(cmm.incidentHedgeSuccessMetric.WithLabelValues(labels...)))
	require.Equal(t, float64(0), testutil.ToFloat64(cmm.incidentHedgeFailedMetric.WithLabelValues(labels...)))
}

func TestConsumerRecordIncidentHedgeResult_Failure(t *testing.T) {
	cmm := newConsumerForHedgeTest()
	labels := []string{"ETH1", "jsonrpc", "eth_blockNumber"}

	cmm.RecordIncidentHedgeResult("ETH1", "jsonrpc", "eth_blockNumber", 3, false)

	require.Equal(t, float64(1), testutil.ToFloat64(cmm.incidentHedgeTotalMetric.WithLabelValues(labels...)))
	require.Equal(t, float64(0), testutil.ToFloat64(cmm.incidentHedgeSuccessMetric.WithLabelValues(labels...)))
	require.Equal(t, float64(1), testutil.ToFloat64(cmm.incidentHedgeFailedMetric.WithLabelValues(labels...)))
}

func TestConsumerRecordIncidentHedgeResult_TotalAlwaysIncByOne(t *testing.T) {
	cmm := newConsumerForHedgeTest()
	labels := []string{"ETH1", "jsonrpc", "eth_blockNumber"}

	cmm.RecordIncidentHedgeResult("ETH1", "jsonrpc", "eth_blockNumber", 5, true)
	cmm.RecordIncidentHedgeResult("ETH1", "jsonrpc", "eth_blockNumber", 10, false)

	require.Equal(t, float64(2), testutil.ToFloat64(cmm.incidentHedgeTotalMetric.WithLabelValues(labels...)))
}

func TestConsumerRecordIncidentHedgeResult_TotalEqualsSuccessPlusFailed(t *testing.T) {
	cmm := newConsumerForHedgeTest()
	labels := []string{"ETH1", "jsonrpc", "eth_blockNumber"}

	cmm.RecordIncidentHedgeResult("ETH1", "jsonrpc", "eth_blockNumber", 2, true)
	cmm.RecordIncidentHedgeResult("ETH1", "jsonrpc", "eth_blockNumber", 1, true)
	cmm.RecordIncidentHedgeResult("ETH1", "jsonrpc", "eth_blockNumber", 4, false)

	total := testutil.ToFloat64(cmm.incidentHedgeTotalMetric.WithLabelValues(labels...))
	success := testutil.ToFloat64(cmm.incidentHedgeSuccessMetric.WithLabelValues(labels...))
	failed := testutil.ToFloat64(cmm.incidentHedgeFailedMetric.WithLabelValues(labels...))

	require.Equal(t, float64(3), total)
	require.Equal(t, float64(2), success)
	require.Equal(t, float64(1), failed)
	require.Equal(t, total, success+failed)
}

func TestConsumerRecordIncidentHedgeResult_AttemptsHistogramObserved(t *testing.T) {
	cmm := newConsumerForHedgeTest()

	require.NotPanics(t, func() {
		cmm.RecordIncidentHedgeResult("ETH1", "jsonrpc", "eth_blockNumber", 3, true)
		cmm.RecordIncidentHedgeResult("ETH1", "jsonrpc", "eth_blockNumber", 7, false)
	})
	require.Equal(t, 1, testutil.CollectAndCount(cmm.incidentHedgeAttemptsHistogram))
}

func TestConsumerRecordIncidentHedgeResult_NilManager(t *testing.T) {
	var cmm *ConsumerMetricsManager
	require.NotPanics(t, func() {
		cmm.RecordIncidentHedgeResult("ETH1", "jsonrpc", "eth_blockNumber", 2, true)
	})
}
