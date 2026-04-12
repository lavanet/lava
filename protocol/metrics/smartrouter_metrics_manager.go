package metrics

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/lavanet/lava/v5/utils"
	pairingtypes "github.com/lavanet/lava/v5/x/pairing/types"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

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
	endpointOverallHealth          *MappedLabelsGaugeVec   // lava_rpc_endpoint_overall_health
	endpointOverallHealthBreakdown *prometheus.GaugeVec    // lava_rpc_endpoint_overall_health_breakdown {spec, apiInterface}
	endpointSelectionScore         *prometheus.GaugeVec    // lava_rpc_endpoint_selection_score {spec, apiInterface, endpoint_id, score_type}
	optimizerSelectionScore        *prometheus.GaugeVec    // lava_rpc_optimizer_selection_score {spec, endpoint_id, score_type}
	endpointLatestBlock            *MappedLabelsGaugeVec   // lava_rpc_endpoint_latest_block
	endpointFetchLatestFails       *MappedLabelsCounterVec // lava_rpc_endpoint_fetch_latest_fails
	endpointFetchBlockFails        *MappedLabelsCounterVec // lava_rpc_endpoint_fetch_block_fails
	endpointFetchLatestSuccess     *MappedLabelsCounterVec // lava_rpc_endpoint_fetch_latest_success
	endpointFetchBlockSuccess      *MappedLabelsCounterVec // lava_rpc_endpoint_fetch_block_success

	// Router-scoped metrics (labels: spec, apiInterface, function)
	routerTotalRelaysServiced *prometheus.CounterVec   // lava_rpcsmartrouter_total_relays_serviced
	routerTotalErrored        *prometheus.CounterVec   // lava_rpcsmartrouter_total_errored
	routerEndToEndLatency     *prometheus.HistogramVec // lava_rpcsmartrouter_end_to_end_latency_milliseconds

	// Router-scoped scalar metrics (no labels)
	routerOverallHealth          prometheus.Gauge
	routerOverallHealthBreakdown *prometheus.GaugeVec // lava_rpcsmartrouter_overall_health_breakdown {spec, apiInterface}

	// Router-scoped metrics (labels: spec, apiInterface)
	routerLatestBlock          *prometheus.GaugeVec
	routerProtocolVersion      *prometheus.GaugeVec
	routerWsConnectionsActive  *prometheus.GaugeVec
	routerWsSubscriptionsTotal *prometheus.CounterVec
	routerWsSubscriptionErrors *prometheus.CounterVec

	// Cross-validation group metrics
	crossValidationRequestsTotalMetric              *prometheus.CounterVec // lava_rpcsmartrouter_cross_validation_requests_total        {spec, apiInterface, method}
	crossValidationSuccessTotalMetric               *prometheus.CounterVec // lava_rpcsmartrouter_cross_validation_success_total         {spec, apiInterface, method}
	crossValidationFailedTotalMetric                *prometheus.CounterVec // lava_rpcsmartrouter_cross_validation_failed_total          {spec, apiInterface, method}
	crossValidationProviderAgreementsTotalMetric    *prometheus.CounterVec // lava_rpcsmartrouter_cross_validation_provider_agreements_total    {spec, apiInterface, method, provider_address}
	crossValidationProviderDisagreementsTotalMetric *prometheus.CounterVec // lava_rpcsmartrouter_cross_validation_provider_disagreements_total {spec, apiInterface, method, provider_address}

	// Incident group metrics
	incidentNodeErrorsTotalMetric     *prometheus.CounterVec   // lava_rpcsmartrouter_node_errors_total         {spec, apiInterface, provider_address, method}
	incidentProtocolErrorsTotalMetric *prometheus.CounterVec   // lava_rpcsmartrouter_protocol_errors_total     {spec, apiInterface, provider_address, method}
	incidentRetriesTotalMetric        *prometheus.CounterVec   // lava_rpcsmartrouter_retries_total             {spec, apiInterface, method}
	incidentRetriesSuccessMetric      *prometheus.CounterVec   // lava_rpcsmartrouter_retries_success_total     {spec, apiInterface, method}
	incidentRetriesFailedMetric       *prometheus.CounterVec   // lava_rpcsmartrouter_retries_failed_total      {spec, apiInterface, method}
	incidentRetryAttemptsHistogram    *prometheus.HistogramVec // lava_rpcsmartrouter_retry_attempts            {spec, apiInterface, method}
	incidentConsistencyTotalMetric    *prometheus.CounterVec   // lava_rpcsmartrouter_consistency_total         {spec, apiInterface, method}
	incidentConsistencySuccessMetric  *prometheus.CounterVec   // lava_rpcsmartrouter_consistency_success_total {spec, apiInterface, method}
	incidentConsistencyFailedMetric   *prometheus.CounterVec   // lava_rpcsmartrouter_consistency_failed_total  {spec, apiInterface, method}
	incidentHedgeTotalMetric          *prometheus.CounterVec   // lava_rpcsmartrouter_hedge_total               {spec, apiInterface, method}
	incidentHedgeSuccessMetric        *prometheus.CounterVec   // lava_rpcsmartrouter_hedge_success_total       {spec, apiInterface, method}
	incidentHedgeFailedMetric         *prometheus.CounterVec   // lava_rpcsmartrouter_hedge_failed_total        {spec, apiInterface, method}
	incidentHedgeAttemptsHistogram    *prometheus.HistogramVec // lava_rpcsmartrouter_hedge_attempts            {spec, apiInterface, method}

	// Cache group metrics (labels: spec, apiInterface, method)
	cacheRequestsTotalMetric *prometheus.CounterVec   // lava_rpcsmartrouter_cache_requests_total
	cacheSuccessTotalMetric  *prometheus.CounterVec   // lava_rpcsmartrouter_cache_success_total
	cacheFailedTotalMetric   *prometheus.CounterVec   // lava_rpcsmartrouter_cache_failed_total
	cacheLatencyHistogram    *prometheus.HistogramVec // lava_rpcsmartrouter_cache_latency_milliseconds

	// Router-scoped request group metrics (labels: spec, apiInterface, provider_address, method)
	routerRequestsTotal      *prometheus.CounterVec
	routerRequestsSuccess    *prometheus.CounterVec
	routerRequestsFailed     *prometheus.CounterVec
	routerRequestsRead       *prometheus.CounterVec
	routerRequestsWrite      *prometheus.CounterVec
	routerRequestsDebugTrace *prometheus.CounterVec
	routerRequestsArchive    *prometheus.CounterVec
	routerRequestsBatch      *prometheus.CounterVec

	// Internal state
	lock                    sync.RWMutex
	endpointsHealthChecksOk uint64
	endpointMetrics         map[string]*EndpointMetrics
	urlToProviderName       map[string]string // maps endpoint URL → provider name for metric label resolution
	optimizerQoSClient      *ConsumerOptimizerQoSClient
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
	NetworkAddress     string
	StartHTTPServer    bool // If false, only register metrics (for use alongside ConsumerMetricsManager)
	OptimizerQoSClient *ConsumerOptimizerQoSClient
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
		Name:       "lava_rpc_endpoint_total_relays_serviced",
		Help:       "Total relays successfully serviced by this RPC endpoint, by function.",
		Labels:     endpointFunctionLabels,
		Registerer: prometheus.DefaultRegisterer,
	})

	endpointTotalErrored := NewMappedLabelsCounterVec(MappedLabelsMetricOpts{
		Name:       "lava_rpc_endpoint_total_errored",
		Help:       "Total errored relays for this RPC endpoint, by function.",
		Labels:     endpointFunctionLabels,
		Registerer: prometheus.DefaultRegisterer,
	})

	endpointInFlight := NewMappedLabelsGaugeVec(MappedLabelsMetricOpts{
		Name:       "lava_rpc_endpoint_requests_in_flight",
		Help:       "Current number of in-flight relays for this RPC endpoint, by function.",
		Labels:     endpointFunctionLabels,
		Registerer: prometheus.DefaultRegisterer,
	})

	endpointOverallHealth := NewMappedLabelsGaugeVec(MappedLabelsMetricOpts{
		Name:       "lava_rpc_endpoint_overall_health",
		Help:       "Health status of this RPC endpoint (1=healthy, 0=unhealthy).",
		Labels:     endpointLabels,
		Registerer: prometheus.DefaultRegisterer,
	})

	endpointOverallHealthBreakdown := registerOrReuse(prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "lava_rpc_endpoint_overall_health_breakdown",
		Help: "Health check status per chain for RPC endpoints (1=healthy, 0=unhealthy).",
	}, []string{"spec", "apiInterface"}))

	endpointSelectionScore := registerOrReuse(prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "lava_rpc_endpoint_selection_score",
		Help: "Latest selection score for each RPC endpoint by score type (availability/latency/sync/stake/composite).",
	}, []string{"spec", "apiInterface", "endpoint_id", "score_type"}))

	optimizerSelectionScore := registerOrReuse(prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "lava_rpc_optimizer_selection_score",
		Help: "Periodic optimizer selection score per provider by score type (availability/latency/sync/stake/composite). Chain-level, not scoped to apiInterface.",
	}, []string{"spec", "endpoint_id", "score_type"}))

	endpointLatestBlock := NewMappedLabelsGaugeVec(MappedLabelsMetricOpts{
		Name:       "lava_rpc_endpoint_latest_block",
		Help:       "Latest block known by this RPC endpoint.",
		Labels:     endpointLabels,
		Registerer: prometheus.DefaultRegisterer,
	})

	endpointFetchLatestFails := NewMappedLabelsCounterVec(MappedLabelsMetricOpts{
		Name:       "lava_rpc_endpoint_fetch_latest_fails",
		Help:       "Total failed latest-block fetch operations for this RPC endpoint.",
		Labels:     endpointLabels,
		Registerer: prometheus.DefaultRegisterer,
	})

	endpointFetchBlockFails := NewMappedLabelsCounterVec(MappedLabelsMetricOpts{
		Name:       "lava_rpc_endpoint_fetch_block_fails",
		Help:       "Total failed specific-block fetch operations for this RPC endpoint.",
		Labels:     endpointLabels,
		Registerer: prometheus.DefaultRegisterer,
	})

	endpointFetchLatestSuccess := NewMappedLabelsCounterVec(MappedLabelsMetricOpts{
		Name:       "lava_rpc_endpoint_fetch_latest_success",
		Help:       "Total successful latest-block fetch operations for this RPC endpoint.",
		Labels:     endpointLabels,
		Registerer: prometheus.DefaultRegisterer,
	})

	endpointFetchBlockSuccess := NewMappedLabelsCounterVec(MappedLabelsMetricOpts{
		Name:       "lava_rpc_endpoint_fetch_block_success",
		Help:       "Total successful specific-block fetch operations for this RPC endpoint.",
		Labels:     endpointLabels,
		Registerer: prometheus.DefaultRegisterer,
	})

	// =========================================================================
	// Endpoint-scoped histogram (labels: spec, apiInterface, endpoint_id, function)
	// =========================================================================

	endpointEndToEndLatency := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "lava_rpc_endpoint_end_to_end_latency_milliseconds",
		Help:    "Distribution of end-to-end relay latency for this RPC endpoint by function in milliseconds. Use histogram_quantile() for percentiles.",
		Buckets: latencyBuckets,
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
		Buckets: latencyBuckets,
	}, routerFunctionLabels)

	routerOverallHealth := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "lava_rpcsmartrouter_overall_health",
		Help: "Overall health of the RPC smart router (1=healthy, 0=unhealthy).",
	})
	routerOverallHealth = registerOrReuse(routerOverallHealth)
	routerOverallHealth.Set(1)

	routerOverallHealthBreakdown := registerOrReuse(prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "lava_rpcsmartrouter_overall_health_breakdown",
		Help: "Health check status per chain on the smart router (1=healthy, 0=unhealthy).",
	}, []string{"spec", "apiInterface"}))

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

	crossValidationLabels := []string{"spec", "apiInterface", "method"}
	crossValidationProviderLabels := []string{"spec", "apiInterface", "method", "provider_address"}
	crossValidationRequestsTotalMetric := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "lava_rpcsmartrouter_cross_validation_requests_total",
		Help: "Total number of cross-validated requests.",
	}, crossValidationLabels)
	crossValidationSuccessTotalMetric := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "lava_rpcsmartrouter_cross_validation_success_total",
		Help: "Total number of cross-validated requests that reached consensus.",
	}, crossValidationLabels)
	crossValidationFailedTotalMetric := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "lava_rpcsmartrouter_cross_validation_failed_total",
		Help: "Total number of cross-validated requests that failed to reach consensus.",
	}, crossValidationLabels)
	crossValidationProviderAgreementsTotalMetric := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "lava_rpcsmartrouter_cross_validation_provider_agreements_total",
		Help: "Total number of times a provider agreed with the cross-validation consensus.",
	}, crossValidationProviderLabels)
	crossValidationProviderDisagreementsTotalMetric := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "lava_rpcsmartrouter_cross_validation_provider_disagreements_total",
		Help: "Total number of times a provider's response disagreed with the cross-validation consensus.",
	}, crossValidationProviderLabels)

	// =========================================================================
	// Incident group metrics
	// =========================================================================
	incidentProviderLabels := []string{"spec", "apiInterface", "provider_address", "method"}
	incidentMethodLabels := []string{"spec", "apiInterface", "method"}

	incidentNodeErrorsTotalMetric := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "lava_rpcsmartrouter_node_errors_total",
		Help: "Total node errors received from RPC endpoints by the smart router.",
	}, incidentProviderLabels)
	incidentProtocolErrorsTotalMetric := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "lava_rpcsmartrouter_protocol_errors_total",
		Help: "Total protocol errors (transport/timeout) encountered by the smart router.",
	}, incidentProviderLabels)
	incidentRetriesTotalMetric := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "lava_rpcsmartrouter_retries_total",
		Help: "Total relay retries attempted by the smart router (extra attempts beyond the first).",
	}, incidentMethodLabels)
	incidentRetriesSuccessMetric := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "lava_rpcsmartrouter_retries_success_total",
		Help: "Retried relay requests on the smart router that ultimately succeeded.",
	}, incidentMethodLabels)
	incidentRetriesFailedMetric := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "lava_rpcsmartrouter_retries_failed_total",
		Help: "Retried relay requests on the smart router that ultimately failed.",
	}, incidentMethodLabels)
	incidentRetryAttemptsHistogram := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "lava_rpcsmartrouter_retry_attempts",
		Help:    "Distribution of how many provider attempts were needed per retried request.",
		Buckets: []float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
	}, incidentMethodLabels)
	incidentConsistencyTotalMetric := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "lava_rpcsmartrouter_consistency_total",
		Help: "Total relay requests on the smart router that enforced a minimum seen block (consistency).",
	}, incidentMethodLabels)
	incidentConsistencySuccessMetric := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "lava_rpcsmartrouter_consistency_success_total",
		Help: "Consistency-enforced relay requests on the smart router that succeeded.",
	}, incidentMethodLabels)
	incidentConsistencyFailedMetric := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "lava_rpcsmartrouter_consistency_failed_total",
		Help: "Consistency-enforced relay requests on the smart router that failed.",
	}, incidentMethodLabels)
	incidentHedgeTotalMetric := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "lava_rpcsmartrouter_hedge_total",
		Help: "Total relay requests on the smart router that triggered a hedge (batch ticker).",
	}, incidentMethodLabels)
	incidentHedgeSuccessMetric := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "lava_rpcsmartrouter_hedge_success_total",
		Help: "Hedged relay requests on the smart router that succeeded.",
	}, incidentMethodLabels)
	incidentHedgeFailedMetric := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "lava_rpcsmartrouter_hedge_failed_total",
		Help: "Hedged relay requests on the smart router that failed.",
	}, incidentMethodLabels)
	incidentHedgeAttemptsHistogram := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "lava_rpcsmartrouter_hedge_attempts",
		Help:    "Distribution of how many hedge relays were sent per hedged request.",
		Buckets: []float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
	}, incidentMethodLabels)

	// =========================================================================
	// Request group metrics
	// =========================================================================
	routerRequestLabels := []string{"spec", "apiInterface", "provider_address", "method"}

	routerRequestsTotal := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "lava_rpcsmartrouter_requests_total",
		Help: "Total number of requests on the smart router.",
	}, routerRequestLabels)
	routerRequestsSuccess := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "lava_rpcsmartrouter_requests_success_total",
		Help: "Total number of successful requests on the smart router.",
	}, routerRequestLabels)
	routerRequestsFailed := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "lava_rpcsmartrouter_requests_failed_total",
		Help: "Total number of failed requests on the smart router.",
	}, routerRequestLabels)
	routerRequestsRead := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "lava_rpcsmartrouter_requests_read_total",
		Help: "Total number of read (stateful=0) requests on the smart router.",
	}, routerRequestLabels)
	routerRequestsWrite := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "lava_rpcsmartrouter_requests_write_total",
		Help: "Total number of write (stateful=1) requests on the smart router.",
	}, routerRequestLabels)
	routerRequestsDebugTrace := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "lava_rpcsmartrouter_requests_debug_trace_total",
		Help: "Total number of debug/trace addon requests on the smart router.",
	}, routerRequestLabels)
	routerRequestsArchive := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "lava_rpcsmartrouter_requests_archive_total",
		Help: "Total number of archive requests on the smart router.",
	}, routerRequestLabels)
	routerRequestsBatch := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "lava_rpcsmartrouter_requests_batch_total",
		Help: "Total number of batch requests on the smart router.",
	}, routerRequestLabels)

	cacheLabels := []string{"spec", "apiInterface", "method"}
	cacheRequestsTotalMetric := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "lava_rpcsmartrouter_cache_requests_total",
		Help: "Total number of cache lookup attempts.",
	}, cacheLabels)
	cacheSuccessTotalMetric := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "lava_rpcsmartrouter_cache_success_total",
		Help: "Total number of cache lookups that returned a cached response (hits).",
	}, cacheLabels)
	cacheFailedTotalMetric := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "lava_rpcsmartrouter_cache_failed_total",
		Help: "Total number of cache lookups that did not find a cached response (misses).",
	}, cacheLabels)
	cacheLatencyHistogram := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "lava_rpcsmartrouter_cache_latency_milliseconds",
		Help:    "Distribution of cache lookup latency in milliseconds.",
		Buckets: latencyBuckets,
	}, cacheLabels)

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
	endpointEndToEndLatency = registerOrReuse(endpointEndToEndLatency)
	crossValidationRequestsTotalMetric = registerOrReuse(crossValidationRequestsTotalMetric)
	crossValidationSuccessTotalMetric = registerOrReuse(crossValidationSuccessTotalMetric)
	crossValidationFailedTotalMetric = registerOrReuse(crossValidationFailedTotalMetric)
	crossValidationProviderAgreementsTotalMetric = registerOrReuse(crossValidationProviderAgreementsTotalMetric)
	crossValidationProviderDisagreementsTotalMetric = registerOrReuse(crossValidationProviderDisagreementsTotalMetric)
	incidentNodeErrorsTotalMetric = registerOrReuse(incidentNodeErrorsTotalMetric)
	incidentProtocolErrorsTotalMetric = registerOrReuse(incidentProtocolErrorsTotalMetric)
	incidentRetriesTotalMetric = registerOrReuse(incidentRetriesTotalMetric)
	incidentRetriesSuccessMetric = registerOrReuse(incidentRetriesSuccessMetric)
	incidentRetriesFailedMetric = registerOrReuse(incidentRetriesFailedMetric)
	incidentRetryAttemptsHistogram = registerOrReuse(incidentRetryAttemptsHistogram)
	incidentConsistencyTotalMetric = registerOrReuse(incidentConsistencyTotalMetric)
	incidentConsistencySuccessMetric = registerOrReuse(incidentConsistencySuccessMetric)
	incidentConsistencyFailedMetric = registerOrReuse(incidentConsistencyFailedMetric)
	incidentHedgeTotalMetric = registerOrReuse(incidentHedgeTotalMetric)
	incidentHedgeSuccessMetric = registerOrReuse(incidentHedgeSuccessMetric)
	incidentHedgeFailedMetric = registerOrReuse(incidentHedgeFailedMetric)
	incidentHedgeAttemptsHistogram = registerOrReuse(incidentHedgeAttemptsHistogram)
	routerRequestsTotal = registerOrReuse(routerRequestsTotal)
	routerRequestsSuccess = registerOrReuse(routerRequestsSuccess)
	routerRequestsFailed = registerOrReuse(routerRequestsFailed)
	routerRequestsRead = registerOrReuse(routerRequestsRead)
	routerRequestsWrite = registerOrReuse(routerRequestsWrite)
	routerRequestsDebugTrace = registerOrReuse(routerRequestsDebugTrace)
	routerRequestsArchive = registerOrReuse(routerRequestsArchive)
	routerRequestsBatch = registerOrReuse(routerRequestsBatch)
	cacheRequestsTotalMetric = registerOrReuse(cacheRequestsTotalMetric)
	cacheSuccessTotalMetric = registerOrReuse(cacheSuccessTotalMetric)
	cacheFailedTotalMetric = registerOrReuse(cacheFailedTotalMetric)
	cacheLatencyHistogram = registerOrReuse(cacheLatencyHistogram)

	manager := &SmartRouterMetricsManager{
		// Endpoint-scoped (with function)
		endpointTotalRelaysServiced: endpointTotalRelaysServiced,
		endpointTotalErrored:        endpointTotalErrored,
		endpointEndToEndLatency:     endpointEndToEndLatency,
		endpointInFlight:            endpointInFlight,

		// Endpoint-scoped (without function)
		endpointOverallHealth:          endpointOverallHealth,
		endpointOverallHealthBreakdown: endpointOverallHealthBreakdown,
		endpointSelectionScore:         endpointSelectionScore,
		optimizerSelectionScore:        optimizerSelectionScore,
		endpointLatestBlock:            endpointLatestBlock,
		endpointFetchLatestFails:       endpointFetchLatestFails,
		endpointFetchBlockFails:        endpointFetchBlockFails,
		endpointFetchLatestSuccess:     endpointFetchLatestSuccess,
		endpointFetchBlockSuccess:      endpointFetchBlockSuccess,

		// Router-scoped (with function)
		routerTotalRelaysServiced: routerTotalRelaysServiced,
		routerTotalErrored:        routerTotalErrored,
		routerEndToEndLatency:     routerEndToEndLatency,

		// Router-scoped scalar
		routerOverallHealth:          routerOverallHealth,
		routerOverallHealthBreakdown: routerOverallHealthBreakdown,

		// Router-scoped (without function)
		routerLatestBlock:          routerLatestBlock,
		routerProtocolVersion:      routerProtocolVersion,
		routerWsConnectionsActive:  routerWsConnectionsActive,
		routerWsSubscriptionsTotal: routerWsSubscriptionsTotal,
		routerWsSubscriptionErrors: routerWsSubscriptionErrors,

		// Cross-validation group
		crossValidationRequestsTotalMetric:              crossValidationRequestsTotalMetric,
		crossValidationSuccessTotalMetric:               crossValidationSuccessTotalMetric,
		crossValidationFailedTotalMetric:                crossValidationFailedTotalMetric,
		crossValidationProviderAgreementsTotalMetric:    crossValidationProviderAgreementsTotalMetric,
		crossValidationProviderDisagreementsTotalMetric: crossValidationProviderDisagreementsTotalMetric,

		// Incident group
		incidentNodeErrorsTotalMetric:     incidentNodeErrorsTotalMetric,
		incidentProtocolErrorsTotalMetric: incidentProtocolErrorsTotalMetric,
		incidentRetriesTotalMetric:        incidentRetriesTotalMetric,
		incidentRetriesSuccessMetric:      incidentRetriesSuccessMetric,
		incidentRetriesFailedMetric:       incidentRetriesFailedMetric,
		incidentRetryAttemptsHistogram:    incidentRetryAttemptsHistogram,
		incidentConsistencyTotalMetric:    incidentConsistencyTotalMetric,
		incidentConsistencySuccessMetric:  incidentConsistencySuccessMetric,
		incidentConsistencyFailedMetric:   incidentConsistencyFailedMetric,
		incidentHedgeTotalMetric:          incidentHedgeTotalMetric,
		incidentHedgeSuccessMetric:        incidentHedgeSuccessMetric,
		incidentHedgeFailedMetric:         incidentHedgeFailedMetric,
		incidentHedgeAttemptsHistogram:    incidentHedgeAttemptsHistogram,

		// Router-scoped request group
		routerRequestsTotal:      routerRequestsTotal,
		routerRequestsSuccess:    routerRequestsSuccess,
		routerRequestsFailed:     routerRequestsFailed,
		routerRequestsRead:       routerRequestsRead,
		routerRequestsWrite:      routerRequestsWrite,
		routerRequestsDebugTrace: routerRequestsDebugTrace,
		routerRequestsArchive:    routerRequestsArchive,
		routerRequestsBatch:      routerRequestsBatch,

		// Cache group
		cacheRequestsTotalMetric: cacheRequestsTotalMetric,
		cacheSuccessTotalMetric:  cacheSuccessTotalMetric,
		cacheFailedTotalMetric:   cacheFailedTotalMetric,
		cacheLatencyHistogram:    cacheLatencyHistogram,

		// Internal state
		endpointsHealthChecksOk: 1,
		endpointMetrics:         make(map[string]*EndpointMetrics),
		urlToProviderName:       make(map[string]string),
		optimizerQoSClient:      options.OptimizerQoSClient,
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

// SetEndpointOverallHealthBreakdown sets the aggregate health for a spec/apiInterface across endpoints
func (m *SmartRouterMetricsManager) SetEndpointOverallHealthBreakdown(spec, apiInterface string, healthy bool) {
	if m == nil {
		return
	}
	value := 0.0
	if healthy {
		value = 1.0
	}
	m.endpointOverallHealthBreakdown.WithLabelValues(spec, apiInterface).Set(value)
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

// RegisterEndpoint registers an endpoint with the metrics manager.
// endpointID is the raw URL (used internally for URL→name resolution via the chain tracker callbacks).
// providerName is used as endpoint_id in all Prometheus metrics.
func (m *SmartRouterMetricsManager) RegisterEndpoint(spec, apiInterface, endpointID, providerName string) {
	if m == nil {
		return
	}

	m.lock.Lock()
	key := spec + "|" + apiInterface + "|" + endpointID
	if _, exists := m.endpointMetrics[key]; !exists {
		m.endpointMetrics[key] = &EndpointMetrics{
			spec:         spec,
			apiInterface: apiInterface,
			endpointID:   endpointID,
		}
	}
	m.urlToProviderName[endpointID] = providerName
	m.lock.Unlock()

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
	if m.routerTotalRelaysServiced != nil {
		m.routerTotalRelaysServiced.WithLabelValues(spec, apiInterface, function).Inc()
	}
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
	if m.routerTotalErrored != nil {
		m.routerTotalErrored.WithLabelValues(spec, apiInterface, function).Inc()
	}
	m.AddEndpointRelayErrored(spec, apiInterface, endpointID, function)
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

// RecordDirectRelayEnd records the end of an in-flight request and updates metrics.
// relay carries the request classification (IsWrite/IsArchive/IsDebugTrace/IsBatch)
// that the call site should populate on the shared RelayMetrics before calling here.
func (m *SmartRouterMetricsManager) RecordDirectRelayEnd(spec, apiInterface, endpointID, function string, latencyMs float64, success bool, relay *RelayMetrics) {
	if m == nil {
		return
	}
	m.SubEndpointInFlightRequest(spec, apiInterface, endpointID, function)

	if success {
		m.RecordRelaySuccess(spec, apiInterface, endpointID, function, latencyMs)
	} else {
		m.RecordRelayError(spec, apiInterface, endpointID, function)
	}

	providerName := m.resolveProviderName(endpointID)
	routerLabels := []string{spec, apiInterface, providerName, function}

	m.routerRequestsTotal.WithLabelValues(routerLabels...).Inc()
	if success {
		m.routerRequestsSuccess.WithLabelValues(routerLabels...).Inc()
	} else {
		m.routerRequestsFailed.WithLabelValues(routerLabels...).Inc()
	}
	if relay != nil && relay.IsBatch {
		m.routerRequestsBatch.WithLabelValues(routerLabels...).Inc()
	} else {
		if relay != nil && relay.IsWrite {
			m.routerRequestsWrite.WithLabelValues(routerLabels...).Inc()
		} else {
			m.routerRequestsRead.WithLabelValues(routerLabels...).Inc()
		}
		if relay != nil && relay.IsDebugTrace {
			m.routerRequestsDebugTrace.WithLabelValues(routerLabels...).Inc()
		}
		if relay != nil && relay.IsArchive {
			m.routerRequestsArchive.WithLabelValues(routerLabels...).Inc()
		}
	}
}

// RecordCacheHitRequest records a request served from the router's in-process cache
// in the router-level request-group counters. Cache hits are always successful so only
// the success counter is incremented (alongside total). The provider_address label is
// set to "Cached" so dashboards can distinguish cache traffic from endpoint traffic.
func (m *SmartRouterMetricsManager) RecordCacheHitRequest(spec, apiInterface, function string, relay *RelayMetrics) {
	if m == nil {
		return
	}
	routerLabels := []string{spec, apiInterface, "Cached", function}
	m.routerRequestsTotal.WithLabelValues(routerLabels...).Inc()
	m.routerRequestsSuccess.WithLabelValues(routerLabels...).Inc()
	if relay != nil && relay.IsBatch {
		m.routerRequestsBatch.WithLabelValues(routerLabels...).Inc()
	} else {
		if relay != nil && relay.IsWrite {
			m.routerRequestsWrite.WithLabelValues(routerLabels...).Inc()
		} else {
			m.routerRequestsRead.WithLabelValues(routerLabels...).Inc()
		}
		if relay != nil && relay.IsDebugTrace {
			m.routerRequestsDebugTrace.WithLabelValues(routerLabels...).Inc()
		}
		if relay != nil && relay.IsArchive {
			m.routerRequestsArchive.WithLabelValues(routerLabels...).Inc()
		}
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

// RecordEndToEndLatency is a no-op for SmartRouter — end-to-end latency is
// recorded via RecordRouterEndToEndLatency in SendParsedRelay.
func (m *SmartRouterMetricsManager) RecordEndToEndLatency(string, string, string, float64) {}

// RecordProviderLatency is a no-op for SmartRouter — provider latency is
// already captured per-endpoint via RecordDirectRelayEnd.
func (m *SmartRouterMetricsManager) RecordProviderLatency(string, string, string, string, float64) {}

func (m *SmartRouterMetricsManager) SetRelayNodeErrorMetric(chainId string, apiInterface string, providerAddress string, method string) {
	if m == nil {
		return
	}
	m.incidentNodeErrorsTotalMetric.WithLabelValues(chainId, apiInterface, providerAddress, method).Inc()
}

func (m *SmartRouterMetricsManager) RecordCacheResult(chainId, apiInterface, method string, hit bool, latencyMs float64) {
	if m == nil {
		return
	}
	m.cacheRequestsTotalMetric.WithLabelValues(chainId, apiInterface, method).Inc()
	if hit {
		m.cacheSuccessTotalMetric.WithLabelValues(chainId, apiInterface, method).Inc()
	} else {
		m.cacheFailedTotalMetric.WithLabelValues(chainId, apiInterface, method).Inc()
	}
	if hit {
		m.cacheLatencyHistogram.WithLabelValues(chainId, apiInterface, method).Observe(latencyMs)
	}
}

func (m *SmartRouterMetricsManager) SetProtocolError(chainId string, apiInterface string, providerAddress string, method string) {
	if m == nil {
		return
	}
	m.incidentProtocolErrorsTotalMetric.WithLabelValues(chainId, apiInterface, providerAddress, method).Inc()
}

func (m *SmartRouterMetricsManager) RecordIncidentRetry(chainId string, apiInterface string, method string, count uint64, success bool) {
	if m == nil {
		return
	}
	m.incidentRetriesTotalMetric.WithLabelValues(chainId, apiInterface, method).Inc()
	m.incidentRetryAttemptsHistogram.WithLabelValues(chainId, apiInterface, method).Observe(float64(count))
	if success {
		m.incidentRetriesSuccessMetric.WithLabelValues(chainId, apiInterface, method).Inc()
	} else {
		m.incidentRetriesFailedMetric.WithLabelValues(chainId, apiInterface, method).Inc()
	}
}

func (m *SmartRouterMetricsManager) RecordIncidentConsistency(chainId string, apiInterface string, method string, success bool) {
	if m == nil {
		return
	}
	m.incidentConsistencyTotalMetric.WithLabelValues(chainId, apiInterface, method).Inc()
	if success {
		m.incidentConsistencySuccessMetric.WithLabelValues(chainId, apiInterface, method).Inc()
	} else {
		m.incidentConsistencyFailedMetric.WithLabelValues(chainId, apiInterface, method).Inc()
	}
}

func (m *SmartRouterMetricsManager) RecordIncidentHedgeResult(chainId string, apiInterface string, method string, count uint64, success bool) {
	if m == nil {
		return
	}
	m.incidentHedgeTotalMetric.WithLabelValues(chainId, apiInterface, method).Inc()
	m.incidentHedgeAttemptsHistogram.WithLabelValues(chainId, apiInterface, method).Observe(float64(count))
	if success {
		m.incidentHedgeSuccessMetric.WithLabelValues(chainId, apiInterface, method).Inc()
	} else {
		m.incidentHedgeFailedMetric.WithLabelValues(chainId, apiInterface, method).Inc()
	}
}

func (m *SmartRouterMetricsManager) SetCrossValidationMetric(chainId, apiInterface, method string, success bool, agreeingProviders, disagreeingProviders []string) {
	if m == nil {
		return
	}
	m.crossValidationRequestsTotalMetric.WithLabelValues(chainId, apiInterface, method).Inc()
	if success {
		m.crossValidationSuccessTotalMetric.WithLabelValues(chainId, apiInterface, method).Inc()
	} else {
		m.crossValidationFailedTotalMetric.WithLabelValues(chainId, apiInterface, method).Inc()
	}
	for _, provider := range agreeingProviders {
		m.crossValidationProviderAgreementsTotalMetric.WithLabelValues(chainId, apiInterface, method, provider).Inc()
	}
	for _, provider := range disagreeingProviders {
		m.crossValidationProviderDisagreementsTotalMetric.WithLabelValues(chainId, apiInterface, method, provider).Inc()
	}
}

func (m *SmartRouterMetricsManager) UpdateHealthCheckStatus(status bool) {
	m.UpdateOverallHealthStatus(status)
}

func (m *SmartRouterMetricsManager) UpdateHealthcheckStatusBreakdown(chainId, apiInterface string, status bool) {
	if m == nil {
		return
	}
	value := 0.0
	if status {
		value = 1.0
	}
	m.routerOverallHealthBreakdown.WithLabelValues(chainId, apiInterface).Set(value)
}

func (m *SmartRouterMetricsManager) SetProviderLiveness(string, string, string, bool) {}

func (m *SmartRouterMetricsManager) SetProviderSelected(chainId string, apiInterface string, _ string, allProviderScores []ProviderSelectionScores, _ float64) {
	if m == nil {
		return
	}
	for _, scores := range allProviderScores {
		endpointID := scores.ProviderAddress
		m.endpointSelectionScore.WithLabelValues(chainId, apiInterface, endpointID, "availability").Set(scores.Availability)
		m.endpointSelectionScore.WithLabelValues(chainId, apiInterface, endpointID, "latency").Set(scores.Latency)
		m.endpointSelectionScore.WithLabelValues(chainId, apiInterface, endpointID, "sync").Set(scores.Sync)
		m.endpointSelectionScore.WithLabelValues(chainId, apiInterface, endpointID, "stake").Set(scores.Stake)
		m.endpointSelectionScore.WithLabelValues(chainId, apiInterface, endpointID, "composite").Set(scores.Composite)
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

func (m *SmartRouterMetricsManager) StartSelectionStatsUpdater(ctx context.Context, updateInterval time.Duration) {
	if m == nil || m.optimizerQoSClient == nil {
		return
	}
	if updateInterval <= 0 {
		utils.LavaFormatWarning("StartSelectionStatsUpdater: invalid updateInterval, selection stats will not be reported", nil,
			utils.LogAttr("updateInterval", updateInterval))
		return
	}
	go func() {
		ticker := time.NewTicker(updateInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				for _, report := range m.optimizerQoSClient.GetReportsToSend() {
					endpointID := report.ProviderAddress
					m.optimizerSelectionScore.WithLabelValues(report.ChainId, endpointID, "availability").Set(report.SelectionAvailability)
					m.optimizerSelectionScore.WithLabelValues(report.ChainId, endpointID, "latency").Set(report.SelectionLatency)
					m.optimizerSelectionScore.WithLabelValues(report.ChainId, endpointID, "sync").Set(report.SelectionSync)
					m.optimizerSelectionScore.WithLabelValues(report.ChainId, endpointID, "stake").Set(report.SelectionStake)
					m.optimizerSelectionScore.WithLabelValues(report.ChainId, endpointID, "composite").Set(report.SelectionComposite)
				}
			}
		}
	}()
}
