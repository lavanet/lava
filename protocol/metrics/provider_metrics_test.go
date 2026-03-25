package metrics

import (
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/require"
)

func newProviderMetricsForTest(t *testing.T) *ProviderMetrics {
	t.Helper()
	// Use t.Name() as suffix so each test gets its own metric instances,
	// preventing counter accumulation across tests via the global registry.
	p := strings.NewReplacer("/", "_", "-", "_", " ", "_").Replace(t.Name())
	functionLabels := []string{"spec", "apiInterface", "function"}
	return &ProviderMetrics{
		specID:       "ETH1",
		apiInterface: "jsonrpc",
		totalCUServicedMetric: prometheus.NewCounterVec(
			prometheus.CounterOpts{Name: "t_p_cu_serviced_" + p}, []string{"spec", "apiInterface"}),
		totalCUPaidMetric: prometheus.NewCounterVec(
			prometheus.CounterOpts{Name: "t_p_cu_paid_" + p}, []string{"spec"}),
		totalRelaysServicedMetric: NewMappedLabelsCounterVec(MappedLabelsMetricOpts{
			Name: "t_p_relays_serviced_" + p, Labels: functionLabels,
		}),
		totalErroredMetric: NewMappedLabelsCounterVec(MappedLabelsMetricOpts{
			Name: "t_p_errored_" + p, Labels: functionLabels,
		}),
		inFlightPerFunctionMetric: NewMappedLabelsGaugeVec(MappedLabelsMetricOpts{
			Name: "t_p_in_flight_" + p, Labels: functionLabels,
		}),
		requestLatencyPerFunctionMetric: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{Name: "t_p_req_latency_" + p, Buckets: prometheus.DefBuckets}, functionLabels),
		loadRateMetric: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{Name: "t_p_load_rate_" + p}, []string{"spec"}),
		providerLatencyMetric: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{Name: "t_p_latency_" + p, Buckets: prometheus.DefBuckets}, []string{"spec", "apiInterface"}),
		providerEndToEndLatencyMetric: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{Name: "t_p_e2e_latency_" + p, Buckets: prometheus.DefBuckets}, []string{"spec", "apiInterface"}),
	}
}

// ---- AddRelay ----

func TestProviderMetricsAddRelay_IncrementsRelaysServiced(t *testing.T) {
	pm := newProviderMetricsForTest(t)
	labels := map[string]string{"spec": "ETH1", "apiInterface": "jsonrpc", "function": "eth_blockNumber"}

	pm.AddRelay(10, "eth_blockNumber")

	require.Equal(t, float64(1), testutil.ToFloat64(pm.totalRelaysServicedMetric.WithLabelValues(labels)))
}

func TestProviderMetricsAddRelay_AccumulatesCU(t *testing.T) {
	pm := newProviderMetricsForTest(t)

	pm.AddRelay(10, "eth_blockNumber")
	pm.AddRelay(5, "eth_blockNumber")

	require.Equal(t, float64(15), testutil.ToFloat64(pm.totalCUServicedMetric.WithLabelValues("ETH1", "jsonrpc")))
}

func TestProviderMetricsAddRelay_MultipleRelaysAccumulate(t *testing.T) {
	pm := newProviderMetricsForTest(t)
	labels := map[string]string{"spec": "ETH1", "apiInterface": "jsonrpc", "function": "eth_blockNumber"}

	pm.AddRelay(10, "eth_blockNumber")
	pm.AddRelay(10, "eth_blockNumber")
	pm.AddRelay(10, "eth_blockNumber")

	require.Equal(t, float64(3), testutil.ToFloat64(pm.totalRelaysServicedMetric.WithLabelValues(labels)))
}

func TestProviderMetricsAddRelay_NilSafe(t *testing.T) {
	var pm *ProviderMetrics
	require.NotPanics(t, func() { pm.AddRelay(10, "fn") })
}

// ---- AddFunctionError ----

func TestProviderMetricsAddFunctionError_IncrementsRelaysAndErrored(t *testing.T) {
	pm := newProviderMetricsForTest(t)
	labels := map[string]string{"spec": "ETH1", "apiInterface": "jsonrpc", "function": "eth_blockNumber"}

	pm.AddFunctionError("eth_blockNumber")

	require.Equal(t, float64(1), testutil.ToFloat64(pm.totalRelaysServicedMetric.WithLabelValues(labels)))
	require.Equal(t, float64(1), testutil.ToFloat64(pm.totalErroredMetric.WithLabelValues(labels)))
}

