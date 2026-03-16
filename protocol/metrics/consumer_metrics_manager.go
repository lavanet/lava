package metrics

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/lavanet/lava/v5/utils"
	pairingtypes "github.com/lavanet/lava/v5/x/pairing/types"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	WsDisconnectionReasonConsumer = "consumer-disconnect"
	WsDisconnectionReasonProvider = "provider-disconnect"
	WsDisconnectionReasonUser     = "user-disconnect"
)

var ShowProviderEndpointInMetrics = false


type ConsumerMetricsManager struct {
	totalCURequestedMetric *prometheus.CounterVec
	totalWsSubscriptionRequestsMetric              *prometheus.CounterVec
	totalFailedWsSubscriptionRequestsMetric        *prometheus.CounterVec
	totalWsSubscriptionDisconnectMetric            *prometheus.CounterVec
	totalDuplicatedWsSubscriptionRequestsMetric    *prometheus.CounterVec
	totalLoLSuccessMetric                          prometheus.Counter
	totalLoLErrorsMetric                           prometheus.Counter
	totalWebSocketConnectionsActive                *prometheus.GaugeVec
	blockMetric                                    *prometheus.GaugeVec
	qosMetric                                      *MappedLabelsGaugeVec
	providerReputationMetric                       *MappedLabelsGaugeVec
	selectionStatsMetric                           *MappedLabelsGaugeVec
	LatestBlockMetric                              *MappedLabelsGaugeVec
	LatestProviderRelay                            *prometheus.GaugeVec
	virtualEpochMetric                             *prometheus.GaugeVec
	apiMethodCalls                                 *prometheus.GaugeVec
	endpointsHealthChecksOkMetric                  prometheus.Gauge
	endpointsHealthChecksOk                        uint64
	endpointsHealthChecksBreakdownMetric           *prometheus.GaugeVec
	lock                                           sync.Mutex
	protocolVersionMetric                          *prometheus.GaugeVec
	protocolErrorsPerProviderMetric                *prometheus.CounterVec
	providerSelectionsMetric                       *prometheus.CounterVec
	providerRelays                                 map[string]uint64
	addMethodsApiGauge                             bool
	consumerOptimizerQoSClient                     *ConsumerOptimizerQoSClient
	providerLivenessMetric                         *prometheus.GaugeVec
	blockedProviderMetric                          *MappedLabelsGaugeVec
	// Cross-validation group metrics
	crossValidationRequestsTotalMetric              *prometheus.CounterVec // lava_consumer_cross_validation_requests_total        {spec, apiInterface, method}
	crossValidationSuccessTotalMetric               *prometheus.CounterVec // lava_consumer_cross_validation_success_total         {spec, apiInterface, method}
	crossValidationFailedTotalMetric                *prometheus.CounterVec // lava_consumer_cross_validation_failed_total          {spec, apiInterface, method}
	crossValidationProviderAgreementsTotalMetric    *prometheus.CounterVec // lava_consumer_cross_validation_provider_agreements_total    {spec, apiInterface, method, provider_address}
	crossValidationProviderDisagreementsTotalMetric *prometheus.CounterVec // lava_consumer_cross_validation_provider_disagreements_total {spec, apiInterface, method, provider_address}
	selectionRNGValueGauge                         *prometheus.GaugeVec
	// Request group metrics (labels: spec, provider_address, method)
	requestsTotalMetric      *prometheus.CounterVec
	requestsSuccessMetric    *prometheus.CounterVec
	requestsFailedMetric     *prometheus.CounterVec
	requestsReadMetric       *prometheus.CounterVec
	requestsWriteMetric      *prometheus.CounterVec
	requestsDebugTraceMetric *prometheus.CounterVec
	requestsArchiveMetric    *prometheus.CounterVec
	requestsBatchMetric      *prometheus.CounterVec
	// Latency group metrics
	latencyEndToEndHistogram  *prometheus.HistogramVec // lava_consumer_end_to_end_latency_milliseconds {spec, apiInterface, method}
	latencyProviderHistogram  *prometheus.HistogramVec // lava_consumer_provider_latency_milliseconds   {spec, apiInterface, provider_address, method}
	// Cache group metrics (labels: spec, apiInterface, method)
	cacheRequestsTotalMetric *prometheus.CounterVec  // lava_consumer_cache_requests_total
	cacheSuccessTotalMetric  *prometheus.CounterVec  // lava_consumer_cache_success_total
	cacheFailedTotalMetric   *prometheus.CounterVec  // lava_consumer_cache_failed_total
	cacheLatencyHistogram    *prometheus.HistogramVec // lava_consumer_cache_latency_milliseconds
	// Incident group metrics
	incidentNodeErrorsTotalMetric     *prometheus.CounterVec // labels: spec, apiInterface, provider_address, method
	incidentProtocolErrorsTotalMetric *prometheus.CounterVec // labels: spec, apiInterface, provider_address, method
	incidentRetriesTotalMetric        *prometheus.CounterVec // labels: spec, apiInterface, method
	incidentRetriesSuccessMetric      *prometheus.CounterVec // labels: spec, apiInterface, method
	incidentRetriesFailedMetric       *prometheus.CounterVec // labels: spec, apiInterface, method
	incidentConsistencyTotalMetric    *prometheus.CounterVec // labels: spec, apiInterface, method
	incidentConsistencySuccessMetric  *prometheus.CounterVec // labels: spec, apiInterface, method
	incidentConsistencyFailedMetric   *prometheus.CounterVec // labels: spec, apiInterface, method
	incidentHedgeTotalMetric          *prometheus.CounterVec // labels: spec, apiInterface, method
	incidentHedgeSuccessMetric        *prometheus.CounterVec // labels: spec, apiInterface, method
	incidentHedgeFailedMetric         *prometheus.CounterVec // labels: spec, apiInterface, method
}

type ConsumerMetricsManagerOptions struct {
	NetworkAddress             string
	AddMethodsApiGauge         bool
	EnableQoSListener          bool
	ConsumerOptimizerQoSClient *ConsumerOptimizerQoSClient
}

// ProviderSelectionScores contains all scores for a provider at time of selection
type ProviderSelectionScores struct {
	ProviderAddress string
	Availability    float64 // Availability score (0-1)
	Latency         float64 // Latency score (0-1)
	Sync            float64 // Sync score (0-1)
	Stake           float64 // Stake score (0-1)
	Composite       float64 // Combined QoS score (0-1)
}

