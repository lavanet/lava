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

// requestGroupRunner is a thin adapter that lets the shared test case table
// drive ConsumerMetricsManager.
type requestGroupRunner struct {
	// invoke dispatches a single relay through the manager under test.
	// err==nil means success; err!=nil means failure.
	invoke  func(relay RelayMetrics, err error)
	labels  func(method string) []string
	total   *prometheus.CounterVec
	success *prometheus.CounterVec
	failed  *prometheus.CounterVec
	read    *prometheus.CounterVec
	write   *prometheus.CounterVec
	archive *prometheus.CounterVec
	debug   *prometheus.CounterVec
	batch   *prometheus.CounterVec
}

func newConsumerRequestGroupRunner() *requestGroupRunner {
	cmm := newConsumerForRequestGroupTest()
	return &requestGroupRunner{
		invoke: func(r RelayMetrics, e error) {
			r.ChainID, r.APIType, r.ProviderAddress = "ETH1", "jsonrpc", "provider1"
			cmm.SetRelayMetrics(&r, e)
		},
		labels:  func(method string) []string { return []string{"ETH1", "jsonrpc", "provider1", method} },
		total:   cmm.requestsTotalMetric,
		success: cmm.requestsSuccessMetric,
		failed:  cmm.requestsFailedMetric,
		read:    cmm.requestsReadMetric,
		write:   cmm.requestsWriteMetric,
		archive: cmm.requestsArchiveMetric,
		debug:   cmm.requestsDebugTraceMetric,
		batch:   cmm.requestsBatchMetric,
	}
}

func (r *requestGroupRunner) assertCounters(t *testing.T, method string,
	wantTotal, wantSuccess, wantFailed,
	wantRead, wantWrite, wantArchive, wantDebug, wantBatch float64,
) {
	t.Helper()
	l := r.labels(method)
	require.Equal(t, wantTotal, testutil.ToFloat64(r.total.WithLabelValues(l...)), "total")
	require.Equal(t, wantSuccess, testutil.ToFloat64(r.success.WithLabelValues(l...)), "success")
	require.Equal(t, wantFailed, testutil.ToFloat64(r.failed.WithLabelValues(l...)), "failed")
	require.Equal(t, wantRead, testutil.ToFloat64(r.read.WithLabelValues(l...)), "read")
	require.Equal(t, wantWrite, testutil.ToFloat64(r.write.WithLabelValues(l...)), "write")
	require.Equal(t, wantArchive, testutil.ToFloat64(r.archive.WithLabelValues(l...)), "archive")
	require.Equal(t, wantDebug, testutil.ToFloat64(r.debug.WithLabelValues(l...)), "debug")
	require.Equal(t, wantBatch, testutil.ToFloat64(r.batch.WithLabelValues(l...)), "batch")
}

// TestSetRelayMetrics covers the request-group counter logic for
// ConsumerMetricsManager (via SetRelayMetrics) using a table of cases.
func TestSetRelayMetrics(t *testing.T) {
	cases := []struct {
		name        string
		relay       RelayMetrics
		err         error // nil = success
		wantTotal   float64
		wantSuccess float64
		wantFailed  float64
		wantRead    float64
		wantWrite   float64
		wantArchive float64
		wantDebug   float64
		wantBatch   float64
	}{
		{
			name:      "read/success",
			relay:     RelayMetrics{ApiMethod: "eth_blockNumber"},
			wantTotal: 1, wantSuccess: 1,
			wantRead: 1,
		},
		{
			name:      "read/failure",
			relay:     RelayMetrics{ApiMethod: "eth_blockNumber"},
			err:       errors.New("relay failed"),
			wantTotal: 1, wantFailed: 1,
			wantRead: 1,
		},
		{
			name:      "write",
			relay:     RelayMetrics{ApiMethod: "eth_sendRawTransaction", IsWrite: true},
			wantTotal: 1, wantSuccess: 1,
			wantWrite: 1,
		},
		{
			name:      "archive",
			relay:     RelayMetrics{ApiMethod: "eth_getBlockByNumber", IsArchive: true},
			wantTotal: 1, wantSuccess: 1,
			wantRead: 1, wantArchive: 1,
		},
		{
			name:      "debug_trace",
			relay:     RelayMetrics{ApiMethod: "debug_traceTransaction", IsDebugTrace: true},
			wantTotal: 1, wantSuccess: 1,
			wantRead: 1, wantDebug: 1,
		},
		{
			// IsBatch is mutually exclusive: read/write/archive/debug must all stay zero.
			name:      "batch/mutually_exclusive",
			relay:     RelayMetrics{ApiMethod: "batch", IsBatch: true, IsWrite: true, IsArchive: true, IsDebugTrace: true},
			wantTotal: 1, wantSuccess: 1,
			wantBatch: 1,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			runner := newConsumerRequestGroupRunner()
			runner.invoke(tc.relay, tc.err)
			runner.assertCounters(t, tc.relay.ApiMethod,
				tc.wantTotal, tc.wantSuccess, tc.wantFailed,
				tc.wantRead, tc.wantWrite, tc.wantArchive, tc.wantDebug, tc.wantBatch,
			)
		})
	}
}

func TestConsumerSetRelayMetrics_NilManager(t *testing.T) {
	var cmm *ConsumerMetricsManager
	require.NotPanics(t, func() {
		cmm.SetRelayMetrics(&RelayMetrics{ChainID: "ETH1", Success: true}, nil)
	})
}

// TestConsumerSetRelayMetrics_PartitionInvariant asserts the documented counting
// invariants: batch+read+write==total and success+failed==total across a mixed
// workload. debug_trace and archive are orthogonal sub-categories and are NOT
// expected to sum to total.
func TestConsumerSetRelayMetrics_PartitionInvariant(t *testing.T) {
	cmm := newConsumerForRequestGroupTest()
	l := []string{"ETH1", "jsonrpc", "p1", "m"}

	send := func(rm RelayMetrics, err error) {
		rm.ChainID, rm.APIType, rm.ProviderAddress, rm.ApiMethod = "ETH1", "jsonrpc", "p1", "m"
		cmm.SetRelayMetrics(&rm, err)
	}

	send(RelayMetrics{IsWrite: false}, nil)                 // read, success
	send(RelayMetrics{IsWrite: true}, nil)                  // write, success
	send(RelayMetrics{IsBatch: true}, nil)                  // batch, success
	send(RelayMetrics{IsArchive: true}, errors.New("e"))    // read+archive, failed
	send(RelayMetrics{IsDebugTrace: true}, nil)             // read+debug, success
	send(RelayMetrics{IsWrite: true, IsArchive: true}, nil) // write+archive, success

	total := testutil.ToFloat64(cmm.requestsTotalMetric.WithLabelValues(l...))
	read := testutil.ToFloat64(cmm.requestsReadMetric.WithLabelValues(l...))
	write := testutil.ToFloat64(cmm.requestsWriteMetric.WithLabelValues(l...))
	batch := testutil.ToFloat64(cmm.requestsBatchMetric.WithLabelValues(l...))
	success := testutil.ToFloat64(cmm.requestsSuccessMetric.WithLabelValues(l...))
	failed := testutil.ToFloat64(cmm.requestsFailedMetric.WithLabelValues(l...))

	require.Equal(t, float64(6), total)
	require.Equal(t, total, batch+read+write, "batch+read+write must equal total")
	require.Equal(t, total, success+failed, "success+failed must equal total")
}
