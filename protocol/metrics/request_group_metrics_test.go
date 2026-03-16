package metrics

import (
	"errors"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/require"
)

// newConsumerForRequestGroupTest constructs a minimal ConsumerMetricsManager
// with fresh, unregistered counters. This avoids HTTP-mux and Prometheus
// registry conflicts when running multiple test functions.
func newConsumerForRequestGroupTest() *ConsumerMetricsManager {
	reqLabels := []string{"spec", "apiInterface", "provider_address", "method"}
	return &ConsumerMetricsManager{
		// Fields used by SetRelayMetrics before the request-group block
		totalCURequestedMetric: prometheus.NewCounterVec(prometheus.CounterOpts{Name: "t_cu"}, []string{"spec", "apiInterface"}),
		// Request-group counters under test
		requestsTotalMetric:      prometheus.NewCounterVec(prometheus.CounterOpts{Name: "t_req_total"}, reqLabels),
		requestsSuccessMetric:    prometheus.NewCounterVec(prometheus.CounterOpts{Name: "t_req_success"}, reqLabels),
		requestsFailedMetric:     prometheus.NewCounterVec(prometheus.CounterOpts{Name: "t_req_failed"}, reqLabels),
		requestsReadMetric:       prometheus.NewCounterVec(prometheus.CounterOpts{Name: "t_req_read"}, reqLabels),
		requestsWriteMetric:      prometheus.NewCounterVec(prometheus.CounterOpts{Name: "t_req_write"}, reqLabels),
		requestsDebugTraceMetric: prometheus.NewCounterVec(prometheus.CounterOpts{Name: "t_req_debug"}, reqLabels),
		requestsArchiveMetric:    prometheus.NewCounterVec(prometheus.CounterOpts{Name: "t_req_archive"}, reqLabels),
		requestsBatchMetric:      prometheus.NewCounterVec(prometheus.CounterOpts{Name: "t_req_batch"}, reqLabels),
	}
}

// newSmartRouterForRequestGroupTest constructs a minimal SmartRouterMetricsManager
// with fresh, unregistered counters for testing RecordDirectRelayEnd logic.
func newSmartRouterForRequestGroupTest() *SmartRouterMetricsManager {
	reqLabels := []string{"spec", "apiInterface", "provider_address", "method"}
	endpointLabels := []string{"spec", "apiInterface", "endpoint_id", "function"}
	funcLabels := []string{"spec", "apiInterface", "endpoint_id", "function"}
	return &SmartRouterMetricsManager{
		// Fields touched by RecordDirectRelayEnd before the request-group block
		endpointInFlight:            NewMappedLabelsGaugeVec(MappedLabelsMetricOpts{Name: "t_sr_inflight", Labels: funcLabels}),
		endpointTotalRelaysServiced: NewMappedLabelsCounterVec(MappedLabelsMetricOpts{Name: "t_sr_relays", Labels: funcLabels}),
		endpointTotalErrored:        NewMappedLabelsCounterVec(MappedLabelsMetricOpts{Name: "t_sr_errored", Labels: funcLabels}),
		endpointEndToEndLatency:     prometheus.NewHistogramVec(prometheus.HistogramOpts{Name: "t_sr_latency"}, endpointLabels),
		// Request-group counters under test
		routerRequestsTotal:      prometheus.NewCounterVec(prometheus.CounterOpts{Name: "t_sr_req_total"}, reqLabels),
		routerRequestsSuccess:    prometheus.NewCounterVec(prometheus.CounterOpts{Name: "t_sr_req_success"}, reqLabels),
		routerRequestsFailed:     prometheus.NewCounterVec(prometheus.CounterOpts{Name: "t_sr_req_failed"}, reqLabels),
		routerRequestsRead:       prometheus.NewCounterVec(prometheus.CounterOpts{Name: "t_sr_req_read"}, reqLabels),
		routerRequestsWrite:      prometheus.NewCounterVec(prometheus.CounterOpts{Name: "t_sr_req_write"}, reqLabels),
		routerRequestsDebugTrace: prometheus.NewCounterVec(prometheus.CounterOpts{Name: "t_sr_req_debug"}, reqLabels),
		routerRequestsArchive:    prometheus.NewCounterVec(prometheus.CounterOpts{Name: "t_sr_req_archive"}, reqLabels),
		routerRequestsBatch:      prometheus.NewCounterVec(prometheus.CounterOpts{Name: "t_sr_req_batch"}, reqLabels),
		urlToProviderName:        make(map[string]string),
	}
}

