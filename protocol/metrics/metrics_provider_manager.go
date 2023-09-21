package metrics

import (
	"net/http"
	"sync"
	"time"

	"github.com/lavanet/lava/utils"
	spectypes "github.com/lavanet/lava/x/spec/types"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	MetricsListenFlagName = "metrics-listen-address"
	DisabledFlagOption    = "disabled"
)

type ProviderMetricsManager struct {
	providerMetrics             map[string]*ProviderMetrics
	lock                        sync.RWMutex
	totalCUServicedMetric       *prometheus.CounterVec
	totalCUPaidMetric           *prometheus.CounterVec
	totalRelaysServicedMetric   *prometheus.CounterVec
	totalErroredMetric          *prometheus.CounterVec
	consumerQoSMetric           *prometheus.GaugeVec
	blockMetric                 *prometheus.GaugeVec
	lastServicedBlockTimeMetric *prometheus.GaugeVec
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

	// Register the metrics with the Prometheus registry.
	prometheus.MustRegister(totalCUServicedMetric)
	prometheus.MustRegister(totalCUPaidMetric)
	prometheus.MustRegister(totalRelaysServicedMetric)
	prometheus.MustRegister(totalErroredMetric)
	prometheus.MustRegister(consumerQoSMetric)
	prometheus.MustRegister(blockMetric)
	prometheus.MustRegister(lastServicedBlockTimeMetric)
	http.Handle("/metrics", promhttp.Handler())
	go func() {
		utils.LavaFormatInfo("prometheus endpoint listening", utils.Attribute{Key: "Listen Address", Value: networkAddress})
		http.ListenAndServe(networkAddress, nil)
	}()
	return &ProviderMetricsManager{
		providerMetrics:             map[string]*ProviderMetrics{},
		totalCUServicedMetric:       totalCUServicedMetric,
		totalCUPaidMetric:           totalCUPaidMetric,
		totalRelaysServicedMetric:   totalRelaysServicedMetric,
		totalErroredMetric:          totalErroredMetric,
		consumerQoSMetric:           consumerQoSMetric,
		blockMetric:                 blockMetric,
		lastServicedBlockTimeMetric: lastServicedBlockTimeMetric,
	}
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

func (pme *ProviderMetricsManager) SetBlock(latestLavaBlock int64) {
	if pme == nil {
		return
	}
	pme.blockMetric.WithLabelValues("lava").Set(float64(latestLavaBlock))
}
