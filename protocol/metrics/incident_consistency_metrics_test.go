package metrics

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/require"
)

var consistencyLabels = []string{"spec", "apiInterface", "method"}

func newConsumerForConsistencyTest() *ConsumerMetricsManager {
	return &ConsumerMetricsManager{
		incidentConsistencyTotalMetric:   prometheus.NewCounterVec(prometheus.CounterOpts{Name: "t_c_consistency_total"}, consistencyLabels),
		incidentConsistencySuccessMetric: prometheus.NewCounterVec(prometheus.CounterOpts{Name: "t_c_consistency_success"}, consistencyLabels),
		incidentConsistencyFailedMetric:  prometheus.NewCounterVec(prometheus.CounterOpts{Name: "t_c_consistency_failed"}, consistencyLabels),
	}
}

func newSmartRouterForConsistencyTest() *SmartRouterMetricsManager {
	return &SmartRouterMetricsManager{
		incidentConsistencyTotalMetric:   prometheus.NewCounterVec(prometheus.CounterOpts{Name: "t_sr_consistency_total"}, consistencyLabels),
		incidentConsistencySuccessMetric: prometheus.NewCounterVec(prometheus.CounterOpts{Name: "t_sr_consistency_success"}, consistencyLabels),
		incidentConsistencyFailedMetric:  prometheus.NewCounterVec(prometheus.CounterOpts{Name: "t_sr_consistency_failed"}, consistencyLabels),
		urlToProviderName:                make(map[string]string),
	}
}

// ---- Consumer tests ----

func TestConsumerRecordIncidentConsistency_Success(t *testing.T) {
	cmm := newConsumerForConsistencyTest()
	labels := []string{"ETH1", "jsonrpc", "eth_blockNumber"}

	cmm.RecordIncidentConsistency("ETH1", "jsonrpc", "eth_blockNumber", true)

	require.Equal(t, float64(1), testutil.ToFloat64(cmm.incidentConsistencyTotalMetric.WithLabelValues(labels...)))
	require.Equal(t, float64(1), testutil.ToFloat64(cmm.incidentConsistencySuccessMetric.WithLabelValues(labels...)))
	require.Equal(t, float64(0), testutil.ToFloat64(cmm.incidentConsistencyFailedMetric.WithLabelValues(labels...)))
}

func TestConsumerRecordIncidentConsistency_Failure(t *testing.T) {
	cmm := newConsumerForConsistencyTest()
	labels := []string{"ETH1", "jsonrpc", "eth_blockNumber"}

	cmm.RecordIncidentConsistency("ETH1", "jsonrpc", "eth_blockNumber", false)

	require.Equal(t, float64(1), testutil.ToFloat64(cmm.incidentConsistencyTotalMetric.WithLabelValues(labels...)))
	require.Equal(t, float64(0), testutil.ToFloat64(cmm.incidentConsistencySuccessMetric.WithLabelValues(labels...)))
	require.Equal(t, float64(1), testutil.ToFloat64(cmm.incidentConsistencyFailedMetric.WithLabelValues(labels...)))
}

func TestConsumerRecordIncidentConsistency_TotalEqualsSuccessPlusFailed(t *testing.T) {
	cmm := newConsumerForConsistencyTest()
	labels := []string{"ETH1", "jsonrpc", "eth_blockNumber"}

	cmm.RecordIncidentConsistency("ETH1", "jsonrpc", "eth_blockNumber", true)
	cmm.RecordIncidentConsistency("ETH1", "jsonrpc", "eth_blockNumber", true)
	cmm.RecordIncidentConsistency("ETH1", "jsonrpc", "eth_blockNumber", false)

	total := testutil.ToFloat64(cmm.incidentConsistencyTotalMetric.WithLabelValues(labels...))
	success := testutil.ToFloat64(cmm.incidentConsistencySuccessMetric.WithLabelValues(labels...))
	failed := testutil.ToFloat64(cmm.incidentConsistencyFailedMetric.WithLabelValues(labels...))

	require.Equal(t, float64(3), total)
	require.Equal(t, float64(2), success)
	require.Equal(t, float64(1), failed)
	require.Equal(t, total, success+failed)
}

func TestConsumerRecordIncidentConsistency_NilManager(t *testing.T) {
	var cmm *ConsumerMetricsManager
	require.NotPanics(t, func() {
		cmm.RecordIncidentConsistency("ETH1", "jsonrpc", "eth_blockNumber", true)
	})
}

// ---- SmartRouter tests ----

func TestSmartRouterRecordIncidentConsistency_Success(t *testing.T) {
	m := newSmartRouterForConsistencyTest()
	labels := []string{"ETH1", "jsonrpc", "eth_blockNumber"}

	m.RecordIncidentConsistency("ETH1", "jsonrpc", "eth_blockNumber", true)

	require.Equal(t, float64(1), testutil.ToFloat64(m.incidentConsistencyTotalMetric.WithLabelValues(labels...)))
	require.Equal(t, float64(1), testutil.ToFloat64(m.incidentConsistencySuccessMetric.WithLabelValues(labels...)))
	require.Equal(t, float64(0), testutil.ToFloat64(m.incidentConsistencyFailedMetric.WithLabelValues(labels...)))
}

func TestSmartRouterRecordIncidentConsistency_Failure(t *testing.T) {
	m := newSmartRouterForConsistencyTest()
	labels := []string{"ETH1", "jsonrpc", "eth_blockNumber"}

	m.RecordIncidentConsistency("ETH1", "jsonrpc", "eth_blockNumber", false)

	require.Equal(t, float64(1), testutil.ToFloat64(m.incidentConsistencyTotalMetric.WithLabelValues(labels...)))
	require.Equal(t, float64(0), testutil.ToFloat64(m.incidentConsistencySuccessMetric.WithLabelValues(labels...)))
	require.Equal(t, float64(1), testutil.ToFloat64(m.incidentConsistencyFailedMetric.WithLabelValues(labels...)))
}

func TestSmartRouterRecordIncidentConsistency_TotalEqualsSuccessPlusFailed(t *testing.T) {
	m := newSmartRouterForConsistencyTest()
	labels := []string{"ETH1", "jsonrpc", "eth_blockNumber"}

	m.RecordIncidentConsistency("ETH1", "jsonrpc", "eth_blockNumber", true)
	m.RecordIncidentConsistency("ETH1", "jsonrpc", "eth_blockNumber", true)
	m.RecordIncidentConsistency("ETH1", "jsonrpc", "eth_blockNumber", false)

	total := testutil.ToFloat64(m.incidentConsistencyTotalMetric.WithLabelValues(labels...))
	success := testutil.ToFloat64(m.incidentConsistencySuccessMetric.WithLabelValues(labels...))
	failed := testutil.ToFloat64(m.incidentConsistencyFailedMetric.WithLabelValues(labels...))

	require.Equal(t, float64(3), total)
	require.Equal(t, float64(2), success)
	require.Equal(t, float64(1), failed)
	require.Equal(t, total, success+failed)
}

func TestSmartRouterRecordIncidentConsistency_NilManager(t *testing.T) {
	var m *SmartRouterMetricsManager
	require.NotPanics(t, func() {
		m.RecordIncidentConsistency("ETH1", "jsonrpc", "eth_blockNumber", true)
	})
}