// ---- Consumer tests ----

func TestConsumerSetRelayMetrics_SuccessCounters(t *testing.T) {
	cmm := newConsumerForRequestGroupTest()
	labels := []string{"ETH1", "jsonrpc", "provider1", "eth_blockNumber"}

	cmm.SetRelayMetrics(&RelayMetrics{ChainID: "ETH1", APIType: "jsonrpc", ProviderAddress: "provider1", ApiMethod: "eth_blockNumber", Success: true}, nil)

	require.Equal(t, float64(1), testutil.ToFloat64(cmm.requestsTotalMetric.WithLabelValues(labels...)))
	require.Equal(t, float64(1), testutil.ToFloat64(cmm.requestsSuccessMetric.WithLabelValues(labels...)))
	require.Equal(t, float64(0), testutil.ToFloat64(cmm.requestsFailedMetric.WithLabelValues(labels...)))
}

func TestConsumerSetRelayMetrics_FailureCounters(t *testing.T) {
	cmm := newConsumerForRequestGroupTest()
	labels := []string{"ETH1", "jsonrpc", "provider1", "eth_blockNumber"}

	// Pass a non-nil error so SetRelayMetrics sets Success=false (it overwrites Success = err == nil)
	cmm.SetRelayMetrics(&RelayMetrics{ChainID: "ETH1", APIType: "jsonrpc", ProviderAddress: "provider1", ApiMethod: "eth_blockNumber"}, errors.New("relay failed"))

	require.Equal(t, float64(1), testutil.ToFloat64(cmm.requestsTotalMetric.WithLabelValues(labels...)))
	require.Equal(t, float64(0), testutil.ToFloat64(cmm.requestsSuccessMetric.WithLabelValues(labels...)))
	require.Equal(t, float64(1), testutil.ToFloat64(cmm.requestsFailedMetric.WithLabelValues(labels...)))
}

func TestConsumerSetRelayMetrics_ReadRequest(t *testing.T) {
	cmm := newConsumerForRequestGroupTest()
	labels := []string{"ETH1", "jsonrpc", "provider1", "eth_blockNumber"}

	cmm.SetRelayMetrics(&RelayMetrics{ChainID: "ETH1", APIType: "jsonrpc", ProviderAddress: "provider1", ApiMethod: "eth_blockNumber", Success: true, IsWrite: false}, nil)

	require.Equal(t, float64(1), testutil.ToFloat64(cmm.requestsReadMetric.WithLabelValues(labels...)))
	require.Equal(t, float64(0), testutil.ToFloat64(cmm.requestsWriteMetric.WithLabelValues(labels...)))
}

func TestConsumerSetRelayMetrics_WriteRequest(t *testing.T) {
	cmm := newConsumerForRequestGroupTest()
	labels := []string{"ETH1", "jsonrpc", "provider1", "eth_sendRawTransaction"}

	cmm.SetRelayMetrics(&RelayMetrics{ChainID: "ETH1", APIType: "jsonrpc", ProviderAddress: "provider1", ApiMethod: "eth_sendRawTransaction", Success: true, IsWrite: true}, nil)

	require.Equal(t, float64(1), testutil.ToFloat64(cmm.requestsWriteMetric.WithLabelValues(labels...)))
	require.Equal(t, float64(0), testutil.ToFloat64(cmm.requestsReadMetric.WithLabelValues(labels...)))
}

func TestConsumerSetRelayMetrics_ArchiveRequest(t *testing.T) {
	cmm := newConsumerForRequestGroupTest()
	labels := []string{"ETH1", "jsonrpc", "provider1", "eth_getBlockByNumber"}

	cmm.SetRelayMetrics(&RelayMetrics{ChainID: "ETH1", APIType: "jsonrpc", ProviderAddress: "provider1", ApiMethod: "eth_getBlockByNumber", Success: true, IsArchive: true}, nil)

	require.Equal(t, float64(1), testutil.ToFloat64(cmm.requestsArchiveMetric.WithLabelValues(labels...)))
	require.Equal(t, float64(1), testutil.ToFloat64(cmm.requestsReadMetric.WithLabelValues(labels...)))
	require.Equal(t, float64(0), testutil.ToFloat64(cmm.requestsBatchMetric.WithLabelValues(labels...)))
}

