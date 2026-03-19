package metrics

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/require"
)

var retryLabels = []string{"spec", "apiInterface", "method"}

func newConsumerForRetryTest() *ConsumerMetricsManager {
	return &ConsumerMetricsManager{
		incidentRetriesTotalMetric:     prometheus.NewCounterVec(prometheus.CounterOpts{Name: "t_c_retries_total"}, retryLabels),
		incidentRetriesSuccessMetric:   prometheus.NewCounterVec(prometheus.CounterOpts{Name: "t_c_retries_success"}, retryLabels),
		incidentRetriesFailedMetric:    prometheus.NewCounterVec(prometheus.CounterOpts{Name: "t_c_retries_failed"}, retryLabels),
		incidentRetryAttemptsHistogram: prometheus.NewHistogramVec(prometheus.HistogramOpts{Name: "t_c_retry_attempts", Buckets: []float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}}, retryLabels),
	}
}

func newSmartRouterForRetryTest() *SmartRouterMetricsManager {
	return &SmartRouterMetricsManager{
		incidentRetriesTotalMetric:     prometheus.NewCounterVec(prometheus.CounterOpts{Name: "t_sr_retries_total"}, retryLabels),
		incidentRetriesSuccessMetric:   prometheus.NewCounterVec(prometheus.CounterOpts{Name: "t_sr_retries_success"}, retryLabels),
		incidentRetriesFailedMetric:    prometheus.NewCounterVec(prometheus.CounterOpts{Name: "t_sr_retries_failed"}, retryLabels),
		incidentRetryAttemptsHistogram: prometheus.NewHistogramVec(prometheus.HistogramOpts{Name: "t_sr_retry_attempts", Buckets: []float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}}, retryLabels),
		urlToProviderName:              make(map[string]string),
	}
}

// ---- Consumer tests ----

func TestConsumerRecordIncidentRetry_Success(t *testing.T) {
	cmm := newConsumerForRetryTest()
	labels := []string{"ETH1", "jsonrpc", "eth_blockNumber"}

	cmm.RecordIncidentRetry("ETH1", "jsonrpc", "eth_blockNumber", 2, true)

	require.Equal(t, float64(1), testutil.ToFloat64(cmm.incidentRetriesTotalMetric.WithLabelValues(labels...)))
	require.Equal(t, float64(1), testutil.ToFloat64(cmm.incidentRetriesSuccessMetric.WithLabelValues(labels...)))
	require.Equal(t, float64(0), testutil.ToFloat64(cmm.incidentRetriesFailedMetric.WithLabelValues(labels...)))
}

func TestConsumerRecordIncidentRetry_Failure(t *testing.T) {
	cmm := newConsumerForRetryTest()
	labels := []string{"ETH1", "jsonrpc", "eth_blockNumber"}

	cmm.RecordIncidentRetry("ETH1", "jsonrpc", "eth_blockNumber", 3, false)

	require.Equal(t, float64(1), testutil.ToFloat64(cmm.incidentRetriesTotalMetric.WithLabelValues(labels...)))
	require.Equal(t, float64(0), testutil.ToFloat64(cmm.incidentRetriesSuccessMetric.WithLabelValues(labels...)))
	require.Equal(t, float64(1), testutil.ToFloat64(cmm.incidentRetriesFailedMetric.WithLabelValues(labels...)))
}

// total always increments by 1 regardless of how many attempts count says
func TestConsumerRecordIncidentRetry_TotalAlwaysIncByOne(t *testing.T) {
	cmm := newConsumerForRetryTest()
	labels := []string{"ETH1", "jsonrpc", "eth_blockNumber"}

	cmm.RecordIncidentRetry("ETH1", "jsonrpc", "eth_blockNumber", 5, true)
	cmm.RecordIncidentRetry("ETH1", "jsonrpc", "eth_blockNumber", 10, false)

	require.Equal(t, float64(2), testutil.ToFloat64(cmm.incidentRetriesTotalMetric.WithLabelValues(labels...)))
}

func TestConsumerRecordIncidentRetry_TotalEqualsSuccessPlusFailed(t *testing.T) {
	cmm := newConsumerForRetryTest()
	labels := []string{"ETH1", "jsonrpc", "eth_blockNumber"}

	cmm.RecordIncidentRetry("ETH1", "jsonrpc", "eth_blockNumber", 2, true)
	cmm.RecordIncidentRetry("ETH1", "jsonrpc", "eth_blockNumber", 1, true)
	cmm.RecordIncidentRetry("ETH1", "jsonrpc", "eth_blockNumber", 4, false)

	total := testutil.ToFloat64(cmm.incidentRetriesTotalMetric.WithLabelValues(labels...))
	success := testutil.ToFloat64(cmm.incidentRetriesSuccessMetric.WithLabelValues(labels...))
	failed := testutil.ToFloat64(cmm.incidentRetriesFailedMetric.WithLabelValues(labels...))

	require.Equal(t, float64(3), total)
	require.Equal(t, float64(2), success)
	require.Equal(t, float64(1), failed)
	require.Equal(t, total, success+failed)
}

