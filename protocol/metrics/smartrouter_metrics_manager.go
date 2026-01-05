package metrics

import (
	"errors"
	"net/http"
	"sync"
	"sync/atomic"

	"github.com/lavanet/lava/v5/utils"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Default latency histogram buckets in milliseconds
// Covers range from 1ms to 30s with good granularity for RPC latencies
var DefaultLatencyBuckets = []float64{1, 2, 5, 10, 25, 50, 100, 250, 500, 1000, 2500, 5000, 10000, 30000}

// SmartRouterMetricsManager manages metrics for the RPC Smart Router.
// It follows the post-unification metric spec with:
// - Endpoint-scoped metrics: lava_rpc_endpoint_*
// - Router-scoped metrics: lava_rpcsmartrouter_*
type SmartRouterMetricsManager struct {
	// Endpoint-scoped metrics (labels: spec, apiInterface, endpoint_id)
	endpointOverallHealth             *MappedLabelsGaugeVec
	endpointTotalRelaysServiced       *MappedLabelsCounterVec
	endpointEndToEndLatency           *MappedLabelsGaugeVec
	endpointTotalRelaysPerFunction    *MappedLabelsCounterVec
	endpointRequestLatencyPerFunction *MappedLabelsGaugeVec
	endpointTotalErroredPerFunction   *MappedLabelsCounterVec
	endpointRequestsInFlightPerFunc   *MappedLabelsGaugeVec
	endpointTotalErrored              *MappedLabelsCounterVec
	endpointFetchLatestFails          *MappedLabelsCounterVec
	endpointFetchBlockFails           *MappedLabelsCounterVec
	endpointFetchLatestSuccess        *MappedLabelsCounterVec
	endpointFetchBlockSuccess         *MappedLabelsCounterVec
	endpointOverallHealthBreakdown    *MappedLabelsGaugeVec

	// Histogram metrics for latency distribution (for comparison with gauge metrics)
	// These allow calculating percentiles (p50, p95, p99) and averages over time
	endpointEndToEndLatencyHistogram           *prometheus.HistogramVec
	endpointRequestLatencyPerFunctionHistogram *prometheus.HistogramVec

	// Router-scoped metrics (labels: spec, apiInterface)
	routerLatestBlock *prometheus.GaugeVec
	routerQoS         *prometheus.GaugeVec

	// Optional info metric (labels: spec, apiInterface, endpoint_id, endpoint_url)
	endpointInfo *MappedLabelsGaugeVec

	// Internal state
	lock                    sync.RWMutex
	endpointsHealthChecksOk uint64

	// Per-endpoint metrics storage for function-level tracking
	endpointMetrics map[string]*EndpointMetrics
}

// EndpointMetrics holds per-endpoint metrics state for function-level tracking
type EndpointMetrics struct {
	spec         string
	apiInterface string
	endpointID   string
	lock         sync.Mutex
}

// SmartRouterMetricsManagerOptions contains configuration for the metrics manager
type SmartRouterMetricsManagerOptions struct {
	NetworkAddress  string
	StartHTTPServer bool // If false, only register metrics (for use alongside ConsumerMetricsManager)
}

// NewSmartRouterMetricsManager creates a new SmartRouterMetricsManager instance
func NewSmartRouterMetricsManager(options SmartRouterMetricsManagerOptions) *SmartRouterMetricsManager {
	if options.NetworkAddress == DisabledFlagOption {
		utils.LavaFormatWarning("prometheus endpoint inactive, option is disabled", nil)
		return nil
	}

	// =========================================================================
	// Endpoint-scoped metrics (lava_rpc_endpoint_*)
	// Required labels: spec, apiInterface, endpoint_id
	// =========================================================================

	endpointLabels := []string{"spec", "apiInterface", "endpoint_id"}
	endpointFunctionLabels := []string{"spec", "apiInterface", "endpoint_id", "function"}

	endpointOverallHealth := NewMappedLabelsGaugeVec(MappedLabelsMetricOpts{
		Name:   "lava_rpc_endpoint_overall_health",
		Help:   "Health status of this RPC endpoint (1=healthy, 0=unhealthy).",
		Labels: endpointLabels,
	})

	endpointTotalRelaysServiced := NewMappedLabelsCounterVec(MappedLabelsMetricOpts{
		Name:   "lava_rpc_endpoint_total_relays_serviced",
		Help:   "Total number of relays successfully serviced by this RPC endpoint.",
		Labels: endpointLabels,
	})

	endpointEndToEndLatency := NewMappedLabelsGaugeVec(MappedLabelsMetricOpts{
		Name:   "lava_rpc_endpoint_end_to_end_latency_milliseconds",
		Help:   "Latest measured end-to-end relay latency for this RPC endpoint in milliseconds.",
		Labels: endpointLabels,
	})

	endpointTotalRelaysPerFunction := NewMappedLabelsCounterVec(MappedLabelsMetricOpts{
		Name:   "lava_rpc_endpoint_total_relays_serviced_per_function",
		Help:   "Total number of relays serviced by this RPC endpoint for each function.",
		Labels: endpointFunctionLabels,
	})

	endpointRequestLatencyPerFunction := NewMappedLabelsGaugeVec(MappedLabelsMetricOpts{
		Name:   "lava_rpc_endpoint_request_latency_per_function_milliseconds",
		Help:   "Latest measured request latency by function for this RPC endpoint in milliseconds.",
		Labels: endpointFunctionLabels,
	})

	endpointTotalErroredPerFunction := NewMappedLabelsCounterVec(MappedLabelsMetricOpts{
		Name:   "lava_rpc_endpoint_total_relays_errored_per_function",
		Help:   "Total number of relays that ended in error by function for this RPC endpoint.",
		Labels: endpointFunctionLabels,
	})

	endpointRequestsInFlightPerFunc := NewMappedLabelsGaugeVec(MappedLabelsMetricOpts{
		Name:   "lava_rpc_endpoint_requests_in_flight_per_function",
		Help:   "Current number of in-flight relays by function for this RPC endpoint.",
		Labels: endpointFunctionLabels,
	})

	endpointTotalErrored := NewMappedLabelsCounterVec(MappedLabelsMetricOpts{
		Name:   "lava_rpc_endpoint_total_errored",
		Help:   "Total number of errored relays for this RPC endpoint.",
		Labels: endpointLabels,
	})

	endpointFetchLatestFails := NewMappedLabelsCounterVec(MappedLabelsMetricOpts{
		Name:   "lava_rpc_endpoint_fetch_latest_fails",
		Help:   "Total failed latest-block fetch operations for this RPC endpoint.",
		Labels: endpointLabels,
	})

	endpointFetchBlockFails := NewMappedLabelsCounterVec(MappedLabelsMetricOpts{
		Name:   "lava_rpc_endpoint_fetch_block_fails",
		Help:   "Total failed specific-block fetch operations for this RPC endpoint.",
		Labels: endpointLabels,
	})

	endpointFetchLatestSuccess := NewMappedLabelsCounterVec(MappedLabelsMetricOpts{
		Name:   "lava_rpc_endpoint_fetch_latest_success",
		Help:   "Total successful latest-block fetch operations for this RPC endpoint.",
		Labels: endpointLabels,
	})

	// Additional metric for symmetry (from spec)
	endpointFetchBlockSuccess := NewMappedLabelsCounterVec(MappedLabelsMetricOpts{
		Name:   "lava_rpc_endpoint_fetch_block_success",
		Help:   "Total successful specific-block fetch operations for this RPC endpoint.",
		Labels: endpointLabels,
	})

	endpointOverallHealthBreakdown := NewMappedLabelsGaugeVec(MappedLabelsMetricOpts{
		Name:   "lava_rpc_endpoint_overall_health_breakdown",
		Help:   "Per-endpoint health status breakdown (1=healthy, 0=unhealthy).",
		Labels: endpointLabels,
	})

	// Optional info metric (recommended for URL visibility without raising cardinality)
	endpointInfoLabels := []string{"spec", "apiInterface", "endpoint_id", "endpoint_url"}
	endpointInfo := NewMappedLabelsGaugeVec(MappedLabelsMetricOpts{
		Name:   "lava_rpc_endpoint_info",
		Help:   "Static metadata mapping for RPC endpoint identity.",
		Labels: endpointInfoLabels,
	})

	// =========================================================================
	// Histogram metrics for latency distribution
	// These complement the gauge metrics and allow percentile calculations
	// =========================================================================

	endpointEndToEndLatencyHistogram := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "lava_rpc_endpoint_end_to_end_latency_milliseconds_histogram",
		Help:    "Distribution of end-to-end relay latency for this RPC endpoint in milliseconds. Use histogram_quantile() for percentiles.",
		Buckets: DefaultLatencyBuckets,
	}, []string{"spec", "apiInterface", "endpoint_id"})

	endpointRequestLatencyPerFunctionHistogram := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "lava_rpc_endpoint_request_latency_per_function_milliseconds_histogram",
		Help:    "Distribution of request latency by function for this RPC endpoint in milliseconds. Use histogram_quantile() for percentiles.",
		Buckets: DefaultLatencyBuckets,
	}, []string{"spec", "apiInterface", "endpoint_id", "function"})

	// =========================================================================
	// Router-scoped metrics (lava_rpcsmartrouter_*)
	// Required labels: spec, apiInterface
	// =========================================================================

	routerLatestBlock := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "lava_rpcsmartrouter_latest_block",
		Help: "Latest block known by the RPC smart router for this chain/interface.",
	}, []string{"spec", "apiInterface"})

	routerQoS := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "lava_rpcsmartrouter_qos",
		Help: "Current smart-router QoS score by metric (availability/sync/latency).",
	}, []string{"spec", "apiInterface", "qos_metric"})

	// Register router-scoped metrics (endpoint-scoped metrics are auto-registered by NewMappedLabels*)
	registerMetric := func(c prometheus.Collector) {
		if err := prometheus.Register(c); err != nil {
			are := &prometheus.AlreadyRegisteredError{}
			if !errors.As(err, are) {
				panic(err)
			}
			utils.LavaFormatDebug("Prometheus metric already registered, reusing existing collector", utils.LogAttr("error", err))
		}
	}

	registerMetric(routerLatestBlock)
	registerMetric(routerQoS)
	registerMetric(endpointEndToEndLatencyHistogram)
	registerMetric(endpointRequestLatencyPerFunctionHistogram)

	manager := &SmartRouterMetricsManager{
		// Endpoint-scoped metrics
		endpointOverallHealth:             endpointOverallHealth,
		endpointTotalRelaysServiced:       endpointTotalRelaysServiced,
		endpointEndToEndLatency:           endpointEndToEndLatency,
		endpointTotalRelaysPerFunction:    endpointTotalRelaysPerFunction,
		endpointRequestLatencyPerFunction: endpointRequestLatencyPerFunction,
		endpointTotalErroredPerFunction:   endpointTotalErroredPerFunction,
		endpointRequestsInFlightPerFunc:   endpointRequestsInFlightPerFunc,
		endpointTotalErrored:              endpointTotalErrored,
		endpointFetchLatestFails:          endpointFetchLatestFails,
		endpointFetchBlockFails:           endpointFetchBlockFails,
		endpointFetchLatestSuccess:        endpointFetchLatestSuccess,
		endpointFetchBlockSuccess:         endpointFetchBlockSuccess,
		endpointOverallHealthBreakdown:    endpointOverallHealthBreakdown,
		endpointInfo:                      endpointInfo,

		// Histogram metrics for latency distribution
		endpointEndToEndLatencyHistogram:           endpointEndToEndLatencyHistogram,
		endpointRequestLatencyPerFunctionHistogram: endpointRequestLatencyPerFunctionHistogram,

		// Router-scoped metrics
		routerLatestBlock: routerLatestBlock,
		routerQoS:         routerQoS,

		// Internal state
		endpointsHealthChecksOk: 1,
		endpointMetrics:         make(map[string]*EndpointMetrics),
	}

	// Only start HTTP server if requested (to avoid conflicts with ConsumerMetricsManager)
	// When running alongside ConsumerMetricsManager, set StartHTTPServer=false
	// The metrics are still registered with Prometheus and will be served by the existing server
	if options.StartHTTPServer && options.NetworkAddress != "" {
		// Set up HTTP handlers
		http.Handle("/metrics", promhttp.Handler())

		overallHealthHandler := func(w http.ResponseWriter, r *http.Request) {
			statusCode := http.StatusOK
			message := "Health status OK"
			if atomic.LoadUint64(&manager.endpointsHealthChecksOk) == 0 {
				statusCode = http.StatusServiceUnavailable
				message = "Unhealthy"
			}

			w.WriteHeader(statusCode)
			w.Write([]byte(message))
		}

		// Backward compatibility - old path for health check alongside new path
		http.HandleFunc("/metrics/overall-health", overallHealthHandler)
		http.HandleFunc("/metrics/health-overall", overallHealthHandler)

		go func() {
			utils.LavaFormatInfo("prometheus endpoint listening", utils.Attribute{Key: "Listen Address", Value: options.NetworkAddress})
			http.ListenAndServe(options.NetworkAddress, nil)
		}()
	}

	return manager
}