func NewConsumerMetricsManager(options ConsumerMetricsManagerOptions) *ConsumerMetricsManager {
	if options.NetworkAddress == DisabledFlagOption {
		utils.LavaFormatWarning("prometheus endpoint inactive, option is disabled", nil)
		return nil
	}

	totalCURequestedMetric := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "lava_consumer_total_cu_requested",
		Help: "The total number of CUs requested by the consumer over time.",
	}, []string{"spec", "apiInterface"})


	totalWsSubscriptionRequestsMetric := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "lava_consumer_total_ws_subscription_requests",
		Help: "The total number of websocket subscription requests over time per chain id per api interface.",
	}, []string{"spec", "apiInterface"})

	totalFailedWsSubscriptionRequestsMetric := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "lava_consumer_total_failed_ws_subscription_requests",
		Help: "The total number of failed websocket subscription requests over time per chain id per api interface.",
	}, []string{"spec", "apiInterface"})

	totalDuplicatedWsSubscriptionRequestsMetric := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "lava_consumer_total_duplicated_ws_subscription_requests",
		Help: "The total number of duplicated webscket subscription requests over time per chain id per api interface.",
	}, []string{"spec", "apiInterface"})

	totalLoLSuccessMetric := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "lava_consumer_total_lol_successes",
		Help: "The total number of requests sent to lava over lava successfully",
	})

	totalLoLErrorsMetric := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "lava_consumer_total_lol_errors",
		Help: "The total number of requests sent to lava over lava and failed",
	})

	totalWebSocketConnectionsActive := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "lava_consumer_total_websocket_connections_active",
		Help: "The total number of currently active websocket connections with users",
	}, []string{"spec", "apiInterface"})

	totalWsSubscriptionDisconnectMetric := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "lava_consumer_total_ws_subscription_disconnect",
		Help: "The total number of websocket subscription disconnects over time per chain id per api interface per disconnect reason.",
	}, []string{"spec", "apiInterface", "disconnectReason"})

	blockMetric := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "lava_consumer_latest_block",
		Help: "The latest block measured",
	}, []string{"spec"})

	providerLivenessMetric := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "lava_consumer_provider_liveness",
		Help: "The liveness of connected provider based on probe",
	}, []string{"spec", "provider_address", "provider_endpoint"})

	blockedProviderMetricLabels := []string{"spec", "apiInterface", "provider_address"}
	if ShowProviderEndpointInMetrics {
		blockedProviderMetricLabels = append(blockedProviderMetricLabels, "provider_endpoint")
	}

	blockedProviderMetric := NewMappedLabelsGaugeVec(MappedLabelsMetricOpts{
		Name:   "lava_consumer_provider_blocked",
		Help:   "Is provider blocked. 1-blocked, 0-not blocked",
		Labels: blockedProviderMetricLabels,
	})

	qosMetricLabels := []string{"spec", "apiInterface", "provider_address", "qos_metric"}
	if ShowProviderEndpointInMetrics {
		qosMetricLabels = append(qosMetricLabels, "provider_endpoint")
	}
	qosMetric := NewMappedLabelsGaugeVec(MappedLabelsMetricOpts{
		Name:   "lava_consumer_qos_metrics",
		Help:   "The QOS metrics per provider for current epoch for the session with the most relays.",
		Labels: qosMetricLabels,
	})

	selectionStatsMetricLabels := []string{"spec", "apiInterface", "provider_address", "selection_metric"}
	if ShowProviderEndpointInMetrics {
		selectionStatsMetricLabels = append(selectionStatsMetricLabels, "provider_endpoint")
	}
	selectionStatsMetric := NewMappedLabelsGaugeVec(MappedLabelsMetricOpts{
		Name:   "lava_consumer_selection_stats",
		Help:   "The provider selection statistics showing normalized scores used in provider selection algorithm.",
		Labels: selectionStatsMetricLabels,
	})

	providerReputationMetricLabels := []string{"spec", "provider_address", "qos_metric"}
	if ShowProviderEndpointInMetrics {
		providerReputationMetricLabels = append(providerReputationMetricLabels, "provider_endpoint")
	}
	providerReputationMetric := NewMappedLabelsGaugeVec(MappedLabelsMetricOpts{
		Name:   "lava_consumer_provider_reputation_metrics",
		Help:   "The provider reputation metrics per provider",
		Labels: providerReputationMetricLabels,
	})

	latestBlockMetricLabels := []string{"spec", "provider_address", "apiInterface"}
	if ShowProviderEndpointInMetrics {
		latestBlockMetricLabels = append(latestBlockMetricLabels, "provider_endpoint")
	}
	latestBlockMetric := NewMappedLabelsGaugeVec(MappedLabelsMetricOpts{
		Name:   "lava_consumer_latest_provider_block",
		Help:   "The latest block reported by provider",
		Labels: latestBlockMetricLabels,
	})

	latestProviderRelay := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "lava_consumer_latest_provider_relay_time",
		Help: "The latest time we sent a relay to provider",
	}, []string{"spec", "provider_address", "apiInterface"})

	virtualEpochMetric := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "lava_consumer_virtual_epoch",
		Help: "The current virtual epoch measured",
	}, []string{"spec"})

	endpointsHealthChecksOkMetric := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "lava_consumer_overall_health",
		Help: "At least one endpoint is healthy",
	})

	endpointsHealthChecksBreakdownMetric := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "lava_consumer_overall_health_breakdown",
		Help: "Health check status per chain. At least one endpoint is healthy",
	}, []string{"spec", "apiInterface"})

	apiSpecificsMetric := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "lava_consumer_api_specifics",
		Help: "api usage specifics",
	}, []string{"spec", "apiInterface", "method"})

	endpointsHealthChecksOkMetric.Set(1)
	protocolVersionMetric := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "lava_consumer_protocol_version",
		Help: "The current running lavap version for the process. major := version / 1000000, minor := (version / 1000) % 1000, patch := version % 1000",
	}, []string{"version"})

	protocolErrorsPerProviderMetric := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "lava_consumer_protocol_errors_per_provider",
		Help: "The number of protocol errors per provider and spec",
	}, []string{"spec", "provider_address"})

	crossValidationLabels := []string{"spec", "apiInterface", "method"}
	crossValidationProviderLabels := []string{"spec", "apiInterface", "method", "provider_address"}
	crossValidationRequestsTotalMetric := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "lava_consumer_cross_validation_requests_total",
		Help: "Total number of cross-validated requests.",
	}, crossValidationLabels)
	crossValidationSuccessTotalMetric := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "lava_consumer_cross_validation_success_total",
		Help: "Total number of cross-validated requests that reached consensus.",
	}, crossValidationLabels)
	crossValidationFailedTotalMetric := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "lava_consumer_cross_validation_failed_total",
		Help: "Total number of cross-validated requests that failed to reach consensus.",
	}, crossValidationLabels)
	crossValidationProviderAgreementsTotalMetric := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "lava_consumer_cross_validation_provider_agreements_total",
		Help: "Total number of times a provider agreed with the cross-validation consensus.",
	}, crossValidationProviderLabels)
	crossValidationProviderDisagreementsTotalMetric := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "lava_consumer_cross_validation_provider_disagreements_total",
		Help: "Total number of times a provider's response disagreed with the cross-validation consensus.",
	}, crossValidationProviderLabels)

	providerSelectionsMetric := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "lava_consumer_provider_selections",
		Help: "The total number of times each provider was selected for relay (before request attempt)",
	}, []string{"spec", "provider_address"})

	selectionRNGValueGauge := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "lava_consumer_selection_rng_value",
		Help: "Last RNG value used for provider selection",
	}, []string{"spec"})

	requestLabels := []string{"spec", "apiInterface", "provider_address", "method"}
	requestsTotalMetric := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "lava_consumer_requests_total",
		Help: "Total number of requests made by the consumer.",
	}, requestLabels)
	requestsSuccessMetric := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "lava_consumer_requests_success_total",
		Help: "Total number of successful requests made by the consumer.",
	}, requestLabels)
	requestsFailedMetric := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "lava_consumer_requests_failed_total",
		Help: "Total number of failed requests made by the consumer.",
	}, requestLabels)
	requestsReadMetric := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "lava_consumer_requests_read_total",
		Help: "Total number of read (stateful=0) requests made by the consumer.",
	}, requestLabels)
	requestsWriteMetric := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "lava_consumer_requests_write_total",
		Help: "Total number of write (stateful=1) requests made by the consumer.",
	}, requestLabels)
	requestsDebugTraceMetric := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "lava_consumer_requests_debug_trace_total",
		Help: "Total number of debug/trace addon requests made by the consumer.",
	}, requestLabels)
	requestsArchiveMetric := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "lava_consumer_requests_archive_total",
		Help: "Total number of archive requests made by the consumer.",
	}, requestLabels)
	requestsBatchMetric := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "lava_consumer_requests_batch_total",
		Help: "Total number of batch requests made by the consumer.",
	}, requestLabels)

	// Incident group metrics
	incidentLabels := []string{"spec", "apiInterface", "provider_address", "method"}
	incidentShortLabels := []string{"spec", "apiInterface", "method"}
	incidentNodeErrorsTotalMetric := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "lava_consumer_node_errors_total",
		Help: "Total number of node errors received from providers.",
	}, incidentLabels)
	incidentProtocolErrorsTotalMetric := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "lava_consumer_protocol_errors_total",
		Help: "Total number of protocol errors (connection/session failures) per provider.",
	}, incidentLabels)
	incidentRetriesTotalMetric := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "lava_consumer_retries_total",
		Help: "Total number of retry attempts triggered by the consumer.",
	}, incidentShortLabels)
	incidentRetriesSuccessMetric := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "lava_consumer_retries_success_total",
		Help: "Total number of retry attempts that eventually led to a successful response.",
	}, incidentShortLabels)
	incidentRetriesFailedMetric := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "lava_consumer_retries_failed_total",
		Help: "Total number of retry attempts that did not lead to a successful response.",
	}, incidentShortLabels)
	incidentConsistencyTotalMetric := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "lava_consumer_consistency_total",
		Help: "Total number of requests that triggered consistency (seenBlock) enforcement.",
	}, incidentShortLabels)
	incidentConsistencySuccessMetric := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "lava_consumer_consistency_success_total",
		Help: "Total number of consistency-enforced requests that succeeded.",
	}, incidentShortLabels)
	incidentConsistencyFailedMetric := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "lava_consumer_consistency_failed_total",
		Help: "Total number of consistency-enforced requests that failed.",
	}, incidentShortLabels)
	incidentHedgeTotalMetric := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "lava_consumer_hedge_total",
		Help: "Total number of hedge (batch-ticker) relays sent.",
	}, incidentShortLabels)
	incidentHedgeSuccessMetric := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "lava_consumer_hedge_success_total",
		Help: "Total number of hedged requests that ultimately succeeded.",
	}, incidentShortLabels)
	incidentHedgeFailedMetric := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "lava_consumer_hedge_failed_total",
		Help: "Total number of hedged requests that ultimately failed.",
	}, incidentShortLabels)

	var consumerLatencyBuckets = []float64{10, 25, 50, 75, 100, 150, 200, 350, 500, 750, 1000, 1500, 2000, 5000, 10000, 30000}

	latencyEndToEndHistogram := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "lava_consumer_end_to_end_latency_milliseconds",
		Help:    "Distribution of end-to-end relay latency seen by the consumer, from relay start to result, in milliseconds.",
		Buckets: consumerLatencyBuckets,
	}, []string{"spec", "apiInterface", "method"})

	latencyProviderHistogram := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "lava_consumer_provider_latency_milliseconds",
		Help:    "Distribution of network round-trip latency to each provider, in milliseconds.",
		Buckets: consumerLatencyBuckets,
	}, []string{"spec", "apiInterface", "provider_address", "method"})

	cacheLabels := []string{"spec", "apiInterface", "method"}
	cacheRequestsTotalMetric := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "lava_consumer_cache_requests_total",
		Help: "Total number of cache lookup attempts.",
	}, cacheLabels)
	cacheSuccessTotalMetric := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "lava_consumer_cache_success_total",
		Help: "Total number of cache lookups that returned a cached response (hits).",
	}, cacheLabels)
	cacheFailedTotalMetric := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "lava_consumer_cache_failed_total",
		Help: "Total number of cache lookups that did not find a cached response (misses).",
	}, cacheLabels)
	cacheLatencyHistogram := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "lava_consumer_cache_latency_milliseconds",
		Help:    "Distribution of cache lookup latency in milliseconds.",
		Buckets: consumerLatencyBuckets,
	}, cacheLabels)

	// Register the metrics with the Prometheus registry.
	// Use a helper function to handle AlreadyRegisteredError gracefully
	registerMetric := func(c prometheus.Collector) {
		if err := prometheus.Register(c); err != nil {
			are := &prometheus.AlreadyRegisteredError{}
			if !errors.As(err, are) {
				// Only panic if it's not an "already registered" error
				panic(err)
			}
			// Metric already registered, which can happen on restart - this is fine
			utils.LavaFormatDebug("Prometheus metric already registered, reusing existing collector", utils.LogAttr("error", err))
		}
	}

	registerMetric(totalCURequestedMetric)
	registerMetric(blockMetric)
	registerMetric(latencyEndToEndHistogram)
	registerMetric(latencyProviderHistogram)
	registerMetric(providerLivenessMetric)
	// qosMetric, providerReputationMetric, blockedProviderMetric, selectionStatsMetric,
	// and latestBlockMetric are already registered inside NewMappedLabelsGaugeVec.
	registerMetric(latestProviderRelay)
	registerMetric(virtualEpochMetric)
	registerMetric(endpointsHealthChecksOkMetric)
	registerMetric(endpointsHealthChecksBreakdownMetric)
	registerMetric(protocolVersionMetric)
	registerMetric(totalWebSocketConnectionsActive)
	registerMetric(apiSpecificsMetric)
	registerMetric(totalWsSubscriptionRequestsMetric)
	registerMetric(totalFailedWsSubscriptionRequestsMetric)
	registerMetric(totalDuplicatedWsSubscriptionRequestsMetric)
	registerMetric(totalWsSubscriptionDisconnectMetric)
	registerMetric(totalLoLSuccessMetric)
	registerMetric(totalLoLErrorsMetric)
	registerMetric(protocolErrorsPerProviderMetric)
	registerMetric(crossValidationRequestsTotalMetric)
	registerMetric(crossValidationSuccessTotalMetric)
	registerMetric(crossValidationFailedTotalMetric)
	registerMetric(crossValidationProviderAgreementsTotalMetric)
	registerMetric(crossValidationProviderDisagreementsTotalMetric)
	registerMetric(providerSelectionsMetric)
	registerMetric(selectionRNGValueGauge)
	registerMetric(requestsTotalMetric)
	registerMetric(requestsSuccessMetric)
	registerMetric(requestsFailedMetric)
	registerMetric(requestsReadMetric)
	registerMetric(requestsWriteMetric)
	registerMetric(requestsDebugTraceMetric)
	registerMetric(requestsArchiveMetric)
	registerMetric(requestsBatchMetric)
	registerMetric(cacheRequestsTotalMetric)
	registerMetric(cacheSuccessTotalMetric)
	registerMetric(cacheFailedTotalMetric)
	registerMetric(cacheLatencyHistogram)
	registerMetric(incidentNodeErrorsTotalMetric)
	registerMetric(incidentProtocolErrorsTotalMetric)
	registerMetric(incidentRetriesTotalMetric)
	registerMetric(incidentRetriesSuccessMetric)
	registerMetric(incidentRetriesFailedMetric)
	registerMetric(incidentConsistencyTotalMetric)
	registerMetric(incidentConsistencySuccessMetric)
	registerMetric(incidentConsistencyFailedMetric)
	registerMetric(incidentHedgeTotalMetric)
	registerMetric(incidentHedgeSuccessMetric)
	registerMetric(incidentHedgeFailedMetric)

	consumerMetricsManager := &ConsumerMetricsManager{
		totalCURequestedMetric:                         totalCURequestedMetric,
		totalWsSubscriptionRequestsMetric:              totalWsSubscriptionRequestsMetric,
		totalFailedWsSubscriptionRequestsMetric:        totalFailedWsSubscriptionRequestsMetric,
		totalDuplicatedWsSubscriptionRequestsMetric:    totalDuplicatedWsSubscriptionRequestsMetric,
		totalWsSubscriptionDisconnectMetric:            totalWsSubscriptionDisconnectMetric,
		totalWebSocketConnectionsActive:                totalWebSocketConnectionsActive,
		blockMetric:                                    blockMetric,
		latencyEndToEndHistogram:                       latencyEndToEndHistogram,
		latencyProviderHistogram:                       latencyProviderHistogram,
		qosMetric:                                      qosMetric,
		providerReputationMetric:                       providerReputationMetric,
		selectionStatsMetric:                           selectionStatsMetric,
		LatestBlockMetric:                              latestBlockMetric,
		LatestProviderRelay:                            latestProviderRelay,
		providerRelays:                                 map[string]uint64{},
		virtualEpochMetric:                             virtualEpochMetric,
		endpointsHealthChecksOkMetric:                  endpointsHealthChecksOkMetric,
		endpointsHealthChecksOk:                        1,
		endpointsHealthChecksBreakdownMetric:           endpointsHealthChecksBreakdownMetric,
		protocolVersionMetric:                protocolVersionMetric,
		apiMethodCalls:                       apiSpecificsMetric,
		addMethodsApiGauge:                   options.AddMethodsApiGauge,
		totalLoLSuccessMetric:                          totalLoLSuccessMetric,
		totalLoLErrorsMetric:                           totalLoLErrorsMetric,
		consumerOptimizerQoSClient:                     options.ConsumerOptimizerQoSClient,
		protocolErrorsPerProviderMetric:                protocolErrorsPerProviderMetric,
		providerSelectionsMetric:                       providerSelectionsMetric,
		providerLivenessMetric:                         providerLivenessMetric,
		blockedProviderMetric:                          blockedProviderMetric,
		crossValidationRequestsTotalMetric:              crossValidationRequestsTotalMetric,
		crossValidationSuccessTotalMetric:               crossValidationSuccessTotalMetric,
		crossValidationFailedTotalMetric:                crossValidationFailedTotalMetric,
		crossValidationProviderAgreementsTotalMetric:    crossValidationProviderAgreementsTotalMetric,
		crossValidationProviderDisagreementsTotalMetric: crossValidationProviderDisagreementsTotalMetric,
		selectionRNGValueGauge:                         selectionRNGValueGauge,
		requestsTotalMetric:                            requestsTotalMetric,
		requestsSuccessMetric:                          requestsSuccessMetric,
		requestsFailedMetric:                           requestsFailedMetric,
		requestsReadMetric:                             requestsReadMetric,
		requestsWriteMetric:                            requestsWriteMetric,
		requestsDebugTraceMetric:                       requestsDebugTraceMetric,
		requestsArchiveMetric:                          requestsArchiveMetric,
		requestsBatchMetric:                            requestsBatchMetric,
		cacheRequestsTotalMetric:                       cacheRequestsTotalMetric,
		cacheSuccessTotalMetric:                        cacheSuccessTotalMetric,
		cacheFailedTotalMetric:                         cacheFailedTotalMetric,
		cacheLatencyHistogram:                          cacheLatencyHistogram,
		incidentNodeErrorsTotalMetric:                  incidentNodeErrorsTotalMetric,
		incidentProtocolErrorsTotalMetric:              incidentProtocolErrorsTotalMetric,
		incidentRetriesTotalMetric:                     incidentRetriesTotalMetric,
		incidentRetriesSuccessMetric:                   incidentRetriesSuccessMetric,
		incidentRetriesFailedMetric:                    incidentRetriesFailedMetric,
		incidentConsistencyTotalMetric:                 incidentConsistencyTotalMetric,
		incidentConsistencySuccessMetric:               incidentConsistencySuccessMetric,
		incidentConsistencyFailedMetric:                incidentConsistencyFailedMetric,
		incidentHedgeTotalMetric:                       incidentHedgeTotalMetric,
		incidentHedgeSuccessMetric:                     incidentHedgeSuccessMetric,
		incidentHedgeFailedMetric:                      incidentHedgeFailedMetric,
	}

	http.Handle("/metrics", promhttp.Handler())
	http.HandleFunc("/provider_optimizer_metrics", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if consumerMetricsManager.consumerOptimizerQoSClient == nil {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte("[]"))
			return
		}
		reports := consumerMetricsManager.consumerOptimizerQoSClient.GetReportsToSend()
		jsonData, err := json.Marshal(reports)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(jsonData)
	})

	overallHealthHandler := func(w http.ResponseWriter, r *http.Request) {
		statusCode := http.StatusOK
		message := "Health status OK"
		if atomic.LoadUint64(&consumerMetricsManager.endpointsHealthChecksOk) == 0 {
			statusCode = http.StatusServiceUnavailable
			message = "Unhealthy"
		}

		w.WriteHeader(statusCode)
		w.Write([]byte(message))
	}

	// Backward compatibility - old path for health check alongside new path
	http.HandleFunc("/metrics/overall-health", overallHealthHandler) // New
	http.HandleFunc("/metrics/health-overall", overallHealthHandler) // Old

	go func() {
		utils.LavaFormatInfo("prometheus endpoint listening", utils.Attribute{Key: "Listen Address", Value: options.NetworkAddress})
		http.ListenAndServe(options.NetworkAddress, nil)
	}()

	return consumerMetricsManager
}