func TestConsumerRecordIncidentRetry_AttemptsHistogramObserved(t *testing.T) {
	cmm := newConsumerForRetryTest()

	require.NotPanics(t, func() {
		cmm.RecordIncidentRetry("ETH1", "jsonrpc", "eth_blockNumber", 3, true)
		cmm.RecordIncidentRetry("ETH1", "jsonrpc", "eth_blockNumber", 7, false)
	})
	require.Equal(t, 1, testutil.CollectAndCount(cmm.incidentRetryAttemptsHistogram))
}

func TestConsumerRecordIncidentRetry_NilManager(t *testing.T) {
	var cmm *ConsumerMetricsManager
	require.NotPanics(t, func() {
		cmm.RecordIncidentRetry("ETH1", "jsonrpc", "eth_blockNumber", 2, true)
	})
}

// ---- SmartRouter tests ----

func TestSmartRouterRecordIncidentRetry_Success(t *testing.T) {
	m := newSmartRouterForRetryTest()
	labels := []string{"ETH1", "jsonrpc", "eth_blockNumber"}

	m.RecordIncidentRetry("ETH1", "jsonrpc", "eth_blockNumber", 2, true)

	require.Equal(t, float64(1), testutil.ToFloat64(m.incidentRetriesTotalMetric.WithLabelValues(labels...)))
	require.Equal(t, float64(1), testutil.ToFloat64(m.incidentRetriesSuccessMetric.WithLabelValues(labels...)))
	require.Equal(t, float64(0), testutil.ToFloat64(m.incidentRetriesFailedMetric.WithLabelValues(labels...)))
}

func TestSmartRouterRecordIncidentRetry_Failure(t *testing.T) {
	m := newSmartRouterForRetryTest()
	labels := []string{"ETH1", "jsonrpc", "eth_blockNumber"}

	m.RecordIncidentRetry("ETH1", "jsonrpc", "eth_blockNumber", 3, false)

	require.Equal(t, float64(1), testutil.ToFloat64(m.incidentRetriesTotalMetric.WithLabelValues(labels...)))
	require.Equal(t, float64(0), testutil.ToFloat64(m.incidentRetriesSuccessMetric.WithLabelValues(labels...)))
	require.Equal(t, float64(1), testutil.ToFloat64(m.incidentRetriesFailedMetric.WithLabelValues(labels...)))
}

func TestSmartRouterRecordIncidentRetry_TotalAlwaysIncByOne(t *testing.T) {
	m := newSmartRouterForRetryTest()
	labels := []string{"ETH1", "jsonrpc", "eth_blockNumber"}

	m.RecordIncidentRetry("ETH1", "jsonrpc", "eth_blockNumber", 5, true)
	m.RecordIncidentRetry("ETH1", "jsonrpc", "eth_blockNumber", 10, false)

	require.Equal(t, float64(2), testutil.ToFloat64(m.incidentRetriesTotalMetric.WithLabelValues(labels...)))
}

func TestSmartRouterRecordIncidentRetry_TotalEqualsSuccessPlusFailed(t *testing.T) {
	m := newSmartRouterForRetryTest()
	labels := []string{"ETH1", "jsonrpc", "eth_blockNumber"}

	m.RecordIncidentRetry("ETH1", "jsonrpc", "eth_blockNumber", 2, true)
	m.RecordIncidentRetry("ETH1", "jsonrpc", "eth_blockNumber", 1, true)
	m.RecordIncidentRetry("ETH1", "jsonrpc", "eth_blockNumber", 4, false)

	total := testutil.ToFloat64(m.incidentRetriesTotalMetric.WithLabelValues(labels...))
	success := testutil.ToFloat64(m.incidentRetriesSuccessMetric.WithLabelValues(labels...))
	failed := testutil.ToFloat64(m.incidentRetriesFailedMetric.WithLabelValues(labels...))

	require.Equal(t, float64(3), total)
	require.Equal(t, float64(2), success)
	require.Equal(t, float64(1), failed)
	require.Equal(t, total, success+failed)
}

func TestSmartRouterRecordIncidentRetry_AttemptsHistogramObserved(t *testing.T) {
	m := newSmartRouterForRetryTest()

	require.NotPanics(t, func() {
		m.RecordIncidentRetry("ETH1", "jsonrpc", "eth_blockNumber", 3, true)
		m.RecordIncidentRetry("ETH1", "jsonrpc", "eth_blockNumber", 7, false)
	})
	require.Equal(t, 1, testutil.CollectAndCount(m.incidentRetryAttemptsHistogram))
}

func TestSmartRouterRecordIncidentRetry_NilManager(t *testing.T) {
	var m *SmartRouterMetricsManager
	require.NotPanics(t, func() {
		m.RecordIncidentRetry("ETH1", "jsonrpc", "eth_blockNumber", 2, true)
	})
}