// =============================================================================
// Endpoint-scoped metric setters
// =============================================================================

// SetEndpointOverallHealth sets the health status for an RPC endpoint
func (m *SmartRouterMetricsManager) SetEndpointOverallHealth(spec, apiInterface, endpointID string, healthy bool) {
	if m == nil {
		return
	}
	value := 0.0
	if healthy {
		value = 1.0
	}
	labels := map[string]string{"spec": spec, "apiInterface": apiInterface, "endpoint_id": endpointID}
	m.endpointOverallHealth.WithLabelValues(labels).Set(value)
}

// AddEndpointRelayServiced increments the total relays serviced counter for an endpoint
func (m *SmartRouterMetricsManager) AddEndpointRelayServiced(spec, apiInterface, endpointID string) {
	if m == nil {
		return
	}
	labels := map[string]string{"spec": spec, "apiInterface": apiInterface, "endpoint_id": endpointID}
	m.endpointTotalRelaysServiced.WithLabelValues(labels).Inc()
}

// SetEndpointEndToEndLatency sets the end-to-end latency for an endpoint in milliseconds (gauge - latest value only)
func (m *SmartRouterMetricsManager) SetEndpointEndToEndLatency(spec, apiInterface, endpointID string, latencyMs float64) {
	if m == nil {
		return
	}
	labels := map[string]string{"spec": spec, "apiInterface": apiInterface, "endpoint_id": endpointID}
	m.endpointEndToEndLatency.WithLabelValues(labels).Set(latencyMs)
}

