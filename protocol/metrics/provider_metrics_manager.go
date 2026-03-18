package metrics

import (
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/lavanet/lava/v5/utils"
	spectypes "github.com/lavanet/lava/v5/x/spec/types"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	MetricsListenFlagName         = "metrics-listen-address"
	RelayServerFlagName           = "relay-server-address"
	RelayKafkaFlagName            = "relay-kafka-address"
	RelayKafkaTopicFlagName       = "relay-kafka-topic"
	RelayKafkaUsernameFlagName    = "relay-kafka-username"
	RelayKafkaPasswordFlagName    = "relay-kafka-password"
	RelayKafkaMechanismFlagName   = "relay-kafka-mechanism"
	RelayKafkaTLSEnabledFlagName  = "relay-kafka-tls-enabled"
	RelayKafkaTLSInsecureFlagName = "relay-kafka-tls-insecure"
	DisabledFlagOption            = "disabled"
)

var ShowProviderEndpointInProviderMetrics = false

type ProviderMetricsManager struct {
	providerMetrics                      map[string]*ProviderMetrics
	lock                                 sync.RWMutex
	totalCUServicedMetric                *prometheus.CounterVec
	totalCUPaidMetric                    *prometheus.CounterVec
	totalRelaysServicedMetric            *MappedLabelsCounterVec
	totalErroredMetric                   *MappedLabelsCounterVec
	inFlightPerFunctionMetric            *MappedLabelsGaugeVec
	requestLatencyPerFunctionMetric      *prometheus.HistogramVec
	blockMetric                          *MappedLabelsGaugeVec
	lastServicedBlockTimeMetric          *prometheus.GaugeVec
	disabledChainsMetric                 *prometheus.GaugeVec
	fetchLatestFailedMetric              *prometheus.CounterVec
	fetchBlockFailedMetric               *prometheus.CounterVec
	fetchLatestSuccessMetric             *prometheus.CounterVec
	fetchBlockSuccessMetric              *prometheus.CounterVec
	protocolVersionMetric                *prometheus.GaugeVec
	virtualEpochMetric                   *prometheus.GaugeVec
	endpointsHealthChecksOkMetric        prometheus.Gauge
	endpointsHealthChecksOk              uint64
	endpointsHealthChecksBreakdownMetric *prometheus.GaugeVec
	relaysMonitors                       map[string]*RelaysMonitor
	relaysMonitorsLock                   sync.RWMutex
	frozenStatusMetric                   *prometheus.GaugeVec
	jailStatusMetric                     *prometheus.GaugeVec
	jailedCountMetric                    *prometheus.GaugeVec
	loadRateMetric                       *prometheus.GaugeVec
	providerLatencyMetric                *prometheus.HistogramVec
	providerEndToEndLatencyMetric        *prometheus.HistogramVec
}