func TestConsumerSetRelayMetrics_DebugTraceRequest(t *testing.T) {
	cmm := newConsumerForRequestGroupTest()
	labels := []string{"ETH1", "jsonrpc", "provider1", "debug_traceTransaction"}

	cmm.SetRelayMetrics(&RelayMetrics{ChainID: "ETH1", APIType: "jsonrpc", ProviderAddress: "provider1", ApiMethod: "debug_traceTransaction", Success: true, IsDebugTrace: true}, nil)

	require.Equal(t, float64(1), testutil.ToFloat64(cmm.requestsDebugTraceMetric.WithLabelValues(labels...)))
	require.Equal(t, float64(0), testutil.ToFloat64(cmm.requestsBatchMetric.WithLabelValues(labels...)))
}

func TestConsumerSetRelayMetrics_BatchRequest_MutuallyExclusive(t *testing.T) {
	cmm := newConsumerForRequestGroupTest()
	labels := []string{"ETH1", "jsonrpc", "provider1", "batch"}

	// batch with all other flags set — only batch counter should fire
	cmm.SetRelayMetrics(&RelayMetrics{
		ChainID: "ETH1", APIType: "jsonrpc", ProviderAddress: "provider1", ApiMethod: "batch",
		Success: true, IsBatch: true, IsWrite: true, IsArchive: true, IsDebugTrace: true,
	}, nil)

	require.Equal(t, float64(1), testutil.ToFloat64(cmm.requestsBatchMetric.WithLabelValues(labels...)))
	require.Equal(t, float64(0), testutil.ToFloat64(cmm.requestsReadMetric.WithLabelValues(labels...)))
	require.Equal(t, float64(0), testutil.ToFloat64(cmm.requestsWriteMetric.WithLabelValues(labels...)))
	require.Equal(t, float64(0), testutil.ToFloat64(cmm.requestsArchiveMetric.WithLabelValues(labels...)))
	require.Equal(t, float64(0), testutil.ToFloat64(cmm.requestsDebugTraceMetric.WithLabelValues(labels...)))
}

func TestConsumerSetRelayMetrics_NilManager(t *testing.T) {
	var cmm *ConsumerMetricsManager
	require.NotPanics(t, func() {
		cmm.SetRelayMetrics(&RelayMetrics{ChainID: "ETH1", Success: true}, nil)
	})
}

// ---- SmartRouter tests ----

func TestSmartRouterRecordDirectRelayEnd_SuccessCounters(t *testing.T) {
	m := newSmartRouterForRequestGroupTest()
	labels := []string{"ETH1", "jsonrpc", "ep1", "eth_blockNumber"}

	m.RecordDirectRelayEnd("ETH1", "jsonrpc", "ep1", "eth_blockNumber", 10, true, RequestProperties{})

	require.Equal(t, float64(1), testutil.ToFloat64(m.routerRequestsTotal.WithLabelValues(labels...)))
	require.Equal(t, float64(1), testutil.ToFloat64(m.routerRequestsSuccess.WithLabelValues(labels...)))
	require.Equal(t, float64(0), testutil.ToFloat64(m.routerRequestsFailed.WithLabelValues(labels...)))
}

func TestSmartRouterRecordDirectRelayEnd_FailureCounters(t *testing.T) {
	m := newSmartRouterForRequestGroupTest()
	labels := []string{"ETH1", "jsonrpc", "ep1", "eth_blockNumber"}

	m.RecordDirectRelayEnd("ETH1", "jsonrpc", "ep1", "eth_blockNumber", 10, false, RequestProperties{})

	require.Equal(t, float64(1), testutil.ToFloat64(m.routerRequestsTotal.WithLabelValues(labels...)))
	require.Equal(t, float64(0), testutil.ToFloat64(m.routerRequestsSuccess.WithLabelValues(labels...)))
	require.Equal(t, float64(1), testutil.ToFloat64(m.routerRequestsFailed.WithLabelValues(labels...)))
}