// ObserveEndpointEndToEndLatency records a latency observation in the histogram for percentile calculations
func (m *SmartRouterMetricsManager) ObserveEndpointEndToEndLatency(spec, apiInterface, endpointID string, latencyMs float64) {
	if m == nil {
		return
	}
	m.endpointEndToEndLatencyHistogram.WithLabelValues(spec, apiInterface, endpointID).Observe(latencyMs)
}

// AddEndpointRelayServicedPerFunction increments the relays serviced counter for a specific function
func (m *SmartRouterMetricsManager) AddEndpointRelayServicedPerFunction(spec, apiInterface, endpointID, function string) {
	if m == nil {
		return
	}
	labels := map[string]string{"spec": spec, "apiInterface": apiInterface, "endpoint_id": endpointID, "function": function}
	m.endpointTotalRelaysPerFunction.WithLabelValues(labels).Inc()
}

// SetEndpointRequestLatencyPerFunction sets the request latency for a specific function in milliseconds (gauge - latest value only)
func (m *SmartRouterMetricsManager) SetEndpointRequestLatencyPerFunction(spec, apiInterface, endpointID, function string, latencyMs float64) {
	if m == nil {
		return
	}
	labels := map[string]string{"spec": spec, "apiInterface": apiInterface, "endpoint_id": endpointID, "function": function}
	m.endpointRequestLatencyPerFunction.WithLabelValues(labels).Set(latencyMs)
}