// StartSelectionStatsUpdater starts a background goroutine that periodically updates selection stats metrics
func (pme *ConsumerMetricsManager) StartSelectionStatsUpdater(ctx context.Context, updateInterval time.Duration) {
	if pme == nil || pme.consumerOptimizerQoSClient == nil {
		return
	}

	go func() {
		ticker := time.NewTicker(updateInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				utils.LavaFormatTrace("Selection stats updater context done")
				return
			case <-ticker.C:
				pme.UpdateSelectionStatsFromOptimizerReports()
			}
		}
	}()
}

func (pme *ConsumerMetricsManager) SetCrossValidationMetric(
	chainId, apiInterface, method string,
	success bool,
	agreeingProviders, disagreeingProviders []string,
) {
	if pme == nil {
		return
	}
	pme.crossValidationRequestsTotalMetric.WithLabelValues(chainId, apiInterface, method).Inc()
	if success {
		pme.crossValidationSuccessTotalMetric.WithLabelValues(chainId, apiInterface, method).Inc()
	} else {
		pme.crossValidationFailedTotalMetric.WithLabelValues(chainId, apiInterface, method).Inc()
	}
	for _, provider := range agreeingProviders {
		pme.crossValidationProviderAgreementsTotalMetric.WithLabelValues(chainId, apiInterface, method, provider).Inc()
	}
	for _, provider := range disagreeingProviders {
		pme.crossValidationProviderDisagreementsTotalMetric.WithLabelValues(chainId, apiInterface, method, provider).Inc()
	}
}

