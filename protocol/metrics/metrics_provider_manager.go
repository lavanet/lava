package metrics

import (
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/lavanet/lava/v2/utils"
	spectypes "github.com/lavanet/lava/v2/x/spec/types"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	MetricsListenFlagName    = "metrics-listen-address"
	AddApiMethodCallsMetrics = "add-api-method-metrics"
	RelayServerFlagName      = "relay-server-address"
	DisabledFlagOption       = "disabled"
)

type ProviderMetricsManager struct {
	providerMetrics               map[string]*ProviderMetrics
	lock                          sync.RWMutex
	totalCUServicedMetric         *prometheus.CounterVec
	totalCUPaidMetric             *prometheus.CounterVec
	totalRelaysServicedMetric     *prometheus.CounterVec
	totalErroredMetric            *prometheus.CounterVec
	consumerQoSMetric             *prometheus.GaugeVec
	blockMetric                   *prometheus.GaugeVec
	lastServicedBlockTimeMetric   *prometheus.GaugeVec
	disabledChainsMetric          *prometheus.GaugeVec
	fetchLatestFailedMetric       *prometheus.CounterVec
	fetchBlockFailedMetric        *prometheus.CounterVec
	fetchLatestSuccessMetric      *prometheus.CounterVec
	fetchBlockSuccessMetric       *prometheus.CounterVec
	protocolVersionMetric         *prometheus.GaugeVec
	virtualEpochMetric            *prometheus.GaugeVec
	endpointsHealthChecksOkMetric prometheus.Gauge
	endpointsHealthChecksOk       uint64
	relaysMonitors                map[string]*RelaysMonitor
	relaysMonitorsLock            sync.RWMutex
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

	// Create a new GaugeVec metric to represent the TotalRelaysServiced over time.
	totalRelaysServicedMetric := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "lava_provider_total_relays_serviced",
		Help: "The total number of relays serviced by the provider over time.",
	}, []string{"spec", "apiInterface"})

	// Create a new GaugeVec metric to represent the TotalErrored over time.
	totalErroredMetric := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "lava_provider_total_errored",
		Help: "The total number of errors encountered by the provider over time.",
	}, []string{"spec", "apiInterface"})

	consumerQoSMetric := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "lava_consumer_QoS",
		Help: "The latest QoS score from a consumer",
	}, []string{"spec", "consumer_address", "qos_metric"})

	blockMetric := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "lava_latest_block",
		Help: "The latest block measured",
	}, []string{"spec"})

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
	}, []string{"spec"})

	fetchBlockFailedMetric := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "lava_provider_fetch_block_fails",
		Help: "The total number of get specific block queries that errored by chainfetcher",
	}, []string{"spec"})

	fetchLatestSuccessMetric := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "lava_provider_fetch_latest_success",
		Help: "The total number of get latest block queries that succeeded by chainfetcher",
	}, []string{"spec"})

	fetchBlockSuccessMetric := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "lava_provider_fetch_block_success",
		Help: "The total number of get specific block queries that succeeded by chainfetcher",
	}, []string{"spec"})
	virtualEpochMetric := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "virtual_epoch",
		Help: "The current virtual epoch measured",
	}, []string{"spec"})
	endpointsHealthChecksOkMetric := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "lava_provider_overall_health",
		Help: "At least one endpoint is healthy",
	})
	endpointsHealthChecksOkMetric.Set(1)

	protocolVersionMetric := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "lava_provider_protocol_version",
		Help: "The current running lavap version for the process. major := version / 1000000, minor := (version / 1000) % 1000 patch := version % 1000",
	}, []string{"version"})
	// Register the metrics with the Prometheus registry.
	prometheus.MustRegister(totalCUServicedMetric)
	prometheus.MustRegister(totalCUPaidMetric)
	prometheus.MustRegister(totalRelaysServicedMetric)
	prometheus.MustRegister(totalErroredMetric)
	prometheus.MustRegister(consumerQoSMetric)
	prometheus.MustRegister(blockMetric)
	prometheus.MustRegister(lastServicedBlockTimeMetric)
	prometheus.MustRegister(disabledChainsMetric)
	prometheus.MustRegister(fetchLatestFailedMetric)
	prometheus.MustRegister(fetchBlockFailedMetric)
	prometheus.MustRegister(fetchLatestSuccessMetric)
	prometheus.MustRegister(fetchBlockSuccessMetric)
	prometheus.MustRegister(virtualEpochMetric)
	prometheus.MustRegister(endpointsHealthChecksOkMetric)
	prometheus.MustRegister(protocolVersionMetric)

	providerMetricsManager := &ProviderMetricsManager{
		providerMetrics:               map[string]*ProviderMetrics{},
		totalCUServicedMetric:         totalCUServicedMetric,
		totalCUPaidMetric:             totalCUPaidMetric,
		totalRelaysServicedMetric:     totalRelaysServicedMetric,
		totalErroredMetric:            totalErroredMetric,
		consumerQoSMetric:             consumerQoSMetric,
		blockMetric:                   blockMetric,
		lastServicedBlockTimeMetric:   lastServicedBlockTimeMetric,
		disabledChainsMetric:          disabledChainsMetric,
		fetchLatestFailedMetric:       fetchLatestFailedMetric,
		fetchBlockFailedMetric:        fetchBlockFailedMetric,
		fetchLatestSuccessMetric:      fetchLatestSuccessMetric,
		fetchBlockSuccessMetric:       fetchBlockSuccessMetric,
		virtualEpochMetric:            virtualEpochMetric,
		endpointsHealthChecksOkMetric: endpointsHealthChecksOkMetric,
		endpointsHealthChecksOk:       1,
		protocolVersionMetric:         protocolVersionMetric,
		relaysMonitors:                map[string]*RelaysMonitor{},
	}

	http.Handle("/metrics", promhttp.Handler())

	overallHealthHandler := func(w http.ResponseWriter, r *http.Request) {
		statusCode := http.StatusOK
		message := "Healthy"
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

func (pme *ProviderMetricsManager) AddProviderMetrics(specID, apiInterface string) *ProviderMetrics {
	if pme == nil {
		return nil
	}

	if pme.getProviderMetric(specID, apiInterface) == nil {
		providerMetric := NewProviderMetrics(specID, apiInterface, pme.totalCUServicedMetric, pme.totalCUPaidMetric, pme.totalRelaysServicedMetric, pme.totalErroredMetric, pme.consumerQoSMetric)
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
				resp.Write([]byte("Healthy"))
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

func (pme *ProviderMetricsManager) SetLatestBlock(specID string, block uint64) {
	if pme == nil {
		return
	}
	pme.lock.Lock()
	defer pme.lock.Unlock()
	pme.blockMetric.WithLabelValues(specID).Set(float64(block))
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

func (pme *ProviderMetricsManager) SetBlock(latestLavaBlock int64) {
	if pme == nil {
		return
	}
	pme.blockMetric.WithLabelValues("lava").Set(float64(latestLavaBlock))
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

func (pme *ProviderMetricsManager) SetLatestBlockFetchError(specID string) {
	if pme == nil {
		return
	}
	pme.fetchLatestFailedMetric.WithLabelValues(specID).Add(1)
}

func (pme *ProviderMetricsManager) SetSpecificBlockFetchError(specID string) {
	if pme == nil {
		return
	}
	pme.fetchBlockFailedMetric.WithLabelValues(specID).Add(1)
}

func (pme *ProviderMetricsManager) SetLatestBlockFetchSuccess(specID string) {
	if pme == nil {
		return
	}
	pme.fetchLatestSuccessMetric.WithLabelValues(specID).Add(1)
}

func (pme *ProviderMetricsManager) SetSpecificBlockFetchSuccess(specID string) {
	if pme == nil {
		return
	}
	pme.fetchBlockSuccessMetric.WithLabelValues(specID).Add(1)
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