// ObserveEndpointRequestLatencyPerFunction records a latency observation in the histogram for percentile calculations
func (m *SmartRouterMetricsManager) ObserveEndpointRequestLatencyPerFunction(spec, apiInterface, endpointID, function string, latencyMs float64) {
	if m == nil {
		return
	}
	m.endpointRequestLatencyPerFunctionHistogram.WithLabelValues(spec, apiInterface, endpointID, function).Observe(latencyMs)
}

// AddEndpointRelayErroredPerFunction increments the error counter for a specific function
func (m *SmartRouterMetricsManager) AddEndpointRelayErroredPerFunction(spec, apiInterface, endpointID, function string) {
	if m == nil {
		return
	}
	labels := map[string]string{"spec": spec, "apiInterface": apiInterface, "endpoint_id": endpointID, "function": function}
	m.endpointTotalErroredPerFunction.WithLabelValues(labels).Inc()
}

// AddEndpointInFlightRequest increments the in-flight requests counter for a specific function
func (m *SmartRouterMetricsManager) AddEndpointInFlightRequest(spec, apiInterface, endpointID, function string) {
	if m == nil {
		return
	}
	labels := map[string]string{"spec": spec, "apiInterface": apiInterface, "endpoint_id": endpointID, "function": function}
	m.endpointRequestsInFlightPerFunc.WithLabelValues(labels).Add(1)
}