func (pme *ConsumerMetricsManager) SetWebSocketConnectionActive(chainId string, apiInterface string, add bool) {
	if pme == nil {
		return
	}
	if add {
		pme.totalWebSocketConnectionsActive.WithLabelValues(chainId, apiInterface).Add(1)
	} else {
		pme.totalWebSocketConnectionsActive.WithLabelValues(chainId, apiInterface).Sub(1)
	}
}

func (pme *ConsumerMetricsManager) SetRelayNodeErrorMetric(chainId string, apiInterface string, providerAddress string, method string) {
	if pme == nil {
		return
	}
	pme.incidentNodeErrorsTotalMetric.WithLabelValues(chainId, apiInterface, providerAddress, method).Inc()
}

func (pme *ConsumerMetricsManager) RecordCacheResult(chainId, apiInterface, method string, hit bool, latencyMs float64) {
	if pme == nil {
		return
	}
	pme.cacheRequestsTotalMetric.WithLabelValues(chainId, apiInterface, method).Inc()
	if hit {
		pme.cacheSuccessTotalMetric.WithLabelValues(chainId, apiInterface, method).Inc()
	} else {
		pme.cacheFailedTotalMetric.WithLabelValues(chainId, apiInterface, method).Inc()
	}
	pme.cacheLatencyHistogram.WithLabelValues(chainId, apiInterface, method).Observe(latencyMs)
}