func TestProviderMetricsAddFunctionError_TotalEqualsSuccessPlusErrors(t *testing.T) {
	pm := newProviderMetricsForTest(t)
	labels := map[string]string{"spec": "ETH1", "apiInterface": "jsonrpc", "function": "eth_blockNumber"}

	pm.AddRelay(10, "eth_blockNumber")     // success relay
	pm.AddRelay(10, "eth_blockNumber")     // success relay
	pm.AddFunctionError("eth_blockNumber") // error relay

	total := testutil.ToFloat64(pm.totalRelaysServicedMetric.WithLabelValues(labels))
	errored := testutil.ToFloat64(pm.totalErroredMetric.WithLabelValues(labels))

	require.Equal(t, float64(3), total)
	require.Equal(t, float64(1), errored)
}

func TestProviderMetricsAddFunctionError_NilSafe(t *testing.T) {
	var pm *ProviderMetrics
	require.NotPanics(t, func() { pm.AddFunctionError("fn") })
}

// ---- AddInFlightRelay / SubInFlightRelay ----

func TestProviderMetricsInFlight_AddAndSub(t *testing.T) {
	pm := newProviderMetricsForTest(t)
	labels := map[string]string{"spec": "ETH1", "apiInterface": "jsonrpc", "function": "eth_blockNumber"}

	pm.AddInFlightRelay("eth_blockNumber")
	pm.AddInFlightRelay("eth_blockNumber")
	require.Equal(t, float64(2), testutil.ToFloat64(pm.inFlightPerFunctionMetric.WithLabelValues(labels)))

	pm.SubInFlightRelay("eth_blockNumber")
	require.Equal(t, float64(1), testutil.ToFloat64(pm.inFlightPerFunctionMetric.WithLabelValues(labels)))
}

func TestProviderMetricsInFlight_NilSafe(t *testing.T) {
	var pm *ProviderMetrics
	require.NotPanics(t, func() { pm.AddInFlightRelay("fn") })
	require.NotPanics(t, func() { pm.SubInFlightRelay("fn") })
}

// ---- Latency histograms ----

func TestProviderMetricsSetLatency_ObservedInHistogram(t *testing.T) {
	pm := newProviderMetricsForTest(t)

	pm.SetLatency(42.0)

	reg := prometheus.NewRegistry()
	require.NoError(t, reg.Register(pm.providerLatencyMetric))
	families, err := reg.Gather()
	require.NoError(t, err)
	require.Len(t, families, 1)
	m := families[0].GetMetric()
	require.Len(t, m, 1)
	require.Equal(t, uint64(1), m[0].GetHistogram().GetSampleCount())
	require.InDelta(t, 42.0, m[0].GetHistogram().GetSampleSum(), 0.001)
}

func TestProviderMetricsSetEndToEndLatency_ObservedInHistogram(t *testing.T) {
	pm := newProviderMetricsForTest(t)

	pm.SetEndToEndLatency(75.0)
	pm.SetEndToEndLatency(25.0)

	reg := prometheus.NewRegistry()
	require.NoError(t, reg.Register(pm.providerEndToEndLatencyMetric))
	families, err := reg.Gather()
	require.NoError(t, err)
	require.Len(t, families, 1)
	m := families[0].GetMetric()
	require.Len(t, m, 1)
	require.Equal(t, uint64(2), m[0].GetHistogram().GetSampleCount())
	require.InDelta(t, 100.0, m[0].GetHistogram().GetSampleSum(), 0.001)
}

func TestProviderMetricsAddFunctionLatency_ObservedInHistogram(t *testing.T) {
	pm := newProviderMetricsForTest(t)

	pm.AddFunctionLatency("eth_blockNumber", 30*time.Millisecond)
	pm.AddFunctionLatency("eth_blockNumber", 20*time.Millisecond)

	reg := prometheus.NewRegistry()
	require.NoError(t, reg.Register(pm.requestLatencyPerFunctionMetric))
	families, err := reg.Gather()
	require.NoError(t, err)
	require.Len(t, families, 1)
	m := families[0].GetMetric()
	require.Len(t, m, 1)
	require.Equal(t, uint64(2), m[0].GetHistogram().GetSampleCount())
	require.InDelta(t, 50.0, m[0].GetHistogram().GetSampleSum(), 0.001)
}