// SubEndpointInFlightRequest decrements the in-flight requests counter for a specific function
func (m *SmartRouterMetricsManager) SubEndpointInFlightRequest(spec, apiInterface, endpointID, function string) {
	if m == nil {
		return
	}
	labels := map[string]string{"spec": spec, "apiInterface": apiInterface, "endpoint_id": endpointID, "function": function}
	m.endpointRequestsInFlightPerFunc.WithLabelValues(labels).Sub(1)
}

// AddEndpointError increments the total error counter for an endpoint
func (m *SmartRouterMetricsManager) AddEndpointError(spec, apiInterface, endpointID string) {
	if m == nil {
		return
	}
	labels := map[string]string{"spec": spec, "apiInterface": apiInterface, "endpoint_id": endpointID}
	m.endpointTotalErrored.WithLabelValues(labels).Inc()
}

// AddEndpointFetchLatestFail increments the failed latest-block fetch counter
func (m *SmartRouterMetricsManager) AddEndpointFetchLatestFail(spec, apiInterface, endpointID string) {
	if m == nil {
		return
	}
	labels := map[string]string{"spec": spec, "apiInterface": apiInterface, "endpoint_id": endpointID}
	m.endpointFetchLatestFails.WithLabelValues(labels).Inc()
}

// AddEndpointFetchBlockFail increments the failed specific-block fetch counter
func (m *SmartRouterMetricsManager) AddEndpointFetchBlockFail(spec, apiInterface, endpointID string) {
	if m == nil {
		return
	}
	labels := map[string]string{"spec": spec, "apiInterface": apiInterface, "endpoint_id": endpointID}
	m.endpointFetchBlockFails.WithLabelValues(labels).Inc()
}

// AddEndpointFetchLatestSuccess increments the successful latest-block fetch counter
func (m *SmartRouterMetricsManager) AddEndpointFetchLatestSuccess(spec, apiInterface, endpointID string) {
	if m == nil {
		return
	}
	labels := map[string]string{"spec": spec, "apiInterface": apiInterface, "endpoint_id": endpointID}
	m.endpointFetchLatestSuccess.WithLabelValues(labels).Inc()
}

// AddEndpointFetchBlockSuccess increments the successful specific-block fetch counter
func (m *SmartRouterMetricsManager) AddEndpointFetchBlockSuccess(spec, apiInterface, endpointID string) {
	if m == nil {
		return
	}
	labels := map[string]string{"spec": spec, "apiInterface": apiInterface, "endpoint_id": endpointID}
	m.endpointFetchBlockSuccess.WithLabelValues(labels).Inc()
}

// SetEndpointHealthBreakdown sets the health status breakdown for a specific endpoint
func (m *SmartRouterMetricsManager) SetEndpointHealthBreakdown(spec, apiInterface, endpointID string, healthy bool) {
	if m == nil {
		return
	}
	value := 0.0
	if healthy {
		value = 1.0
	}
	labels := map[string]string{"spec": spec, "apiInterface": apiInterface, "endpoint_id": endpointID}
	m.endpointOverallHealthBreakdown.WithLabelValues(labels).Set(value)
}

// SetEndpointInfo sets the static metadata mapping for an endpoint (for URL visibility)
func (m *SmartRouterMetricsManager) SetEndpointInfo(spec, apiInterface, endpointID, endpointURL string) {
	if m == nil {
		return
	}
	labels := map[string]string{"spec": spec, "apiInterface": apiInterface, "endpoint_id": endpointID, "endpoint_url": endpointURL}
	m.endpointInfo.WithLabelValues(labels).Set(1)
}

// =============================================================================
// Router-scoped metric setters
// =============================================================================

// SetRouterLatestBlock sets the latest block known by the smart router for a chain/interface
func (m *SmartRouterMetricsManager) SetRouterLatestBlock(spec, apiInterface string, block int64) {
	if m == nil {
		return
	}
	m.routerLatestBlock.WithLabelValues(spec, apiInterface).Set(float64(block))
}