func (pme *ConsumerMetricsManager) SetBlock(block int64) {
	if pme == nil {
		return
	}
	pme.blockMetric.WithLabelValues("lava").Set(float64(block))
}

func (pme *ConsumerMetricsManager) SetRelayMetrics(relayMetric *RelayMetrics, err error) {
	if pme == nil {
		return
	}
	relayMetric.Success = err == nil
	pme.totalCURequestedMetric.WithLabelValues(relayMetric.ChainID, relayMetric.APIType).Add(float64(relayMetric.ComputeUnits))
	if pme.addMethodsApiGauge && relayMetric.ApiMethod != "" { // pme.addMethodsApiGauge never changes so its safe to read concurrently
		pme.apiMethodCalls.WithLabelValues(relayMetric.ChainID, relayMetric.APIType, relayMetric.ApiMethod).Add(1)
	}
	// Request group metrics
	reqLabels := []string{relayMetric.ChainID, relayMetric.APIType, relayMetric.ProviderAddress, relayMetric.ApiMethod}
	pme.requestsTotalMetric.WithLabelValues(reqLabels...).Inc()
	if relayMetric.Success {
		pme.requestsSuccessMetric.WithLabelValues(reqLabels...).Inc()
	} else {
		pme.requestsFailedMetric.WithLabelValues(reqLabels...).Inc()
	}
	if relayMetric.IsBatch {
		pme.requestsBatchMetric.WithLabelValues(reqLabels...).Inc()
	} else {
		if relayMetric.IsWrite {
			pme.requestsWriteMetric.WithLabelValues(reqLabels...).Inc()
		} else {
			pme.requestsReadMetric.WithLabelValues(reqLabels...).Inc()
		}
		if relayMetric.IsDebugTrace {
			pme.requestsDebugTraceMetric.WithLabelValues(reqLabels...).Inc()
		}
		if relayMetric.IsArchive {
			pme.requestsArchiveMetric.WithLabelValues(reqLabels...).Inc()
		}
	}
}


