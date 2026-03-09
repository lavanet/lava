package metrics

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/lavanet/lava/v5/utils"
	pairingtypes "github.com/lavanet/lava/v5/x/pairing/types"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Default latency histogram buckets in milliseconds
// Covers range from 1ms to 30s with good granularity for RPC latencies
var defaultLatencyBuckets = []float64{1, 2, 5, 10, 25, 50, 100, 250, 500, 1000, 2500, 5000, 10000, 30000}

// registerOrReuse registers a Prometheus collector, returning the existing
// collector on AlreadyRegisteredError so callers always hold the instance
// that Prometheus actually exposes.
func registerOrReuse[T prometheus.Collector](c T) T {
	if err := prometheus.Register(c); err != nil {
		are := &prometheus.AlreadyRegisteredError{}
		if errors.As(err, are) {
			if existing, ok := are.ExistingCollector.(T); ok {
				return existing
			}
		}
		panic(err)
	}
	return c
}

// Compile-time check: SmartRouterMetricsManager must satisfy ConsumerMetricsManagerInf.
var _ ConsumerMetricsManagerInf = (*SmartRouterMetricsManager)(nil)

// SmartRouterMetricsManager manages metrics for the RPC Smart Router.
// It follows the post-unification metric spec with:
// - Endpoint-scoped metrics: lava_rpc_endpoint_*
// - Router-scoped metrics: lava_rpcsmartrouter_*
//
// Relay/latency metrics include a `function` label so a single metric covers
// both the per-function breakdown and, via sum aggregation, the endpoint/router
// aggregate — no duplicate metrics needed.
type SmartRouterMetricsManager struct {
	// Endpoint-scoped metrics (labels: spec, apiInterface, endpoint_id, function)
	endpointTotalRelaysServiced *MappedLabelsCounterVec  // lava_rpc_endpoint_total_relays_serviced
	endpointTotalErrored        *MappedLabelsCounterVec  // lava_rpc_endpoint_total_errored
	endpointEndToEndLatency     *prometheus.HistogramVec // lava_rpc_endpoint_end_to_end_latency_milliseconds
	endpointInFlight            *MappedLabelsGaugeVec    // lava_rpc_endpoint_requests_in_flight

	// Endpoint-scoped metrics (labels: spec, apiInterface, endpoint_id)
	endpointOverallHealth      *MappedLabelsGaugeVec   // lava_rpc_endpoint_overall_health
	endpointSelectionScore     *prometheus.GaugeVec    // lava_rpc_endpoint_selection_score {spec, endpoint_id, score_type}
	endpointLatestBlock        *MappedLabelsGaugeVec   // lava_rpc_endpoint_latest_block
	endpointFetchLatestFails   *MappedLabelsCounterVec // lava_rpc_endpoint_fetch_latest_fails
	endpointFetchBlockFails    *MappedLabelsCounterVec // lava_rpc_endpoint_fetch_block_fails
	endpointFetchLatestSuccess *MappedLabelsCounterVec // lava_rpc_endpoint_fetch_latest_success
	endpointFetchBlockSuccess  *MappedLabelsCounterVec // lava_rpc_endpoint_fetch_block_success
	endpointInfo               *MappedLabelsGaugeVec   // lava_rpc_endpoint_info (+endpoint_url label)

	// Router-scoped metrics (labels: spec, apiInterface, function)
	routerTotalRelaysServiced *prometheus.CounterVec   // lava_rpcsmartrouter_total_relays_serviced
	routerTotalErrored        *prometheus.CounterVec   // lava_rpcsmartrouter_total_errored
	routerEndToEndLatency     *prometheus.HistogramVec // lava_rpcsmartrouter_end_to_end_latency_milliseconds

	// Router-scoped scalar metrics (no labels)
	routerOverallHealth prometheus.Gauge

	// Router-scoped metrics (labels: spec, apiInterface)
	routerLatestBlock               *prometheus.GaugeVec
	routerProtocolVersion           *prometheus.GaugeVec
	routerWsConnectionsActive       *prometheus.GaugeVec
	routerWsSubscriptionsTotal      *prometheus.CounterVec
	routerWsSubscriptionErrors      *prometheus.CounterVec
	routerWsSubscriptionDuplicates  *prometheus.CounterVec
	routerWsSubscriptionDisconnects *prometheus.CounterVec

	// Router-scoped cross-validation (labels: spec, apiInterface, method, status, max_participants, agreement_threshold, all_endpoints, agreeing_endpoints)
	routerCrossValidation *prometheus.CounterVec

	// Node error recovery metrics
	nodeErrorsReceived  *prometheus.CounterVec // lava_rpcsmartrouter_node_errors_received
	nodeErrorsRecovered *prometheus.CounterVec // lava_rpcsmartrouter_node_errors_recovered

	// Internal state
	lock                    sync.RWMutex
	endpointsHealthChecksOk uint64
	endpointMetrics         map[string]*EndpointMetrics
	urlToProviderName       map[string]string // maps endpoint URL → provider name for metric label resolution
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
	// labels: spec, apiInterface, endpoint_id, function
	// Aggregate with sum by (...) to drop labels you don't need.
	// =========================================================================

	endpointFunctionLabels := []string{"spec", "apiInterface", "endpoint_id", "function"}
	endpointLabels := []string{"spec", "apiInterface", "endpoint_id"}

	endpointTotalRelaysServiced := NewMappedLabelsCounterVec(MappedLabelsMetricOpts{
		Name:   "lava_rpc_endpoint_total_relays_serviced",
		Help:   "Total relays successfully serviced by this RPC endpoint, by function.",
		Labels: endpointFunctionLabels,
	})

	endpointTotalErrored := NewMappedLabelsCounterVec(MappedLabelsMetricOpts{
		Name:   "lava_rpc_endpoint_total_errored",
		Help:   "Total errored relays for this RPC endpoint, by function.",
		Labels: endpointFunctionLabels,
	})

	endpointInFlight := NewMappedLabelsGaugeVec(MappedLabelsMetricOpts{
		Name:   "lava_rpc_endpoint_requests_in_flight",
		Help:   "Current number of in-flight relays for this RPC endpoint, by function.",
		Labels: endpointFunctionLabels,
	})

	endpointOverallHealth := NewMappedLabelsGaugeVec(MappedLabelsMetricOpts{
		Name:   "lava_rpc_endpoint_overall_health",
		Help:   "Health status of this RPC endpoint (1=healthy, 0=unhealthy).",
		Labels: endpointLabels,
	})

	endpointSelectionScore := registerOrReuse(prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "lava_rpc_endpoint_selection_score",
		Help: "Latest selection score for each RPC endpoint by score type (availability/latency/sync/stake/composite).",
	}, []string{"spec", "endpoint_id", "score_type"}))

	endpointLatestBlock := NewMappedLabelsGaugeVec(MappedLabelsMetricOpts{
		Name:   "lava_rpc_endpoint_latest_block",
		Help:   "Latest block known by this RPC endpoint.",
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

	endpointFetchBlockSuccess := NewMappedLabelsCounterVec(MappedLabelsMetricOpts{
		Name:   "lava_rpc_endpoint_fetch_block_success",
		Help:   "Total successful specific-block fetch operations for this RPC endpoint.",
		Labels: endpointLabels,
	})

	// Info metric — endpoint_id is the provider name; endpoint_url carries the raw URL for reference
	endpointInfo := NewMappedLabelsGaugeVec(MappedLabelsMetricOpts{
		Name:   "lava_rpc_endpoint_info",
		Help:   "Static metadata mapping for RPC endpoint identity.",
		Labels: []string{"spec", "apiInterface", "endpoint_id", "endpoint_url"},
	})

	// =========================================================================
	// Endpoint-scoped histogram (labels: spec, apiInterface, endpoint_id, function)
	// =========================================================================

	endpointEndToEndLatency := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "lava_rpc_endpoint_end_to_end_latency_milliseconds",
		Help:    "Distribution of end-to-end relay latency for this RPC endpoint by function in milliseconds. Use histogram_quantile() for percentiles.",
		Buckets: defaultLatencyBuckets,
	}, endpointFunctionLabels)

	// =========================================================================
	// Router-scoped metrics (lava_rpcsmartrouter_*)
	// labels: spec, apiInterface, function  — aggregate with sum by to drop function
	// =========================================================================

	routerFunctionLabels := []string{"spec", "apiInterface", "function"}

	routerTotalRelaysServiced := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "lava_rpcsmartrouter_total_relays_serviced",
		Help: "Total relays successfully serviced by the smart router, by function.",
	}, routerFunctionLabels)

	routerTotalErrored := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "lava_rpcsmartrouter_total_errored",
		Help: "Total errored relays on the smart router, by function.",
	}, routerFunctionLabels)

	routerEndToEndLatency := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "lava_rpcsmartrouter_end_to_end_latency_milliseconds",
		Help:    "Distribution of end-to-end relay latency across all endpoints on the smart router by function in milliseconds. Use histogram_quantile() for percentiles.",
		Buckets: defaultLatencyBuckets,
	}, routerFunctionLabels)

	routerOverallHealth := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "lava_rpcsmartrouter_overall_health",
		Help: "Overall health of the RPC smart router (1=healthy, 0=unhealthy).",
	})
	routerOverallHealth = registerOrReuse(routerOverallHealth)
	routerOverallHealth.Set(1)

	routerLatestBlock := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "lava_rpcsmartrouter_latest_block",
		Help: "Latest block known by the RPC smart router for this chain/interface.",
	}, []string{"spec", "apiInterface"})

	routerProtocolVersion := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "lava_rpcsmartrouter_protocol_version",
		Help: "Protocol version of the RPC smart router encoded as major*1e6+minor*1e3+patch.",
	}, []string{"version"})

	routerWsConnectionsActive := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "lava_rpcsmartrouter_ws_connections_active",
		Help: "Number of currently active WebSocket connections served by the smart router.",
	}, []string{"spec", "apiInterface"})

	routerWsSubscriptionsTotal := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "lava_rpcsmartrouter_ws_subscriptions_total",
		Help: "Total WebSocket subscription requests received by the smart router.",
	}, []string{"spec", "apiInterface"})

	routerWsSubscriptionErrors := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "lava_rpcsmartrouter_ws_subscription_errors_total",
		Help: "Total failed WebSocket subscription requests on the smart router.",
	}, []string{"spec", "apiInterface"})

	routerWsSubscriptionDuplicates := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "lava_rpcsmartrouter_ws_subscription_duplicates_total",
		Help: "Total duplicated WebSocket subscription requests on the smart router.",
	}, []string{"spec", "apiInterface"})

	routerWsSubscriptionDisconnects := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "lava_rpcsmartrouter_ws_subscription_disconnects_total",
		Help: "Total WebSocket subscription disconnects on the smart router.",
	}, []string{"spec", "apiInterface", "reason"})

	routerCrossValidation := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "lava_rpcsmartrouter_cross_validation_requests",
		Help: "Cross-validation request outcomes on the smart router with endpoint details.",
	}, []string{"spec", "apiInterface", "method", "status", "max_participants", "agreement_threshold", "all_endpoints", "agreeing_endpoints"})

	// Node error recovery metrics.
	nodeErrorsReceived := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "lava_rpcsmartrouter_node_errors_received",
		Help: "Total node errors received from RPC endpoints by the smart router.",
	}, []string{"spec", "apiInterface"})

	nodeErrorsRecovered := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "lava_rpcsmartrouter_node_errors_recovered",
		Help: "Total node errors successfully recovered via retry by the smart router.",
	}, []string{"spec", "apiInterface", "attempt"})

	// Register router-scoped and histogram metrics.
	// On duplicate registration, reuse the already-registered collector so the
	// manager fields always point at the collector Prometheus actually exposes.
	routerTotalRelaysServiced = registerOrReuse(routerTotalRelaysServiced)
	routerTotalErrored = registerOrReuse(routerTotalErrored)
	routerEndToEndLatency = registerOrReuse(routerEndToEndLatency)
	routerLatestBlock = registerOrReuse(routerLatestBlock)
	routerProtocolVersion = registerOrReuse(routerProtocolVersion)
	routerWsConnectionsActive = registerOrReuse(routerWsConnectionsActive)
	routerWsSubscriptionsTotal = registerOrReuse(routerWsSubscriptionsTotal)
	routerWsSubscriptionErrors = registerOrReuse(routerWsSubscriptionErrors)
	routerWsSubscriptionDuplicates = registerOrReuse(routerWsSubscriptionDuplicates)
	routerWsSubscriptionDisconnects = registerOrReuse(routerWsSubscriptionDisconnects)
	endpointEndToEndLatency = registerOrReuse(endpointEndToEndLatency)
	routerCrossValidation = registerOrReuse(routerCrossValidation)
	nodeErrorsReceived = registerOrReuse(nodeErrorsReceived)
	nodeErrorsRecovered = registerOrReuse(nodeErrorsRecovered)

	manager := &SmartRouterMetricsManager{
		// Endpoint-scoped (with function)
		endpointTotalRelaysServiced: endpointTotalRelaysServiced,
		endpointTotalErrored:        endpointTotalErrored,
		endpointEndToEndLatency:     endpointEndToEndLatency,
		endpointInFlight:            endpointInFlight,

		// Endpoint-scoped (without function)
		endpointOverallHealth:      endpointOverallHealth,
		endpointSelectionScore:     endpointSelectionScore,
		endpointLatestBlock:        endpointLatestBlock,
		endpointFetchLatestFails:   endpointFetchLatestFails,
		endpointFetchBlockFails:    endpointFetchBlockFails,
		endpointFetchLatestSuccess: endpointFetchLatestSuccess,
		endpointFetchBlockSuccess:  endpointFetchBlockSuccess,
		endpointInfo:               endpointInfo,

		// Router-scoped (with function)
		routerTotalRelaysServiced: routerTotalRelaysServiced,
		routerTotalErrored:        routerTotalErrored,
		routerEndToEndLatency:     routerEndToEndLatency,

		// Router-scoped scalar
		routerOverallHealth: routerOverallHealth,

		// Router-scoped (without function)
		routerLatestBlock:               routerLatestBlock,
		routerProtocolVersion:           routerProtocolVersion,
		routerWsConnectionsActive:       routerWsConnectionsActive,
		routerWsSubscriptionsTotal:      routerWsSubscriptionsTotal,
		routerWsSubscriptionErrors:      routerWsSubscriptionErrors,
		routerWsSubscriptionDuplicates:  routerWsSubscriptionDuplicates,
		routerWsSubscriptionDisconnects: routerWsSubscriptionDisconnects,

		// Router-scoped cross-validation
		routerCrossValidation: routerCrossValidation,

		// Node error recovery
		nodeErrorsReceived:  nodeErrorsReceived,
		nodeErrorsRecovered: nodeErrorsRecovered,

		// Internal state
		endpointsHealthChecksOk: 1,
		endpointMetrics:         make(map[string]*EndpointMetrics),
		urlToProviderName:       make(map[string]string),
	}

	// Only start HTTP server if requested (to avoid conflicts with ConsumerMetricsManager)
	if options.StartHTTPServer && options.NetworkAddress != "" {
		mux := http.NewServeMux()
		mux.Handle("/metrics", promhttp.Handler())

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

		mux.HandleFunc("/metrics/overall-health", overallHealthHandler)
		mux.HandleFunc("/metrics/health-overall", overallHealthHandler)

		go func() {
			utils.LavaFormatInfo("prometheus endpoint listening", utils.Attribute{Key: "Listen Address", Value: options.NetworkAddress})
			if err := http.ListenAndServe(options.NetworkAddress, mux); err != nil {
				utils.LavaFormatError("metrics endpoint failed", err, utils.Attribute{Key: "Listen Address", Value: options.NetworkAddress})
			}
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

// SetEndpointLatestBlock sets the latest block known by an RPC endpoint
func (m *SmartRouterMetricsManager) SetEndpointLatestBlock(spec, apiInterface, endpointID string, block int64) {
	if m == nil {
		return
	}
	labels := map[string]string{"spec": spec, "apiInterface": apiInterface, "endpoint_id": m.resolveProviderName(endpointID)}
	m.endpointLatestBlock.WithLabelValues(labels).Set(float64(block))
}

// AddEndpointRelayServiced increments the relay counter for an endpoint and function
func (m *SmartRouterMetricsManager) AddEndpointRelayServiced(spec, apiInterface, endpointID, function string) {
	if m == nil {
		return
	}
	labels := map[string]string{"spec": spec, "apiInterface": apiInterface, "endpoint_id": endpointID, "function": function}
	m.endpointTotalRelaysServiced.WithLabelValues(labels).Inc()
}

// AddEndpointRelayErrored increments the error counter for an endpoint and function
func (m *SmartRouterMetricsManager) AddEndpointRelayErrored(spec, apiInterface, endpointID, function string) {
	if m == nil {
		return
	}
	labels := map[string]string{"spec": spec, "apiInterface": apiInterface, "endpoint_id": endpointID, "function": function}
	m.endpointTotalErrored.WithLabelValues(labels).Inc()
}

// SetEndpointEndToEndLatency observes a latency sample for an endpoint and function
func (m *SmartRouterMetricsManager) SetEndpointEndToEndLatency(spec, apiInterface, endpointID, function string, latencyMs float64) {
	if m == nil {
		return
	}
	m.endpointEndToEndLatency.WithLabelValues(spec, apiInterface, endpointID, function).Observe(latencyMs)
}

// AddEndpointInFlightRequest increments the in-flight counter for an endpoint and function
func (m *SmartRouterMetricsManager) AddEndpointInFlightRequest(spec, apiInterface, endpointID, function string) {
	if m == nil {
		return
	}
	labels := map[string]string{"spec": spec, "apiInterface": apiInterface, "endpoint_id": endpointID, "function": function}
	m.endpointInFlight.WithLabelValues(labels).Add(1)
}

// SubEndpointInFlightRequest decrements the in-flight counter for an endpoint and function
func (m *SmartRouterMetricsManager) SubEndpointInFlightRequest(spec, apiInterface, endpointID, function string) {
	if m == nil {
		return
	}
	labels := map[string]string{"spec": spec, "apiInterface": apiInterface, "endpoint_id": endpointID, "function": function}
	m.endpointInFlight.WithLabelValues(labels).Sub(1)
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

// SetEndpointInfo sets the static metadata mapping for an endpoint.
// endpoint_id is the provider name; endpoint_url carries the raw URL for reference.
func (m *SmartRouterMetricsManager) SetEndpointInfo(spec, apiInterface, endpointURL, providerName string) {
	if m == nil {
		return
	}
	labels := map[string]string{"spec": spec, "apiInterface": apiInterface, "endpoint_id": providerName, "endpoint_url": endpointURL}
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

// =============================================================================
// Health check methods
// =============================================================================

// UpdateOverallHealthStatus updates the overall health check status for the smart router
func (m *SmartRouterMetricsManager) UpdateOverallHealthStatus(healthy bool) {
	if m == nil {
		return
	}
	value := uint64(0)
	gaugeValue := 0.0
	if healthy {
		value = 1
		gaugeValue = 1.0
	}
	atomic.StoreUint64(&m.endpointsHealthChecksOk, value)
	m.routerOverallHealth.Set(gaugeValue)
}

// IsHealthy returns true if the smart router is healthy
func (m *SmartRouterMetricsManager) IsHealthy() bool {
	if m == nil {
		return true
	}
	return atomic.LoadUint64(&m.endpointsHealthChecksOk) == 1
}

// =============================================================================
// Endpoint registration and management
// =============================================================================

// RegisterEndpoint registers an endpoint and sets its info metric.
// endpointID is the raw URL (used internally for URL→name resolution via the chain tracker callbacks).
// providerName is used as endpoint_id in all Prometheus metrics; the URL is stored as endpoint_url in the info metric only.
func (m *SmartRouterMetricsManager) RegisterEndpoint(spec, apiInterface, endpointID, providerName string) {
	if m == nil {
		return
	}
	m.lock.Lock()
	defer m.lock.Unlock()

	key := spec + "|" + apiInterface + "|" + endpointID
	if _, exists := m.endpointMetrics[key]; !exists {
		m.endpointMetrics[key] = &EndpointMetrics{
			spec:         spec,
			apiInterface: apiInterface,
			endpointID:   endpointID,
		}
	}

	m.urlToProviderName[endpointID] = providerName
	m.SetEndpointInfo(spec, apiInterface, endpointID, providerName)
	// Initialize health to healthy so always-healthy endpoints appear in Prometheus from startup.
	// The relay code only calls SetEndpointOverallHealth on state transitions (unhealthy→healthy),
	// so without this initialization, endpoints that never fail would have no health metric at all.
	m.SetEndpointOverallHealth(spec, apiInterface, providerName, true)
}

// resolveProviderName returns the provider name for a given endpoint URL.
// Falls back to the input value if no mapping is found (e.g. when already a name).
func (m *SmartRouterMetricsManager) resolveProviderName(urlOrName string) string {
	m.lock.RLock()
	defer m.lock.RUnlock()
	if name, ok := m.urlToProviderName[urlOrName]; ok {
		return name
	}
	return urlOrName
}

// =============================================================================
// Convenience methods for recording relay results
// =============================================================================

// RecordRelaySuccess records a successful relay, updating router-level counters
// and endpoint-level counters and histograms with the function label.
// Note: router end-to-end latency is NOT recorded here — it is recorded separately
// via RecordRouterEndToEndLatency from SendParsedRelay, which captures the full
// request duration including provider selection overhead, not just the network hop.
func (m *SmartRouterMetricsManager) RecordRelaySuccess(spec, apiInterface, endpointID, function string, latencyMs float64) {
	if m == nil {
		return
	}
	m.routerTotalRelaysServiced.WithLabelValues(spec, apiInterface, function).Inc()
	m.AddEndpointRelayServiced(spec, apiInterface, endpointID, function)
	m.SetEndpointEndToEndLatency(spec, apiInterface, endpointID, function, latencyMs)
}

// RecordRouterEndToEndLatency records the full end-to-end latency for the router,
// measured from when SendParsedRelay started to when the result was returned.
// This captures the true client-visible latency including provider selection overhead.
func (m *SmartRouterMetricsManager) RecordRouterEndToEndLatency(spec, apiInterface, function string, latencyMs float64) {
	if m == nil {
		return
	}
	m.routerEndToEndLatency.WithLabelValues(spec, apiInterface, function).Observe(latencyMs)
}

// RecordRelayError records a failed relay, updating both router-level and
// endpoint-level error counters with the function label.
func (m *SmartRouterMetricsManager) RecordRelayError(spec, apiInterface, endpointID, function string) {
	if m == nil {
		return
	}
	m.routerTotalErrored.WithLabelValues(spec, apiInterface, function).Inc()
	m.AddEndpointRelayErrored(spec, apiInterface, endpointID, function)
}

// =============================================================================
// Reset methods
// =============================================================================

// ResetEndpointMetrics resets health metrics for a specific endpoint
func (m *SmartRouterMetricsManager) ResetEndpointMetrics(spec, apiInterface, endpointID string) {
	if m == nil {
		return
	}
	labels := map[string]string{"spec": spec, "apiInterface": apiInterface, "endpoint_id": endpointID}
	m.endpointOverallHealth.WithLabelValues(labels).Set(0)
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

	providerName := m.resolveProviderName(endpointID)
	if isLatest {
		if success {
			m.AddEndpointFetchLatestSuccess(spec, apiInterface, providerName)
		} else {
			m.AddEndpointFetchLatestFail(spec, apiInterface, providerName)
		}
	} else {
		if success {
			m.AddEndpointFetchBlockSuccess(spec, apiInterface, providerName)
		} else {
			m.AddEndpointFetchBlockFail(spec, apiInterface, providerName)
		}
	}
}

// =============================================================================
// ConsumerMetricsManagerInf implementation
// =============================================================================

func (m *SmartRouterMetricsManager) SetRelaySentToProviderMetric(string, string) {}

func (m *SmartRouterMetricsManager) SetRequestPerProvider(string, string) {}

func (m *SmartRouterMetricsManager) SetRelayProcessingLatencyBeforeProvider(time.Duration, string, string) {
}

func (m *SmartRouterMetricsManager) SetRelayProcessingLatencyAfterProvider(time.Duration, string, string) {
}

// SetEndToEndLatency is a no-op for SmartRouter.
// In direct RPC mode, RecordDirectRelayEnd handles latency with per-endpoint granularity.
func (m *SmartRouterMetricsManager) SetEndToEndLatency(chainId string, apiInterface string, latency time.Duration) {
}

func (m *SmartRouterMetricsManager) SetRelayNodeErrorMetric(chainId string, apiInterface string) {
	if m == nil {
		return
	}
	m.nodeErrorsReceived.WithLabelValues(chainId, apiInterface).Inc()
}

func (m *SmartRouterMetricsManager) SetNodeErrorRecoveredSuccessfullyMetric(chainId string, apiInterface string, attempt string) {
	if m == nil {
		return
	}
	m.nodeErrorsRecovered.WithLabelValues(chainId, apiInterface, attempt).Inc()
}

func (m *SmartRouterMetricsManager) SetProtocolErrorRecoveredSuccessfullyMetric(string, string, string) {
}

func (m *SmartRouterMetricsManager) SetProtocolError(string, string) {}

func (m *SmartRouterMetricsManager) SetCrossValidationMetric(chainId, apiInterface, method, status string, maxParticipants, agreementThreshold int, allEndpointsSorted, agreeingEndpointsSorted []string) {
	if m == nil {
		return
	}
	m.routerCrossValidation.WithLabelValues(
		chainId, apiInterface, method, status,
		strconv.Itoa(maxParticipants), strconv.Itoa(agreementThreshold),
		strings.Join(allEndpointsSorted, ","),
		strings.Join(agreeingEndpointsSorted, ","),
	).Inc()
}

func (m *SmartRouterMetricsManager) UpdateHealthCheckStatus(status bool) {
	m.UpdateOverallHealthStatus(status)
}

func (m *SmartRouterMetricsManager) UpdateHealthcheckStatusBreakdown(chainId, apiInterface string, status bool) {
}

func (m *SmartRouterMetricsManager) SetProviderLiveness(string, string, string, bool) {}

func (m *SmartRouterMetricsManager) SetProviderSelected(chainId string, _ string, allProviderScores []ProviderSelectionScores, _ float64) {
	if m == nil {
		return
	}
	for _, scores := range allProviderScores {
		endpointID := scores.ProviderAddress
		m.endpointSelectionScore.WithLabelValues(chainId, endpointID, "availability").Set(scores.Availability)
		m.endpointSelectionScore.WithLabelValues(chainId, endpointID, "latency").Set(scores.Latency)
		m.endpointSelectionScore.WithLabelValues(chainId, endpointID, "sync").Set(scores.Sync)
		m.endpointSelectionScore.WithLabelValues(chainId, endpointID, "stake").Set(scores.Stake)
		m.endpointSelectionScore.WithLabelValues(chainId, endpointID, "composite").Set(scores.Composite)
	}
}

func (m *SmartRouterMetricsManager) SetBlockedProvider(string, string, string, string, bool) {}

func (m *SmartRouterMetricsManager) SetQOSMetrics(chainId string, apiInterface string, _ string, _ string, _ *pairingtypes.QualityOfServiceReport, _ *pairingtypes.QualityOfServiceReport, _ int64, _ uint64, _ time.Duration, _ bool) {
}

func (m *SmartRouterMetricsManager) ResetSessionRelatedMetrics() {}

func (m *SmartRouterMetricsManager) ResetBlockedProvidersMetrics(string, string, map[string]string) {
}

// SetRelayMetrics is a no-op for SmartRouter.
// In direct RPC mode, relay tracking is done via RecordDirectRelayEnd which
// uses the correct per-endpoint labels. Allowing this path would cause double
// counting with a meaningless "default" endpoint_id.
func (m *SmartRouterMetricsManager) SetRelayMetrics(relayMetric *RelayMetrics, err error) {}

// SetRelaySentByNewBatchTickerMetric is a no-op for SmartRouter (no batch ticker in direct mode).
func (m *SmartRouterMetricsManager) SetRelaySentByNewBatchTickerMetric(chainId string, apiInterface string) {
}

// SetVersion records the protocol version of the smart router.
func (m *SmartRouterMetricsManager) SetVersion(version string) {
	if m == nil {
		return
	}
	SetVersionInner(m.routerProtocolVersion, version)
}

func (m *SmartRouterMetricsManager) SetWsSubscriptionRequestMetric(chainId, apiInterface string) {
	if m == nil {
		return
	}
	m.routerWsSubscriptionsTotal.WithLabelValues(chainId, apiInterface).Inc()
}

func (m *SmartRouterMetricsManager) SetFailedWsSubscriptionRequestMetric(chainId, apiInterface string) {
	if m == nil {
		return
	}
	m.routerWsSubscriptionErrors.WithLabelValues(chainId, apiInterface).Inc()
}

func (m *SmartRouterMetricsManager) SetDuplicatedWsSubscriptionRequestMetric(chainId, apiInterface string) {
	if m == nil {
		return
	}
	m.routerWsSubscriptionDuplicates.WithLabelValues(chainId, apiInterface).Inc()
}

func (m *SmartRouterMetricsManager) SetWsSubscriptioDisconnectRequestMetric(chainId, apiInterface, disconnectReason string) {
	if m == nil {
		return
	}
	m.routerWsSubscriptionDisconnects.WithLabelValues(chainId, apiInterface, disconnectReason).Inc()
}

func (m *SmartRouterMetricsManager) SetWebSocketConnectionActive(chainId, apiInterface string, add bool) {
	if m == nil {
		return
	}
	if add {
		m.routerWsConnectionsActive.WithLabelValues(chainId, apiInterface).Add(1)
	} else {
		m.routerWsConnectionsActive.WithLabelValues(chainId, apiInterface).Sub(1)
	}
}

func (m *SmartRouterMetricsManager) SetLoLResponse(bool) {}

func (m *SmartRouterMetricsManager) StartSelectionStatsUpdater(context.Context, time.Duration) {}