// SetRouterQoS sets QoS scores for the smart router
func (m *SmartRouterMetricsManager) SetRouterQoS(spec, apiInterface string, availability, sync, latency float64) {
	if m == nil {
		return
	}
	m.routerQoS.WithLabelValues(spec, apiInterface, AvailabilityLabel).Set(availability)
	m.routerQoS.WithLabelValues(spec, apiInterface, SyncLabel).Set(sync)
	m.routerQoS.WithLabelValues(spec, apiInterface, LatencyLabel).Set(latency)
}

// SetRouterQoSAvailability sets the availability QoS score
func (m *SmartRouterMetricsManager) SetRouterQoSAvailability(spec, apiInterface string, availability float64) {
	if m == nil {
		return
	}
	m.routerQoS.WithLabelValues(spec, apiInterface, AvailabilityLabel).Set(availability)
}

// SetRouterQoSSync sets the sync QoS score
func (m *SmartRouterMetricsManager) SetRouterQoSSync(spec, apiInterface string, sync float64) {
	if m == nil {
		return
	}
	m.routerQoS.WithLabelValues(spec, apiInterface, SyncLabel).Set(sync)
}

// SetRouterQoSLatency sets the latency QoS score
func (m *SmartRouterMetricsManager) SetRouterQoSLatency(spec, apiInterface string, latency float64) {
	if m == nil {
		return
	}
	m.routerQoS.WithLabelValues(spec, apiInterface, LatencyLabel).Set(latency)
}

// =============================================================================
// Health check methods
// =============================================================================

// UpdateOverallHealthStatus updates the overall health check status for the smart router
func (m *SmartRouterMetricsManager) UpdateOverallHealthStatus(healthy bool) {
	if m == nil {
		return
	}
	value := uint64(0)
	if healthy {
		value = 1
	}
	atomic.StoreUint64(&m.endpointsHealthChecksOk, value)
}

// IsHealthy returns true if the smart router is healthy
func (m *SmartRouterMetricsManager) IsHealthy() bool {
	if m == nil {
		return true // Default to healthy if metrics manager is disabled
	}
	return atomic.LoadUint64(&m.endpointsHealthChecksOk) == 1
}

// =============================================================================
// Endpoint registration and management
// =============================================================================

// RegisterEndpoint registers an endpoint and sets its info metric
func (m *SmartRouterMetricsManager) RegisterEndpoint(spec, apiInterface, endpointID, endpointURL string) {
	if m == nil {
		return
	}
	m.lock.Lock()
	defer m.lock.Unlock()

	key := spec + apiInterface + endpointID
	if _, exists := m.endpointMetrics[key]; !exists {
		m.endpointMetrics[key] = &EndpointMetrics{
			spec:         spec,
			apiInterface: apiInterface,
			endpointID:   endpointID,
		}
	}

	// Set the info metric to provide URL visibility
	m.SetEndpointInfo(spec, apiInterface, endpointID, endpointURL)
}

// =============================================================================
// Convenience methods for recording relay results
// =============================================================================

// RecordRelaySuccess records a successful relay to an endpoint
// Updates both gauge metrics (latest value) and histogram metrics (for percentile calculations)
func (m *SmartRouterMetricsManager) RecordRelaySuccess(spec, apiInterface, endpointID, function string, latencyMs float64) {
	if m == nil {
		return
	}
	m.AddEndpointRelayServiced(spec, apiInterface, endpointID)
	m.AddEndpointRelayServicedPerFunction(spec, apiInterface, endpointID, function)

	// Update gauge metrics (latest value)
	m.SetEndpointRequestLatencyPerFunction(spec, apiInterface, endpointID, function, latencyMs)
	m.SetEndpointEndToEndLatency(spec, apiInterface, endpointID, latencyMs)

	// Update histogram metrics (for percentile calculations)
	m.ObserveEndpointRequestLatencyPerFunction(spec, apiInterface, endpointID, function, latencyMs)
	m.ObserveEndpointEndToEndLatency(spec, apiInterface, endpointID, latencyMs)
}

// RecordRelayError records a failed relay to an endpoint
func (m *SmartRouterMetricsManager) RecordRelayError(spec, apiInterface, endpointID, function string) {
	if m == nil {
		return
	}
	m.AddEndpointError(spec, apiInterface, endpointID)
	m.AddEndpointRelayErroredPerFunction(spec, apiInterface, endpointID, function)
}

// =============================================================================
// Reset methods
// =============================================================================