func NewProviderMetricsManager(networkAddress string) *ProviderMetricsManager {
	if networkAddress == DisabledFlagOption {
		utils.LavaFormatWarning("prometheus endpoint inactive, option is disabled", nil)
		return nil
	}
	totalCUServicedMetric := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "lava_provider_total_cu_serviced",
		Help: "The total number of CUs serviced by the provider over time.",
	}, []string{"spec", "apiInterface"})

	totalRelaysServicedMetric := NewMappedLabelsCounterVec(MappedLabelsMetricOpts{
		Name:       "lava_provider_total_relays_serviced",
		Help:       "The total number of relays serviced by the provider over time, by function.",
		Labels:     []string{"spec", "apiInterface", "function"},
		Registerer: prometheus.DefaultRegisterer,
	})

	totalErroredMetric := NewMappedLabelsCounterVec(MappedLabelsMetricOpts{
		Name:       "lava_provider_total_errored",
		Help:       "The total number of errored relays by the provider over time, by function.",
		Labels:     []string{"spec", "apiInterface", "function"},
		Registerer: prometheus.DefaultRegisterer,
	})

	totalInFlightPerFunctionMetric := NewMappedLabelsGaugeVec(MappedLabelsMetricOpts{
		Name:       "lava_provider_requests_in_flight",
		Help:       "The number of relays currently being handled, by function.",
		Labels:     []string{"spec", "apiInterface", "function"},
		Registerer: prometheus.DefaultRegisterer,
	})

	requestLatencyPerFunctionMetric := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "lava_provider_request_latency_milliseconds",
		Help:    "Distribution of relay latency per function in milliseconds.",
		Buckets: latencyBuckets,
	}, []string{"spec", "apiInterface", "function"})

	// Create a new GaugeVec metric to represent the TotalCUPaid over time.
	totalCUPaidMetric := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "lava_provider_total_cu_paid",
		Help: "The total number of CUs paid to the provider over time.",
	}, []string{"spec"})

	// Create a new GaugeVec metric to represent the last block update time.
	lastServicedBlockTimeMetric := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "lava_provider_last_serviced_block_update_time_seconds",
		Help: "Timestamp of the last block update received from the serviced node.",
	}, []string{"spec"})

	disabledChainsMetric := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "lava_provider_disabled_chains",
		Help: "value of 1 for each disabled chain at measurement time",
	}, []string{"chainID", "apiInterface"})

	fetchLatestFailedMetric := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "lava_provider_fetch_latest_fails",
		Help: "The total number of get latest block queries that errored by chainfetcher",
	}, []string{"spec", "apiInterface"})

	fetchBlockFailedMetric := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "lava_provider_fetch_block_fails",
		Help: "The total number of get specific block queries that errored by chainfetcher",
	}, []string{"spec", "apiInterface"})

	fetchLatestSuccessMetric := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "lava_provider_fetch_latest_success",
		Help: "The total number of get latest block queries that succeeded by chainfetcher",
	}, []string{"spec", "apiInterface"})

	fetchBlockSuccessMetric := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "lava_provider_fetch_block_success",
		Help: "The total number of get specific block queries that succeeded by chainfetcher",
	}, []string{"spec", "apiInterface"})

	blockMetricLabels := []string{"spec", "apiInterface"}
	if ShowProviderEndpointInProviderMetrics {
		blockMetricLabels = append(blockMetricLabels, "provider_endpoint")
	}

	blockMetric := NewMappedLabelsGaugeVec(MappedLabelsMetricOpts{
		Name:       "lava_provider_latest_block",
		Help:       "The latest block measured",
		Labels:     blockMetricLabels,
		Registerer: prometheus.DefaultRegisterer,
	})

	loadRateMetric := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "lava_provider_load_rate",
		Help: "The load rate according to the load rate limit - Given Y simultaneous relay calls, a value of X  and will measure Y/X load rate.",
	}, []string{"spec"})

	providerLatencyMetric := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "lava_provider_latency_milliseconds",
		Help:    "Distribution of provider response latency in milliseconds.",
		Buckets: latencyBuckets,
	}, []string{"spec", "apiInterface"})

	providerEndToEndLatencyMetric := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "lava_provider_end_to_end_latency_milliseconds",
		Help:    "Distribution of full end-to-end provider processing latency in milliseconds.",
		Buckets: latencyBuckets,
	}, []string{"spec", "apiInterface"})

	virtualEpochMetric := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "lava_provider_virtual_epoch",
		Help: "The current virtual epoch measured",
	}, []string{"spec"})

	endpointsHealthChecksOkMetric := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "lava_provider_overall_health",
		Help: "At least one endpoint is healthy",
	})
	endpointsHealthChecksOkMetric.Set(1)

	endpointsHealthChecksBreakdownMetric := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "lava_provider_overall_health_breakdown",
		Help: "Health check status per chain",
	}, []string{"spec", "apiInterface"})

	frozenStatusMetric := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "lava_provider_frozen_status",
		Help: "Frozen: 1, Not Frozen: 0",
	}, []string{"chainID"})

	jailStatusMetric := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "lava_provider_jail_status",
		Help: "Jailed: 1, Not Jailed: 0",
	}, []string{"chainID"})

	jailedCountMetric := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "lava_provider_jailed_count",
		Help: "The amount of times the provider was jailed in the last 24 hours",
	}, []string{"chainID"})

	protocolVersionMetric := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "lava_provider_protocol_version",
		Help: "The current running lavap version for the process. major := version / 1000000, minor := (version / 1000) % 1000 patch := version % 1000",
	}, []string{"version"})

	// Register the metrics with the Prometheus registry.
	prometheus.MustRegister(totalCUServicedMetric)
	prometheus.MustRegister(totalCUPaidMetric)
	prometheus.MustRegister(lastServicedBlockTimeMetric)
	prometheus.MustRegister(disabledChainsMetric)
	prometheus.MustRegister(fetchLatestFailedMetric)
	prometheus.MustRegister(fetchBlockFailedMetric)
	prometheus.MustRegister(fetchLatestSuccessMetric)
	prometheus.MustRegister(fetchBlockSuccessMetric)
	prometheus.MustRegister(requestLatencyPerFunctionMetric)
	prometheus.MustRegister(virtualEpochMetric)
	prometheus.MustRegister(endpointsHealthChecksOkMetric)
	prometheus.MustRegister(endpointsHealthChecksBreakdownMetric)
	prometheus.MustRegister(protocolVersionMetric)
	prometheus.MustRegister(frozenStatusMetric)
	prometheus.MustRegister(jailStatusMetric)
	prometheus.MustRegister(jailedCountMetric)
	prometheus.MustRegister(loadRateMetric)
	prometheus.MustRegister(providerLatencyMetric)
	prometheus.MustRegister(providerEndToEndLatencyMetric)

	providerMetricsManager := &ProviderMetricsManager{
		providerMetrics:                      map[string]*ProviderMetrics{},
		totalCUServicedMetric:                totalCUServicedMetric,
		totalCUPaidMetric:                    totalCUPaidMetric,
		totalRelaysServicedMetric:            totalRelaysServicedMetric,
		totalErroredMetric:                   totalErroredMetric,
		requestLatencyPerFunctionMetric:      requestLatencyPerFunctionMetric,
		inFlightPerFunctionMetric:            totalInFlightPerFunctionMetric,
		blockMetric:                          blockMetric,
		lastServicedBlockTimeMetric:          lastServicedBlockTimeMetric,
		disabledChainsMetric:                 disabledChainsMetric,
		fetchLatestFailedMetric:              fetchLatestFailedMetric,
		fetchBlockFailedMetric:               fetchBlockFailedMetric,
		fetchLatestSuccessMetric:             fetchLatestSuccessMetric,
		fetchBlockSuccessMetric:              fetchBlockSuccessMetric,
		virtualEpochMetric:                   virtualEpochMetric,
		endpointsHealthChecksOkMetric:        endpointsHealthChecksOkMetric,
		endpointsHealthChecksOk:              1,
		endpointsHealthChecksBreakdownMetric: endpointsHealthChecksBreakdownMetric,
		protocolVersionMetric:                protocolVersionMetric,
		relaysMonitors:                       map[string]*RelaysMonitor{},
		frozenStatusMetric:                   frozenStatusMetric,
		jailStatusMetric:                     jailStatusMetric,
		jailedCountMetric:                    jailedCountMetric,
		loadRateMetric:                       loadRateMetric,
		providerLatencyMetric:                providerLatencyMetric,
		providerEndToEndLatencyMetric:        providerEndToEndLatencyMetric,
	}

	http.Handle("/metrics", promhttp.Handler())

	overallHealthHandler := func(w http.ResponseWriter, r *http.Request) {
		statusCode := http.StatusOK
		message := "Health status OK"
		if atomic.LoadUint64(&providerMetricsManager.endpointsHealthChecksOk) == 0 {
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
		utils.LavaFormatInfo("prometheus endpoint listening", utils.Attribute{Key: "Listen Address", Value: networkAddress})
		http.ListenAndServe(networkAddress, nil)
	}()

	return providerMetricsManager
}

