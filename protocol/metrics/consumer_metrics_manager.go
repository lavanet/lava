package metrics

import (
	"encoding/json"
	"fmt"
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

type LatencyTracker struct {
	AverageLatency time.Duration // in nano seconds (time.Since result)
	TotalRequests  int
}

func (lt *LatencyTracker) AddLatency(latency time.Duration) {
	lt.TotalRequests++
	weight := 1.0 / float64(lt.TotalRequests)
	// Calculate the weighted average of the current average latency and the new latency
	lt.AverageLatency = time.Duration(float64(lt.AverageLatency)*(1-weight) + float64(latency)*weight)
}

type ConsumerMetricsManager struct {
	totalCURequestedMetric                      *prometheus.CounterVec
	totalRelaysRequestedMetric                  *prometheus.CounterVec
	totalErroredMetric                          *prometheus.CounterVec
	totalNodeErroredMetric                      *prometheus.CounterVec
	totalNodeErroredRecoveredSuccessfullyMetric *prometheus.CounterVec
	totalNodeErroredRecoveryAttemptsMetric      *prometheus.CounterVec
	totalRelaysSentToProvidersMetric            *prometheus.CounterVec
	totalRelaysSentByNewBatchTickerMetric       *prometheus.CounterVec
	totalWsSubscriptionRequestsMetric           *prometheus.CounterVec
	totalFailedWsSubscriptionRequestsMetric     *prometheus.CounterVec
	totalWsSubscriptionDisconnectMetric         *prometheus.CounterVec
	totalDuplicatedWsSubscriptionRequestsMetric *prometheus.CounterVec
	totalLoLSuccessMetric                       prometheus.Counter
	totalLoLErrorsMetric                        prometheus.Counter
	totalWebSocketConnectionsActive             *prometheus.GaugeVec
	blockMetric                                 *prometheus.GaugeVec
	latencyMetric                               *prometheus.GaugeVec
	qosMetric                                   *MappedLabelsGaugeVec
	providerReputationMetric                    *MappedLabelsGaugeVec
	LatestBlockMetric                           *MappedLabelsGaugeVec
	LatestProviderRelay                         *prometheus.GaugeVec
	virtualEpochMetric                          *prometheus.GaugeVec
	apiMethodCalls                              *prometheus.GaugeVec
	endpointsHealthChecksOkMetric               prometheus.Gauge
	endpointsHealthChecksOk                     uint64
	endpointsHealthChecksBreakdownMetric        *prometheus.GaugeVec
	lock                                        sync.Mutex
	protocolVersionMetric                       *prometheus.GaugeVec
	requestsPerProviderMetric                   *prometheus.CounterVec
	protocolErrorsPerProviderMetric             *prometheus.CounterVec
	providerRelays                              map[string]uint64
	addMethodsApiGauge                          bool
	averageLatencyPerChain                      map[string]*LatencyTracker // key == chain Id + api interface
	averageLatencyMetric                        *prometheus.GaugeVec
	relayProcessingLatencyBeforeProvider        *prometheus.GaugeVec
	relayProcessingLatencyAfterProvider         *prometheus.GaugeVec
	averageProcessingLatency                    map[string]*LatencyTracker
	consumerOptimizerQoSClient                  *ConsumerOptimizerQoSClient
	providerLivenessMetric                      *prometheus.GaugeVec
	blockedProviderMetric                       *MappedLabelsGaugeVec
}

type ConsumerMetricsManagerOptions struct {
	NetworkAddress             string
	AddMethodsApiGauge         bool
	EnableQoSListener          bool
	ConsumerOptimizerQoSClient *ConsumerOptimizerQoSClient
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

	// Create a new GaugeVec metric to represent the TotalRelaysServiced over time.
	totalRelaysRequestedMetric := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "lava_consumer_total_relays_serviced",
		Help: "The total number of relays serviced by the consumer over time.",
	}, []string{"spec", "apiInterface"})

	totalRelaysSentByNewBatchTickerMetric := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "lava_consumer_total_relays_sent_by_batch_ticker",
		Help: "The total number of relays sent by the batch ticker",
	}, []string{"spec", "apiInterface"})

	// Create a new GaugeVec metric to represent the TotalErrored over time.
	totalErroredMetric := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "lava_consumer_total_errored",
		Help: "The total number of errors encountered by the consumer over time.",
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
		Help: "The total number of websocket subscription disconnects over time per chain id per api interface per dissconnect reason.",
	}, []string{"spec", "apiInterface", "dissconectReason"})

	blockMetric := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "lava_latest_block",
		Help: "The latest block measured",
	}, []string{"spec"})

	latencyMetric := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "lava_consumer_latency_for_request",
		Help: "The latency of requests requested by the consumer over time.",
	}, []string{"spec", "apiInterface"})

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
		Name: "virtual_epoch",
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

	averageLatencyMetric := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "lava_consumer_average_latency_in_milliseconds",
		Help: "average latency per chain id per api interface",
	}, []string{"spec", "apiInterface"})

	totalRelaysSentToProvidersMetric := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "lava_consumer_total_relays_sent_to_providers",
		Help: "The total number of relays sent to providers",
	}, []string{"spec", "apiInterface"})

	totalNodeErroredMetric := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "lava_consumer_total_node_errors_received_from_providers",
		Help: "The total number of relays sent to providers and returned a node error",
	}, []string{"spec", "apiInterface"})

	totalNodeErroredRecoveredSuccessfullyMetric := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "lava_consumer_total_node_errors_recovered_successfully",
		Help: "The total number of node errors that managed to recover using a retry",
	}, []string{"spec", "apiInterface", "attempt"})

	totalNodeErroredRecoveryAttemptsMetric := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "lava_consumer_total_node_errors_recovery_attempts",
		Help: "The total number of retries sent due to retry mechanism",
	}, []string{"spec", "apiInterface"})

	relayProcessingLatencyBeforeProvider := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "lava_relay_processing_latency_before_provider_in_micro_seconds",
		Help: "average latency of processing a successful relay before it is sent to the provider in µs (10^6)",
	}, []string{"spec", "apiInterface"})

	relayProcessingLatencyAfterProvider := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "lava_relay_processing_latency_after_provider_in_micro_seconds",
		Help: "average latency of processing a successful relay after it is received from the provider in µs (10^6)",
	}, []string{"spec", "apiInterface"})

	requestsPerProviderMetric := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "lava_requests_per_provider",
		Help: "The number of requests per provider and spec",
	}, []string{"spec", "provider_address"})

	protocolErrorsPerProviderMetric := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "lava_protocol_errors_per_provider",
		Help: "The number of protocol errors per provider and spec",
	}, []string{"spec", "provider_address"})

	// Register the metrics with the Prometheus registry.
	prometheus.MustRegister(totalCURequestedMetric)
	prometheus.MustRegister(totalRelaysRequestedMetric)
	prometheus.MustRegister(totalErroredMetric)
	prometheus.MustRegister(blockMetric)
	prometheus.MustRegister(latencyMetric)
	prometheus.MustRegister(providerLivenessMetric)
	prometheus.MustRegister(latestProviderRelay)
	prometheus.MustRegister(virtualEpochMetric)
	prometheus.MustRegister(endpointsHealthChecksOkMetric)
	prometheus.MustRegister(endpointsHealthChecksBreakdownMetric)
	prometheus.MustRegister(protocolVersionMetric)
	prometheus.MustRegister(totalRelaysSentByNewBatchTickerMetric)
	prometheus.MustRegister(totalWebSocketConnectionsActive)
	prometheus.MustRegister(apiSpecificsMetric)
	prometheus.MustRegister(averageLatencyMetric)
	prometheus.MustRegister(totalRelaysSentToProvidersMetric)
	prometheus.MustRegister(totalNodeErroredMetric)
	prometheus.MustRegister(totalNodeErroredRecoveredSuccessfullyMetric)
	prometheus.MustRegister(totalNodeErroredRecoveryAttemptsMetric)
	prometheus.MustRegister(relayProcessingLatencyBeforeProvider)
	prometheus.MustRegister(relayProcessingLatencyAfterProvider)
	prometheus.MustRegister(totalWsSubscriptionRequestsMetric)
	prometheus.MustRegister(totalFailedWsSubscriptionRequestsMetric)
	prometheus.MustRegister(totalDuplicatedWsSubscriptionRequestsMetric)
	prometheus.MustRegister(totalWsSubscriptionDisconnectMetric)
	prometheus.MustRegister(totalLoLSuccessMetric)
	prometheus.MustRegister(totalLoLErrorsMetric)
	prometheus.MustRegister(requestsPerProviderMetric)
	prometheus.MustRegister(protocolErrorsPerProviderMetric)

	consumerMetricsManager := &ConsumerMetricsManager{
		totalCURequestedMetric:                      totalCURequestedMetric,
		totalRelaysRequestedMetric:                  totalRelaysRequestedMetric,
		totalWsSubscriptionRequestsMetric:           totalWsSubscriptionRequestsMetric,
		totalFailedWsSubscriptionRequestsMetric:     totalFailedWsSubscriptionRequestsMetric,
		totalDuplicatedWsSubscriptionRequestsMetric: totalDuplicatedWsSubscriptionRequestsMetric,
		totalWsSubscriptionDisconnectMetric:         totalWsSubscriptionDisconnectMetric,
		totalWebSocketConnectionsActive:             totalWebSocketConnectionsActive,
		totalErroredMetric:                          totalErroredMetric,
		blockMetric:                                 blockMetric,
		latencyMetric:                               latencyMetric,
		qosMetric:                                   qosMetric,
		providerReputationMetric:                    providerReputationMetric,
		LatestBlockMetric:                           latestBlockMetric,
		LatestProviderRelay:                         latestProviderRelay,
		providerRelays:                              map[string]uint64{},
		averageLatencyPerChain:                      map[string]*LatencyTracker{},
		virtualEpochMetric:                          virtualEpochMetric,
		endpointsHealthChecksOkMetric:               endpointsHealthChecksOkMetric,
		endpointsHealthChecksOk:                     1,
		endpointsHealthChecksBreakdownMetric:        endpointsHealthChecksBreakdownMetric,
		protocolVersionMetric:                       protocolVersionMetric,
		averageLatencyMetric:                        averageLatencyMetric,
		totalRelaysSentByNewBatchTickerMetric:       totalRelaysSentByNewBatchTickerMetric,
		apiMethodCalls:                              apiSpecificsMetric,
		addMethodsApiGauge:                          options.AddMethodsApiGauge,
		totalNodeErroredMetric:                      totalNodeErroredMetric,
		totalNodeErroredRecoveredSuccessfullyMetric: totalNodeErroredRecoveredSuccessfullyMetric,
		totalNodeErroredRecoveryAttemptsMetric:      totalNodeErroredRecoveryAttemptsMetric,
		totalRelaysSentToProvidersMetric:            totalRelaysSentToProvidersMetric,
		relayProcessingLatencyBeforeProvider:        relayProcessingLatencyBeforeProvider,
		relayProcessingLatencyAfterProvider:         relayProcessingLatencyAfterProvider,
		averageProcessingLatency:                    map[string]*LatencyTracker{},
		totalLoLSuccessMetric:                       totalLoLSuccessMetric,
		totalLoLErrorsMetric:                        totalLoLErrorsMetric,
		consumerOptimizerQoSClient:                  options.ConsumerOptimizerQoSClient,
		requestsPerProviderMetric:                   requestsPerProviderMetric,
		protocolErrorsPerProviderMetric:             protocolErrorsPerProviderMetric,
		providerLivenessMetric:                      providerLivenessMetric,
		blockedProviderMetric:                       blockedProviderMetric,
	}

	http.Handle("/metrics", promhttp.Handler())
	http.HandleFunc("/provider_optimizer_metrics", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
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
		message := "Healthy"
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

func (pme *ConsumerMetricsManager) SetRelaySentToProviderMetric(chainId string, apiInterface string) {
	if pme == nil {
		return
	}
	pme.totalRelaysSentToProvidersMetric.WithLabelValues(chainId, apiInterface).Inc()
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

func (pme *ConsumerMetricsManager) SetRelayNodeErrorMetric(chainId string, apiInterface string) {
	if pme == nil {
		return
	}
	pme.totalNodeErroredMetric.WithLabelValues(chainId, apiInterface).Inc()
}

func (pme *ConsumerMetricsManager) SetNodeErrorRecoveredSuccessfullyMetric(chainId string, apiInterface string, attempt string) {
	if pme == nil {
		return
	}
	pme.totalNodeErroredRecoveredSuccessfullyMetric.WithLabelValues(chainId, apiInterface, attempt).Inc()
}

func (pme *ConsumerMetricsManager) SetNodeErrorAttemptMetric(chainId string, apiInterface string) {
	if pme == nil {
		return
	}
	pme.totalNodeErroredRecoveryAttemptsMetric.WithLabelValues(chainId, apiInterface).Inc()
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
	pme.latencyMetric.WithLabelValues(relayMetric.ChainID, relayMetric.APIType).Set(float64(relayMetric.Latency))
	pme.totalCURequestedMetric.WithLabelValues(relayMetric.ChainID, relayMetric.APIType).Add(float64(relayMetric.ComputeUnits))
	pme.totalRelaysRequestedMetric.WithLabelValues(relayMetric.ChainID, relayMetric.APIType).Add(1)
	if pme.addMethodsApiGauge && relayMetric.ApiMethod != "" { // pme.addMethodsApiGauge never changes so its safe to read concurrently
		pme.apiMethodCalls.WithLabelValues(relayMetric.ChainID, relayMetric.APIType, relayMetric.ApiMethod).Add(1)
	}
	if !relayMetric.Success {
		pme.totalErroredMetric.WithLabelValues(relayMetric.ChainID, relayMetric.APIType).Add(1)
	}
}

func (pme *ConsumerMetricsManager) SetRelayProcessingLatencyBeforeProvider(latency time.Duration, chainId string, apiInterface string) {
	if pme == nil {
		return
	}
	key := pme.getKeyForProcessingLatency(chainId, apiInterface, "before")
	updatedLatency := pme.updateRelayProcessingLatency(latency, key)
	pme.relayProcessingLatencyBeforeProvider.WithLabelValues(chainId, apiInterface).Set(updatedLatency)
}

func (pme *ConsumerMetricsManager) SetRelayProcessingLatencyAfterProvider(latency time.Duration, chainId string, apiInterface string) {
	if pme == nil {
		return
	}
	key := pme.getKeyForProcessingLatency(chainId, apiInterface, "after")
	updatedLatency := pme.updateRelayProcessingLatency(latency, key)
	pme.relayProcessingLatencyAfterProvider.WithLabelValues(chainId, apiInterface).Set(updatedLatency)
}

func (pme *ConsumerMetricsManager) updateRelayProcessingLatency(latency time.Duration, key string) float64 {
	pme.lock.Lock()
	defer pme.lock.Unlock()

	currentLatency, ok := pme.averageProcessingLatency[key]
	if !ok {
		currentLatency = &LatencyTracker{AverageLatency: time.Duration(0), TotalRequests: 0}
	}
	currentLatency.AddLatency(latency)
	pme.averageProcessingLatency[key] = currentLatency
	return float64(currentLatency.AverageLatency.Microseconds())
}

func (pme *ConsumerMetricsManager) SetRelaySentByNewBatchTickerMetric(chainId string, apiInterface string) {
	if pme == nil {
		return
	}
	pme.totalRelaysSentByNewBatchTickerMetric.WithLabelValues(chainId, apiInterface).Inc()
}

func (pme *ConsumerMetricsManager) getKeyForAverageLatency(chainId string, apiInterface string) string {
	return chainId + apiInterface
}

func (pme *ConsumerMetricsManager) getKeyForProcessingLatency(chainId string, apiInterface string, header string) string {
	return header + "_" + chainId + "_" + apiInterface
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

	// calculate average latency on successful sessions only and not hanging apis (transactions etc..)
	if sessionSuccessful {
		averageLatencyKey := pme.getKeyForAverageLatency(chainId, apiInterface)
		existingLatency, foundExistingLatency := pme.averageLatencyPerChain[averageLatencyKey]
		if !foundExistingLatency {
			pme.averageLatencyPerChain[averageLatencyKey] = &LatencyTracker{}
			existingLatency = pme.averageLatencyPerChain[averageLatencyKey]
		}
		existingLatency.AddLatency(relayLatency)
		pme.averageLatencyMetric.WithLabelValues(chainId, apiInterface).Set(float64(existingLatency.AverageLatency.Milliseconds()))
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

func (pme *ConsumerMetricsManager) SetProtocolError(chainId string, providerAddress string) {
	if pme == nil {
		return
	}
	pme.protocolErrorsPerProviderMetric.WithLabelValues(chainId, providerAddress).Inc()
}

func (pme *ConsumerMetricsManager) SetRequestPerProvider(chainId string, providerAddress string) {
	if pme == nil {
		return
	}
	pme.requestsPerProviderMetric.WithLabelValues(chainId, providerAddress).Inc()
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