func TestProviderMetricsSetLatency_NilSafe(t *testing.T) {
	var pm *ProviderMetrics
	require.NotPanics(t, func() { pm.SetLatency(10.0) })
	require.NotPanics(t, func() { pm.SetEndToEndLatency(10.0) })
	require.NotPanics(t, func() { pm.AddFunctionLatency("fn", time.Millisecond) })
}

// ---- AddPayment ----

func TestProviderMetricsAddPayment_AccumulatesCU(t *testing.T) {
	pm := newProviderMetricsForTest(t)

	pm.AddPayment(100)
	pm.AddPayment(50)

	require.Equal(t, float64(150), testutil.ToFloat64(pm.totalCUPaidMetric.WithLabelValues("ETH1")))
}

func TestProviderMetricsAddPayment_NilSafe(t *testing.T) {
	var pm *ProviderMetrics
	require.NotPanics(t, func() { pm.AddPayment(10) })
}

// ---- ProviderMetricsManager ----

// newProviderMetricsManagerForTest builds a ProviderMetricsManager with its own
// registry — no global registration and no HTTP server.
func newProviderMetricsManagerForTest(t *testing.T) *ProviderMetricsManager {
	t.Helper()
	p := strings.NewReplacer("/", "_", "-", "_", " ", "_").Replace(t.Name())
	functionLabels := []string{"spec", "apiInterface", "function"}
	blockLabels := []string{"spec", "apiInterface"}

	return &ProviderMetricsManager{
		providerMetrics: map[string]*ProviderMetrics{},
		totalCUServicedMetric: prometheus.NewCounterVec(
			prometheus.CounterOpts{Name: "tm_cu_serviced_" + p}, []string{"spec", "apiInterface"}),
		totalCUPaidMetric: prometheus.NewCounterVec(
			prometheus.CounterOpts{Name: "tm_cu_paid_" + p}, []string{"spec"}),
		totalRelaysServicedMetric: NewMappedLabelsCounterVec(MappedLabelsMetricOpts{
			Name: "tm_relays_" + p, Labels: functionLabels,
		}),
		totalErroredMetric: NewMappedLabelsCounterVec(MappedLabelsMetricOpts{
			Name: "tm_errored_" + p, Labels: functionLabels,
		}),
		inFlightPerFunctionMetric: NewMappedLabelsGaugeVec(MappedLabelsMetricOpts{
			Name: "tm_inflight_" + p, Labels: functionLabels,
		}),
		requestLatencyPerFunctionMetric: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{Name: "tm_req_lat_" + p, Buckets: prometheus.DefBuckets}, functionLabels),
		blockMetric: NewMappedLabelsGaugeVec(MappedLabelsMetricOpts{
			Name: "tm_block_" + p, Labels: blockLabels,
		}),
		lastServicedBlockTimeMetric: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{Name: "tm_last_block_time_" + p}, []string{"spec"}),
		disabledChainsMetric: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{Name: "tm_disabled_" + p}, []string{"chainID", "apiInterface"}),
		fetchLatestFailedMetric: prometheus.NewCounterVec(
			prometheus.CounterOpts{Name: "tm_fetch_latest_fail_" + p}, []string{"spec", "apiInterface"}),
		fetchBlockFailedMetric: prometheus.NewCounterVec(
			prometheus.CounterOpts{Name: "tm_fetch_block_fail_" + p}, []string{"spec", "apiInterface"}),
		fetchLatestSuccessMetric: prometheus.NewCounterVec(
			prometheus.CounterOpts{Name: "tm_fetch_latest_ok_" + p}, []string{"spec", "apiInterface"}),
		fetchBlockSuccessMetric: prometheus.NewCounterVec(
			prometheus.CounterOpts{Name: "tm_fetch_block_ok_" + p}, []string{"spec", "apiInterface"}),
		virtualEpochMetric: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{Name: "tm_vepoch_" + p}, []string{"spec"}),
		endpointsHealthChecksOkMetric: prometheus.NewGauge(
			prometheus.GaugeOpts{Name: "tm_health_ok_" + p}),
		endpointsHealthChecksBreakdownMetric: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{Name: "tm_health_bd_" + p}, []string{"spec", "apiInterface"}),
		protocolVersionMetric: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{Name: "tm_version_" + p}, []string{"version"}),
		frozenStatusMetric: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{Name: "tm_frozen_" + p}, []string{"chainID"}),
		jailStatusMetric: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{Name: "tm_jail_" + p}, []string{"chainID"}),
		jailedCountMetric: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{Name: "tm_jailed_cnt_" + p}, []string{"chainID"}),
		loadRateMetric: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{Name: "tm_load_" + p}, []string{"spec"}),
		providerLatencyMetric: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{Name: "tm_lat_" + p, Buckets: prometheus.DefBuckets}, []string{"spec", "apiInterface"}),
		providerEndToEndLatencyMetric: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{Name: "tm_e2e_" + p, Buckets: prometheus.DefBuckets}, []string{"spec", "apiInterface"}),
		relaysMonitors: map[string]*RelaysMonitor{},
	}
}