func (pme *ProviderMetricsManager) getProviderMetric(specID, apiInterface string) *ProviderMetrics {
	pme.lock.RLock()
	defer pme.lock.RUnlock()
	return pme.providerMetrics[specID+apiInterface]
}

func (pme *ProviderMetricsManager) setProviderMetric(metrics *ProviderMetrics) {
	pme.lock.Lock()
	defer pme.lock.Unlock()
	specID := metrics.specID
	apiInterface := metrics.apiInterface
	pme.providerMetrics[specID+apiInterface] = metrics
}

func (pme *ProviderMetricsManager) AddProviderMetrics(specID, apiInterface, providerEndpoint string) *ProviderMetrics {
	if pme == nil {
		return nil
	}

	if pme.getProviderMetric(specID, apiInterface) == nil {
		providerMetric := NewProviderMetrics(specID, apiInterface, pme.totalCUServicedMetric, pme.totalCUPaidMetric, pme.totalRelaysServicedMetric, pme.totalErroredMetric, pme.inFlightPerFunctionMetric, pme.requestLatencyPerFunctionMetric, pme.loadRateMetric, pme.providerLatencyMetric, pme.providerEndToEndLatencyMetric)
		pme.setProviderMetric(providerMetric)

		endpoint := fmt.Sprintf("/metrics/%s/%s/health", specID, apiInterface)
		http.HandleFunc(endpoint, func(resp http.ResponseWriter, r *http.Request) {
			pme.relaysMonitorsLock.Lock()
			defer pme.relaysMonitorsLock.Unlock()

			relaysMonitor, ok := pme.relaysMonitors[specID+apiInterface]
			if ok && !relaysMonitor.IsHealthy() {
				resp.WriteHeader(http.StatusServiceUnavailable)
				resp.Write([]byte("Unhealthy"))
			} else {
				resp.WriteHeader(http.StatusOK)
				resp.Write([]byte("Health status OK"))
			}
		})

		utils.LavaFormatInfo("prometheus: health endpoint listening",
			utils.LogAttr("specID", specID),
			utils.LogAttr("apiInterface", apiInterface),
			utils.LogAttr("endpoint", endpoint),
		)
	}
	return pme.getProviderMetric(specID, apiInterface)
}