func TestSmartRouterRecordDirectRelayEnd_ReadRequest(t *testing.T) {
	m := newSmartRouterForRequestGroupTest()
	labels := []string{"ETH1", "jsonrpc", "ep1", "eth_blockNumber"}

	m.RecordDirectRelayEnd("ETH1", "jsonrpc", "ep1", "eth_blockNumber", 10, true, RequestProperties{IsWrite: false})

	require.Equal(t, float64(1), testutil.ToFloat64(m.routerRequestsRead.WithLabelValues(labels...)))
	require.Equal(t, float64(0), testutil.ToFloat64(m.routerRequestsWrite.WithLabelValues(labels...)))
}

func TestSmartRouterRecordDirectRelayEnd_WriteRequest(t *testing.T) {
	m := newSmartRouterForRequestGroupTest()
	labels := []string{"ETH1", "jsonrpc", "ep1", "eth_sendRawTransaction"}

	m.RecordDirectRelayEnd("ETH1", "jsonrpc", "ep1", "eth_sendRawTransaction", 10, true, RequestProperties{IsWrite: true})

	require.Equal(t, float64(1), testutil.ToFloat64(m.routerRequestsWrite.WithLabelValues(labels...)))
	require.Equal(t, float64(0), testutil.ToFloat64(m.routerRequestsRead.WithLabelValues(labels...)))
}

func TestSmartRouterRecordDirectRelayEnd_ArchiveRequest(t *testing.T) {
	m := newSmartRouterForRequestGroupTest()
	labels := []string{"ETH1", "jsonrpc", "ep1", "eth_getBlockByNumber"}

	m.RecordDirectRelayEnd("ETH1", "jsonrpc", "ep1", "eth_getBlockByNumber", 10, true, RequestProperties{IsArchive: true})

	require.Equal(t, float64(1), testutil.ToFloat64(m.routerRequestsArchive.WithLabelValues(labels...)))
	require.Equal(t, float64(1), testutil.ToFloat64(m.routerRequestsRead.WithLabelValues(labels...)))
	require.Equal(t, float64(0), testutil.ToFloat64(m.routerRequestsBatch.WithLabelValues(labels...)))
}

func TestSmartRouterRecordDirectRelayEnd_DebugTraceRequest(t *testing.T) {
	m := newSmartRouterForRequestGroupTest()
	labels := []string{"ETH1", "jsonrpc", "ep1", "debug_traceTransaction"}

	m.RecordDirectRelayEnd("ETH1", "jsonrpc", "ep1", "debug_traceTransaction", 10, true, RequestProperties{IsDebugTrace: true})

	require.Equal(t, float64(1), testutil.ToFloat64(m.routerRequestsDebugTrace.WithLabelValues(labels...)))
	require.Equal(t, float64(0), testutil.ToFloat64(m.routerRequestsBatch.WithLabelValues(labels...)))
}

func TestSmartRouterRecordDirectRelayEnd_BatchRequest_MutuallyExclusive(t *testing.T) {
	m := newSmartRouterForRequestGroupTest()
	labels := []string{"ETH1", "jsonrpc", "ep1", "batch"}

	m.RecordDirectRelayEnd("ETH1", "jsonrpc", "ep1", "batch", 10, true, RequestProperties{
		IsBatch: true, IsWrite: true, IsArchive: true, IsDebugTrace: true,
	})

	require.Equal(t, float64(1), testutil.ToFloat64(m.routerRequestsBatch.WithLabelValues(labels...)))
	require.Equal(t, float64(0), testutil.ToFloat64(m.routerRequestsRead.WithLabelValues(labels...)))
	require.Equal(t, float64(0), testutil.ToFloat64(m.routerRequestsWrite.WithLabelValues(labels...)))
	require.Equal(t, float64(0), testutil.ToFloat64(m.routerRequestsArchive.WithLabelValues(labels...)))
	require.Equal(t, float64(0), testutil.ToFloat64(m.routerRequestsDebugTrace.WithLabelValues(labels...)))
}

func TestSmartRouterRecordDirectRelayEnd_NilManager(t *testing.T) {
	var m *SmartRouterMetricsManager
	require.NotPanics(t, func() {
		m.RecordDirectRelayEnd("ETH1", "jsonrpc", "ep1", "method", 10, true, RequestProperties{})
	})
}