func TestProviderMetricsManager_SetLatestBlock(t *testing.T) {
	mgr := newProviderMetricsManagerForTest(t)
	mgr.SetLatestBlock("ETH1", "jsonrpc", "ep1", 12345)

	labels := map[string]string{"spec": "ETH1", "apiInterface": "jsonrpc"}
	require.Equal(t, float64(12345), testutil.ToFloat64(mgr.blockMetric.WithLabelValues(labels)))
}

func TestProviderMetricsManager_SetLatestBlock_NilSafe(t *testing.T) {
	var mgr *ProviderMetricsManager
	require.NotPanics(t, func() { mgr.SetLatestBlock("ETH1", "jsonrpc", "ep1", 100) })
}

func TestProviderMetricsManager_FetchBlockCounters(t *testing.T) {
	mgr := newProviderMetricsManagerForTest(t)

	mgr.SetLatestBlockFetchError("ETH1", "jsonrpc")
	mgr.SetLatestBlockFetchError("ETH1", "jsonrpc")
	mgr.SetLatestBlockFetchSuccess("ETH1", "jsonrpc")
	mgr.SetSpecificBlockFetchError("ETH1", "jsonrpc")
	mgr.SetSpecificBlockFetchSuccess("ETH1", "jsonrpc")
	mgr.SetSpecificBlockFetchSuccess("ETH1", "jsonrpc")

	require.Equal(t, float64(2), testutil.ToFloat64(mgr.fetchLatestFailedMetric.WithLabelValues("ETH1", "jsonrpc")))
	require.Equal(t, float64(1), testutil.ToFloat64(mgr.fetchLatestSuccessMetric.WithLabelValues("ETH1", "jsonrpc")))
	require.Equal(t, float64(1), testutil.ToFloat64(mgr.fetchBlockFailedMetric.WithLabelValues("ETH1", "jsonrpc")))
	require.Equal(t, float64(2), testutil.ToFloat64(mgr.fetchBlockSuccessMetric.WithLabelValues("ETH1", "jsonrpc")))
}

func TestProviderMetricsManager_FetchCounters_NilSafe(t *testing.T) {
	var mgr *ProviderMetricsManager
	require.NotPanics(t, func() {
		mgr.SetLatestBlockFetchError("ETH1", "jsonrpc")
		mgr.SetLatestBlockFetchSuccess("ETH1", "jsonrpc")
		mgr.SetSpecificBlockFetchError("ETH1", "jsonrpc")
		mgr.SetSpecificBlockFetchSuccess("ETH1", "jsonrpc")
	})
}

func TestProviderMetricsManager_HealthCheckStatus(t *testing.T) {
	mgr := newProviderMetricsManagerForTest(t)

	mgr.UpdateHealthCheckStatus(true)
	require.Equal(t, float64(1), testutil.ToFloat64(mgr.endpointsHealthChecksOkMetric))

	mgr.UpdateHealthCheckStatus(false)
	require.Equal(t, float64(0), testutil.ToFloat64(mgr.endpointsHealthChecksOkMetric))
}