func (pme *ConsumerMetricsManager) RecordEndToEndLatency(chainId string, apiInterface string, method string, latencyMs float64) {
	if pme == nil {
		return
	}
	pme.latencyEndToEndHistogram.WithLabelValues(chainId, apiInterface, method).Observe(latencyMs)
}

func (pme *ConsumerMetricsManager) RecordProviderLatency(chainId string, apiInterface string, providerAddress string, method string, latencyMs float64) {
	if pme == nil {
		return
	}
	pme.latencyProviderHistogram.WithLabelValues(chainId, apiInterface, providerAddress, method).Observe(latencyMs)
}

func (pme *ConsumerMetricsManager) RecordHedgeRelaySent(chainId string, apiInterface string, method string) {
	if pme == nil {
		return
	}
	pme.incidentHedgeTotalMetric.WithLabelValues(chainId, apiInterface, method).Inc()
}


func (pme *ConsumerMetricsManager) SetQOSMetrics(chainId string, apiInterface string, providerAddress string, providerEndpoint string, qos *pairingtypes.QualityOfServiceReport, reputation *pairingtypes.QualityOfServiceReport, latestBlock int64, relays uint64, relayLatency time.Duration, sessionSuccessful bool) {
	if pme == nil {
		return
	}
	pme.lock.Lock()
	defer pme.lock.Unlock()
	providerRelaysKey := providerAddress + apiInterface
	existingRelays, found := pme.providerRelays[providerRelaysKey]
	if found && existingRelays >= relays {
		// do not add Qos metrics there's another session with more statistics
		return
	}

	pme.LatestProviderRelay.WithLabelValues(chainId, providerAddress, apiInterface).SetToCurrentTime()
	// update existing relays
	pme.providerRelays[providerRelaysKey] = relays
	setMetricsForQos := func(qosArg *pairingtypes.QualityOfServiceReport, metric *MappedLabelsGaugeVec, apiInterfaceArg string, providerEndpoint string) {
		if qosArg == nil {
			return
		}
		availability, err := qosArg.Availability.Float64()
		if err == nil {
			labels := map[string]string{"spec": chainId, "provider_address": providerAddress, "provider_endpoint": providerEndpoint, "qos_metric": AvailabilityLabel}
			if apiInterfaceArg != "" {
				labels["apiInterface"] = apiInterface
			}
			metric.WithLabelValues(labels).Set(availability)
		}
		sync, err := qosArg.Sync.Float64()
		if err == nil {
			labels := map[string]string{"spec": chainId, "provider_address": providerAddress, "provider_endpoint": providerEndpoint, "qos_metric": SyncLabel}
			if apiInterfaceArg != "" {
				labels["apiInterface"] = apiInterface
			}
			metric.WithLabelValues(labels).Set(sync)
		}
		latency, err := qosArg.Latency.Float64()
		if err == nil {
			labels := map[string]string{"spec": chainId, "provider_address": providerAddress, "provider_endpoint": providerEndpoint, "qos_metric": LatencyLabel}
			if apiInterfaceArg != "" {
				labels["apiInterface"] = apiInterface
			}
			metric.WithLabelValues(labels).Set(latency)
		}
	}
	setMetricsForQos(qos, pme.qosMetric, apiInterface, providerEndpoint)
	setMetricsForQos(reputation, pme.providerReputationMetric, "", providerEndpoint) // it's one api interface for all of them

	labels := map[string]string{"spec": chainId, "provider_address": providerAddress, "apiInterface": apiInterface, "provider_endpoint": providerEndpoint}
	pme.LatestBlockMetric.WithLabelValues(labels).Set(float64(latestBlock))
}

// SetSelectionStatsMetrics sets the selection statistics metrics for a provider
// These metrics show the normalized scores used in the provider selection algorithm
func (pme *ConsumerMetricsManager) SetSelectionStatsMetrics(chainId string, apiInterface string, providerAddress string, providerEndpoint string, availability, latency, sync, stake, composite float64) {
	if pme == nil {
		return
	}
	pme.lock.Lock()
	defer pme.lock.Unlock()

	// Set availability score
	availabilityLabels := map[string]string{"spec": chainId, "provider_address": providerAddress, "provider_endpoint": providerEndpoint, "selection_metric": SelectionAvailabilityLabel, "apiInterface": apiInterface}
	pme.selectionStatsMetric.WithLabelValues(availabilityLabels).Set(availability)

	// Set latency score
	latencyLabels := map[string]string{"spec": chainId, "provider_address": providerAddress, "provider_endpoint": providerEndpoint, "selection_metric": SelectionLatencyLabel, "apiInterface": apiInterface}
	pme.selectionStatsMetric.WithLabelValues(latencyLabels).Set(latency)

	// Set sync score
	syncLabels := map[string]string{"spec": chainId, "provider_address": providerAddress, "provider_endpoint": providerEndpoint, "selection_metric": SelectionSyncLabel, "apiInterface": apiInterface}
	pme.selectionStatsMetric.WithLabelValues(syncLabels).Set(sync)

	// Set stake score
	stakeLabels := map[string]string{"spec": chainId, "provider_address": providerAddress, "provider_endpoint": providerEndpoint, "selection_metric": SelectionStakeLabel, "apiInterface": apiInterface}
	pme.selectionStatsMetric.WithLabelValues(stakeLabels).Set(stake)

	// Set composite score
	compositeLabels := map[string]string{"spec": chainId, "provider_address": providerAddress, "provider_endpoint": providerEndpoint, "selection_metric": SelectionCompositeLabel, "apiInterface": apiInterface}
	pme.selectionStatsMetric.WithLabelValues(compositeLabels).Set(composite)
}