// ResetEndpointMetrics resets all metrics for a specific endpoint
func (m *SmartRouterMetricsManager) ResetEndpointMetrics(spec, apiInterface, endpointID string) {
	if m == nil {
		return
	}
	labels := map[string]string{"spec": spec, "apiInterface": apiInterface, "endpoint_id": endpointID}
	m.endpointOverallHealth.WithLabelValues(labels).Set(0)
	m.endpointEndToEndLatency.WithLabelValues(labels).Set(0)
	m.endpointOverallHealthBreakdown.WithLabelValues(labels).Set(0)
}

// =============================================================================
// Compatibility methods for ConsumerMetricsManager interface
// These methods allow SmartRouterMetricsManager to work with existing code
// that expects ConsumerMetricsManager. The endpointID is derived from the
// provider address or a stable identifier.
// =============================================================================

// SetRelayMetrics records relay metrics (compatible with ConsumerMetricsManager)
// Maps to endpoint-scoped metrics using chainID as spec and provider as endpoint_id
func (m *SmartRouterMetricsManager) SetRelayMetrics(relayMetric *RelayMetrics, err error) {
	if m == nil || relayMetric == nil {
		return
	}

	spec := relayMetric.ChainID
	apiInterface := relayMetric.APIType
	// Use a stable endpoint identifier - for smart router, this could be derived from provider address
	// or a hash of the endpoint URL
	endpointID := "default" // Will be overridden by specific calls with endpoint info
	function := relayMetric.ApiMethod

	if err == nil {
		m.RecordRelaySuccess(spec, apiInterface, endpointID, function, float64(relayMetric.Latency))
	} else {
		m.RecordRelayError(spec, apiInterface, endpointID, function)
	}
}

// SetEndToEndLatencyWithEndpoint records end-to-end latency for a specific endpoint
func (m *SmartRouterMetricsManager) SetEndToEndLatencyWithEndpoint(spec, apiInterface, endpointID string, latencyMs float64) {
	if m == nil {
		return
	}
	m.SetEndpointEndToEndLatency(spec, apiInterface, endpointID, latencyMs)
	m.ObserveEndpointEndToEndLatency(spec, apiInterface, endpointID, latencyMs)
}

// SetRelaySentByNewBatchTickerMetric tracks relays sent by batch ticker (for state machine)
func (m *SmartRouterMetricsManager) SetRelaySentByNewBatchTickerMetric(chainId string, apiInterface string) {
	// This is a router-level metric, not endpoint-specific
	// For now, we don't have a direct equivalent in the new spec
	// Could be added as lava_rpcsmartrouter_batch_ticker_relays if needed
}

// SetVersion sets the protocol version (router-level)
func (m *SmartRouterMetricsManager) SetVersion(version string) {
	// Could add a router-level version metric if needed
	// lava_rpcsmartrouter_version gauge
}

// =============================================================================
// Methods for direct RPC relay tracking
// These are called from rpcsmartrouter_server.go during relay processing
// =============================================================================

// RecordDirectRelayStart records the start of an in-flight request
func (m *SmartRouterMetricsManager) RecordDirectRelayStart(spec, apiInterface, endpointID, function string) {
	if m == nil {
		return
	}
	m.AddEndpointInFlightRequest(spec, apiInterface, endpointID, function)
}

// RecordDirectRelayEnd records the end of an in-flight request and updates metrics
func (m *SmartRouterMetricsManager) RecordDirectRelayEnd(spec, apiInterface, endpointID, function string, latencyMs float64, success bool) {
	if m == nil {
		return
	}
	m.SubEndpointInFlightRequest(spec, apiInterface, endpointID, function)

	if success {
		m.RecordRelaySuccess(spec, apiInterface, endpointID, function, latencyMs)
	} else {
		m.RecordRelayError(spec, apiInterface, endpointID, function)
	}
}

// RecordBlockFetch records block fetch operations (for chain tracker/state poller)
func (m *SmartRouterMetricsManager) RecordBlockFetch(spec, apiInterface, endpointID string, isLatest bool, success bool) {
	if m == nil {
		return
	}

	if isLatest {
		if success {
			m.AddEndpointFetchLatestSuccess(spec, apiInterface, endpointID)
		} else {
			m.AddEndpointFetchLatestFail(spec, apiInterface, endpointID)
		}
	} else {
		if success {
			m.AddEndpointFetchBlockSuccess(spec, apiInterface, endpointID)
		} else {
			m.AddEndpointFetchBlockFail(spec, apiInterface, endpointID)
		}
	}
}