func TestProviderMetricsManager_HealthCheckBreakdown(t *testing.T) {
	mgr := newProviderMetricsManagerForTest(t)

	mgr.UpdateHealthcheckStatusBreakdown("ETH1", "jsonrpc", true)
	require.Equal(t, float64(1), testutil.ToFloat64(mgr.endpointsHealthChecksBreakdownMetric.WithLabelValues("ETH1", "jsonrpc")))

	mgr.UpdateHealthcheckStatusBreakdown("ETH1", "jsonrpc", false)
	require.Equal(t, float64(0), testutil.ToFloat64(mgr.endpointsHealthChecksBreakdownMetric.WithLabelValues("ETH1", "jsonrpc")))
}

func TestProviderMetricsManager_DisabledEnabledChain(t *testing.T) {
	mgr := newProviderMetricsManagerForTest(t)

	mgr.SetDisabledChain("ETH1", "jsonrpc")
	require.Equal(t, float64(1), testutil.ToFloat64(mgr.disabledChainsMetric.WithLabelValues("ETH1", "jsonrpc")))

	mgr.SetEnabledChain("ETH1", "jsonrpc")
	require.Equal(t, float64(0), testutil.ToFloat64(mgr.disabledChainsMetric.WithLabelValues("ETH1", "jsonrpc")))
}

func TestProviderMetricsManager_SetBlock(t *testing.T) {
	mgr := newProviderMetricsManagerForTest(t)
	mgr.SetBlock(999)

	labels := map[string]string{"spec": "lava", "apiInterface": "lava"}
	require.Equal(t, float64(999), testutil.ToFloat64(mgr.blockMetric.WithLabelValues(labels)))
}

func TestProviderMetricsManager_FrozenJailStatus(t *testing.T) {
	mgr := newProviderMetricsManagerForTest(t)

	mgr.SetFrozenStatus("ETH1", true)
	require.Equal(t, float64(1), testutil.ToFloat64(mgr.frozenStatusMetric.WithLabelValues("ETH1")))
	mgr.SetFrozenStatus("ETH1", false)
	require.Equal(t, float64(0), testutil.ToFloat64(mgr.frozenStatusMetric.WithLabelValues("ETH1")))

	mgr.SetJailStatus("ETH1", true)
	require.Equal(t, float64(1), testutil.ToFloat64(mgr.jailStatusMetric.WithLabelValues("ETH1")))

	mgr.SetJailedCount("ETH1", 5)
	require.Equal(t, float64(5), testutil.ToFloat64(mgr.jailedCountMetric.WithLabelValues("ETH1")))
}

func TestProviderMetricsManager_VirtualEpoch(t *testing.T) {
	mgr := newProviderMetricsManagerForTest(t)
	mgr.SetVirtualEpoch(42)
	require.Equal(t, float64(42), testutil.ToFloat64(mgr.virtualEpochMetric.WithLabelValues("lava")))
}

func TestProviderMetricsManager_AddProviderMetrics(t *testing.T) {
	mgr := newProviderMetricsManagerForTest(t)

	pm := mgr.AddProviderMetrics("ETH1", "jsonrpc", "ep1")
	require.NotNil(t, pm)
	require.Equal(t, "ETH1", pm.specID)
	require.Equal(t, "jsonrpc", pm.apiInterface)

	// Second call returns the same instance
	pm2 := mgr.AddProviderMetrics("ETH1", "jsonrpc", "ep1")
	require.Equal(t, pm, pm2)
}

func TestProviderMetricsManager_AddProviderMetrics_NilSafe(t *testing.T) {
	var mgr *ProviderMetricsManager
	require.Nil(t, mgr.AddProviderMetrics("ETH1", "jsonrpc", "ep1"))
}

func TestProviderMetricsManager_SetProviderLatency(t *testing.T) {
	mgr := newProviderMetricsManagerForTest(t)
	mgr.AddProviderMetrics("LATSPEC", "rest", "ep1")

	require.NotPanics(t, func() { mgr.SetProviderLatency("LATSPEC", "rest", 42.0) })
	require.NotPanics(t, func() { mgr.SetProviderEndToEndLatency("LATSPEC", "rest", 99.0) })
}

func TestProviderMetricsManager_AddPayment(t *testing.T) {
	mgr := newProviderMetricsManagerForTest(t)
	mgr.AddProviderMetrics("PAYSPEC", "grpc", "ep1")

	mgr.AddPayment("PAYSPEC", 100)
	// AddPayment runs in a goroutine internally, just verify no panic
	require.NotPanics(t, func() { mgr.AddPayment("UNKNOWN", 50) })
}