func (pme *ConsumerMetricsManager) SetVirtualEpoch(virtualEpoch uint64) {
	if pme == nil {
		return
	}
	pme.virtualEpochMetric.WithLabelValues("lava").Set(float64(virtualEpoch))
}

func (pme *ConsumerMetricsManager) UpdateHealthCheckStatus(status bool) {
	if pme == nil {
		return
	}
	var value float64 = 0
	if status {
		value = 1
	}
	pme.endpointsHealthChecksOkMetric.Set(value)
	atomic.StoreUint64(&pme.endpointsHealthChecksOk, uint64(value))
}

func (pme *ConsumerMetricsManager) UpdateHealthcheckStatusBreakdown(chainId, apiInterface string, status bool) {
	if pme == nil {
		return
	}
	var value float64 = 0
	if status {
		value = 1
	}

	pme.endpointsHealthChecksBreakdownMetric.WithLabelValues(chainId, apiInterface).Set(value)
}

func (pme *ConsumerMetricsManager) ResetSessionRelatedMetrics() {
	if pme == nil {
		return
	}
	pme.lock.Lock()
	defer pme.lock.Unlock()
	pme.qosMetric.Reset()
	pme.providerReputationMetric.Reset()
	pme.selectionStatsMetric.Reset()
	pme.providerRelays = map[string]uint64{}
}

func (pme *ConsumerMetricsManager) ResetBlockedProvidersMetrics(chainId, apiInterface string, providers map[string]string) {
	if pme == nil {
		return
	}
	pme.lock.Lock()
	defer pme.lock.Unlock()
	pme.blockedProviderMetric.Reset()
	for provider, endpoint := range providers {
		labels := map[string]string{"spec": chainId, "apiInterface": apiInterface, "provider_address": provider, "provider_endpoint": endpoint}
		pme.blockedProviderMetric.WithLabelValues(labels).Set(0)
	}
}

func (pme *ConsumerMetricsManager) SetVersion(version string) {
	if pme == nil {
		return
	}
	SetVersionInner(pme.protocolVersionMetric, version)
}

func SetVersionInner(protocolVersionMetric *prometheus.GaugeVec, version string) {
	var major, minor, patch int
	_, err := fmt.Sscanf(version, "%d.%d.%d", &major, &minor, &patch)
	if err != nil {
		utils.LavaFormatError("Failed parsing version at metrics manager", err, utils.LogAttr("version", version))
		protocolVersionMetric.WithLabelValues("version").Set(0)
		return
	}
	combined := major*1000000 + minor*1000 + patch
	protocolVersionMetric.WithLabelValues("version").Set(float64(combined))
}

func (pme *ConsumerMetricsManager) SetProtocolError(chainId string, apiInterface string, providerAddress string, method string) {
	if pme == nil {
		return
	}
	pme.protocolErrorsPerProviderMetric.WithLabelValues(chainId, providerAddress).Inc()
	pme.incidentProtocolErrorsTotalMetric.WithLabelValues(chainId, apiInterface, providerAddress, method).Inc()
}

func (pme *ConsumerMetricsManager) RecordIncidentRetry(chainId string, apiInterface string, method string, count uint64, success bool) {
	if pme == nil {
		return
	}
	pme.incidentRetriesTotalMetric.WithLabelValues(chainId, apiInterface, method).Add(float64(count))
	if success {
		pme.incidentRetriesSuccessMetric.WithLabelValues(chainId, apiInterface, method).Inc()
	} else {
		pme.incidentRetriesFailedMetric.WithLabelValues(chainId, apiInterface, method).Inc()
	}
}

func (pme *ConsumerMetricsManager) RecordIncidentConsistency(chainId string, apiInterface string, method string, success bool) {
	if pme == nil {
		return
	}
	pme.incidentConsistencyTotalMetric.WithLabelValues(chainId, apiInterface, method).Inc()
	if success {
		pme.incidentConsistencySuccessMetric.WithLabelValues(chainId, apiInterface, method).Inc()
	} else {
		pme.incidentConsistencyFailedMetric.WithLabelValues(chainId, apiInterface, method).Inc()
	}
}

func (pme *ConsumerMetricsManager) RecordIncidentHedgeResult(chainId string, apiInterface string, method string, success bool) {
	if pme == nil {
		return
	}
	if success {
		pme.incidentHedgeSuccessMetric.WithLabelValues(chainId, apiInterface, method).Inc()
	} else {
		pme.incidentHedgeFailedMetric.WithLabelValues(chainId, apiInterface, method).Inc()
	}
}

// SetProviderSelected records when a provider is selected and updates score gauges for all providers
func (pme *ConsumerMetricsManager) SetProviderSelected(chainId string, providerAddress string, allProviderScores []ProviderSelectionScores, rngValue float64) {
	if pme == nil {
		return
	}
	// Increment selection counter for the selected provider
	pme.providerSelectionsMetric.WithLabelValues(chainId, providerAddress).Inc()
	pme.selectionRNGValueGauge.WithLabelValues(chainId).Set(rngValue)

	// Find the selected provider's composite score for the optimizer
	var selectedQoSScore float64
	foundSelectedProvider := false
	for _, scores := range allProviderScores {
		if scores.ProviderAddress == providerAddress {
			selectedQoSScore = scores.Composite
			foundSelectedProvider = true
			break
		}
	}

	// If selected provider wasn't in the scores list, log for debugging
	if !foundSelectedProvider && len(allProviderScores) > 0 {
		// Collect addresses in scores for comparison
		scoreAddresses := make([]string, 0, len(allProviderScores))
		for _, s := range allProviderScores {
			scoreAddresses = append(scoreAddresses, s.ProviderAddress)
		}
		utils.LavaFormatWarning("Selected provider not found in scores list",
			nil,
			utils.LogAttr("selectedProvider", providerAddress),
			utils.LogAttr("scoreAddresses", scoreAddresses),
			utils.LogAttr("numScores", len(allProviderScores)),
			utils.LogAttr("chainId", chainId),
		)
	}
	if len(allProviderScores) == 0 {
		utils.LavaFormatWarning("Selection scores list empty for provider selection",
			nil,
			utils.LogAttr("provider", providerAddress),
			utils.LogAttr("chainId", chainId),
		)
	}
	if foundSelectedProvider && len(allProviderScores) > 0 {
		if math.IsNaN(selectedQoSScore) || math.IsInf(selectedQoSScore, 0) {
			utils.LavaFormatWarning("Selected provider composite score is invalid",
				nil,
				utils.LogAttr("provider", providerAddress),
				utils.LogAttr("chainId", chainId),
				utils.LogAttr("qosScore", selectedQoSScore),
				utils.LogAttr("rngValue", rngValue),
			)
			selectedQoSScore = 0
		} else if selectedQoSScore == 0 {
			utils.LavaFormatWarning("Selected provider composite score is zero",
				nil,
				utils.LogAttr("provider", providerAddress),
				utils.LogAttr("chainId", chainId),
				utils.LogAttr("rngValue", rngValue),
			)
		}
	}

	// Forward to optimizer QoS client for additional tracking
	pme.consumerOptimizerQoSClient.SetProviderSelected(providerAddress, chainId, selectedQoSScore, rngValue)
}