func (pme *ProviderMetricsManager) SetLatestBlock(specID, apiInterface, providerEndpoint string, block uint64) {
	if pme == nil {
		return
	}
	pme.lock.Lock()
	defer pme.lock.Unlock()

	labels := map[string]string{"spec": specID, "apiInterface": apiInterface}
	if ShowProviderEndpointInProviderMetrics {
		labels["provider_endpoint"] = providerEndpoint
	}
	pme.blockMetric.WithLabelValues(labels).Set(float64(block))
	pme.lastServicedBlockTimeMetric.WithLabelValues(specID).Set(float64(time.Now().Unix()))
}

func (pme *ProviderMetricsManager) AddPayment(specID string, cu uint64) {
	if pme == nil {
		return
	}
	availableAPIInterface := []string{
		spectypes.APIInterfaceJsonRPC,
		spectypes.APIInterfaceTendermintRPC,
		spectypes.APIInterfaceRest,
		spectypes.APIInterfaceGrpc,
	}
	for _, apiInterface := range availableAPIInterface {
		providerMetrics := pme.getProviderMetric(specID, apiInterface)
		if providerMetrics != nil {
			go providerMetrics.AddPayment(cu)
			break // we need to increase the metric only once
		}
	}
}

func (pme *ProviderMetricsManager) UpdateHealthCheckStatus(status bool) {
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

func (pme *ProviderMetricsManager) UpdateHealthcheckStatusBreakdown(chainId, apiInterface string, status bool) {
	if pme == nil {
		return
	}
	var value float64 = 0
	if status {
		value = 1
	}

	pme.endpointsHealthChecksBreakdownMetric.WithLabelValues(chainId, apiInterface).Set(value)
}

func (pme *ProviderMetricsManager) SetBlock(latestLavaBlock int64) {
	if pme == nil {
		return
	}

	labels := map[string]string{"spec": "lava", "apiInterface": "lava"}
	pme.blockMetric.WithLabelValues(labels).Set(float64(latestLavaBlock))
}

func (pme *ProviderMetricsManager) SetDisabledChain(specID string, apInterface string) {
	if pme == nil {
		return
	}
	pme.disabledChainsMetric.WithLabelValues(specID, apInterface).Set(1)
}

func (pme *ProviderMetricsManager) SetEnabledChain(specID string, apInterface string) {
	if pme == nil {
		return
	}
	pme.disabledChainsMetric.WithLabelValues(specID, apInterface).Set(0)
}

func (pme *ProviderMetricsManager) SetLatestBlockFetchError(specID, apiInterface string) {
	if pme == nil {
		return
	}
	pme.fetchLatestFailedMetric.WithLabelValues(specID, apiInterface).Add(1)
}

func (pme *ProviderMetricsManager) SetSpecificBlockFetchError(specID, apiInterface string) {
	if pme == nil {
		return
	}
	pme.fetchBlockFailedMetric.WithLabelValues(specID, apiInterface).Add(1)
}

func (pme *ProviderMetricsManager) SetLatestBlockFetchSuccess(specID, apiInterface string) {
	if pme == nil {
		return
	}
	pme.fetchLatestSuccessMetric.WithLabelValues(specID, apiInterface).Add(1)
}

func (pme *ProviderMetricsManager) SetSpecificBlockFetchSuccess(specID, apiInterface string) {
	if pme == nil {
		return
	}
	pme.fetchBlockSuccessMetric.WithLabelValues(specID, apiInterface).Add(1)
}

func (pme *ProviderMetricsManager) SetVirtualEpoch(virtualEpoch uint64) {
	if pme == nil {
		return
	}
	pme.virtualEpochMetric.WithLabelValues("lava").Set(float64(virtualEpoch))
}

func (pme *ProviderMetricsManager) SetVersion(version string) {
	if pme == nil {
		return
	}
	SetVersionInner(pme.protocolVersionMetric, version)
}

func (pme *ProviderMetricsManager) RegisterRelaysMonitor(chainID, apiInterface string, relaysMonitor *RelaysMonitor) {
	if pme == nil {
		return
	}

	pme.relaysMonitorsLock.Lock()
	defer pme.relaysMonitorsLock.Unlock()
	pme.relaysMonitors[chainID+apiInterface] = relaysMonitor
}

func (pme *ProviderMetricsManager) SetFrozenStatus(chain string, frozen bool) {
	if pme == nil {
		return
	}

	pme.frozenStatusMetric.WithLabelValues(chain).Set(utils.Btof(frozen))
}

func (pme *ProviderMetricsManager) SetJailStatus(chain string, jailed bool) {
	if pme == nil {
		return
	}

	pme.jailStatusMetric.WithLabelValues(chain).Set(utils.Btof(jailed))
}

func (pme *ProviderMetricsManager) SetJailedCount(chain string, jailedCount uint64) {
	if pme == nil {
		return
	}

	pme.jailedCountMetric.WithLabelValues(chain).Set(float64(jailedCount))
}

func (pme *ProviderMetricsManager) SetProviderLatency(specID, apiInterface string, latencyMs float64) {
	if pme == nil {
		return
	}

	providerMetric := pme.getProviderMetric(specID, apiInterface)
	if providerMetric != nil {
		providerMetric.SetLatency(latencyMs)
	}
}

func (pme *ProviderMetricsManager) SetProviderEndToEndLatency(specID, apiInterface string, latencyMs float64) {
	if pme == nil {
		return
	}

	providerMetric := pme.getProviderMetric(specID, apiInterface)
	if providerMetric != nil {
		providerMetric.SetEndToEndLatency(latencyMs)
	}
}