func (pme *ConsumerMetricsManager) SetWsSubscriptionRequestMetric(chainId string, apiInterface string) {
	if pme == nil {
		return
	}
	pme.totalWsSubscriptionRequestsMetric.WithLabelValues(chainId, apiInterface).Inc()
}

func (pme *ConsumerMetricsManager) SetFailedWsSubscriptionRequestMetric(chainId string, apiInterface string) {
	if pme == nil {
		return
	}
	pme.totalFailedWsSubscriptionRequestsMetric.WithLabelValues(chainId, apiInterface).Inc()
}

func (pme *ConsumerMetricsManager) SetDuplicatedWsSubscriptionRequestMetric(chainId string, apiInterface string) {
	if pme == nil {
		return
	}
	pme.totalDuplicatedWsSubscriptionRequestsMetric.WithLabelValues(chainId, apiInterface).Inc()
}

func (pme *ConsumerMetricsManager) SetWsSubscriptioDisconnectRequestMetric(chainId string, apiInterface string, disconnectReason string) {
	if pme == nil {
		return
	}
	pme.totalWsSubscriptionDisconnectMetric.WithLabelValues(chainId, apiInterface, disconnectReason).Inc()
}

func (pme *ConsumerMetricsManager) SetLoLResponse(success bool) {
	if pme == nil {
		return
	}
	if success {
		pme.totalLoLSuccessMetric.Inc()
	} else {
		pme.totalLoLErrorsMetric.Inc()
	}
}

func (pme *ConsumerMetricsManager) SetProviderLiveness(chainId string, providerAddress string, providerEndpoint string, isAlive bool) {
	if pme == nil {
		return
	}

	var value float64 = 0
	if isAlive {
		value = 1
	}

	pme.providerLivenessMetric.WithLabelValues(chainId, providerAddress, providerEndpoint).Set(value)
}

func (pme *ConsumerMetricsManager) SetBlockedProvider(chainId, apiInterface, providerAddress, providerEndpoint string, isBlocked bool) {
	if pme == nil {
		return
	}
	var value float64 = 0
	if isBlocked {
		value = 1
	}
	labels := map[string]string{"spec": chainId, "apiInterface": apiInterface, "provider_address": providerAddress, "provider_endpoint": providerEndpoint}
	pme.lock.Lock()
	defer pme.lock.Unlock()
	pme.blockedProviderMetric.WithLabelValues(labels).Set(value)
}

// UpdateSelectionStatsFromOptimizerReports updates the selection stats metrics from the optimizer reports
func (pme *ConsumerMetricsManager) UpdateSelectionStatsFromOptimizerReports() {
	if pme == nil || pme.consumerOptimizerQoSClient == nil {
		return
	}

	reports := pme.consumerOptimizerQoSClient.GetReportsToSend()
	pme.lock.Lock()
	defer pme.lock.Unlock()

	for _, report := range reports {
		providerEndpoint := "" // TODO: Get provider endpoint if needed

		// Set selection stats metrics
		availabilityLabels := map[string]string{
			"spec":              report.ChainId,
			"provider_address":  report.ProviderAddress,
			"provider_endpoint": providerEndpoint,
			"selection_metric":  SelectionAvailabilityLabel,
			"apiInterface":      "", // API interface not available in optimizer reports
		}
		pme.selectionStatsMetric.WithLabelValues(availabilityLabels).Set(report.SelectionAvailability)

		latencyLabels := map[string]string{
			"spec":              report.ChainId,
			"provider_address":  report.ProviderAddress,
			"provider_endpoint": providerEndpoint,
			"selection_metric":  SelectionLatencyLabel,
			"apiInterface":      "",
		}
		pme.selectionStatsMetric.WithLabelValues(latencyLabels).Set(report.SelectionLatency)

		syncLabels := map[string]string{
			"spec":              report.ChainId,
			"provider_address":  report.ProviderAddress,
			"provider_endpoint": providerEndpoint,
			"selection_metric":  SelectionSyncLabel,
			"apiInterface":      "",
		}
		pme.selectionStatsMetric.WithLabelValues(syncLabels).Set(report.SelectionSync)

		stakeLabels := map[string]string{
			"spec":              report.ChainId,
			"provider_address":  report.ProviderAddress,
			"provider_endpoint": providerEndpoint,
			"selection_metric":  SelectionStakeLabel,
			"apiInterface":      "",
		}
		pme.selectionStatsMetric.WithLabelValues(stakeLabels).Set(report.SelectionStake)

		compositeLabels := map[string]string{
			"spec":              report.ChainId,
			"provider_address":  report.ProviderAddress,
			"provider_endpoint": providerEndpoint,
			"selection_metric":  SelectionCompositeLabel,
			"apiInterface":      "",
		}
		pme.selectionStatsMetric.WithLabelValues(compositeLabels).Set(report.SelectionComposite)
	}
}

func (pme *ConsumerMetricsManager) handleOptimizerQoS(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var report OptimizerQoSReportToSend
	if err := json.NewDecoder(r.Body).Decode(&report); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Process the received QoS report here
	utils.LavaFormatDebug("Received QoS report",
		utils.LogAttr("provider", report.ProviderAddress),
		utils.LogAttr("chain_id", report.ChainId),
		utils.LogAttr("sync_score", report.SyncScore),
		utils.LogAttr("availability_score", report.AvailabilityScore),
		utils.LogAttr("latency_score", report.LatencyScore),
	)

	w.WriteHeader(http.StatusOK)
}
