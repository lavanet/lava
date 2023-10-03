package metrics

import (
	"net/http"
	"sync"

	"github.com/lavanet/lava/utils"
	pairingtypes "github.com/lavanet/lava/x/pairing/types"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type ConsumerMetricsManager struct {
	totalCURequestedMetric     *prometheus.CounterVec
	totalRelaysRequestedMetric *prometheus.CounterVec
	totalErroredMetric         *prometheus.CounterVec
	blockMetric                *prometheus.GaugeVec
	latencyMetric              *prometheus.GaugeVec
	qosMetric                  *prometheus.GaugeVec
	qosExcellenceMetric        *prometheus.GaugeVec
	LatestBlockMetric          *prometheus.GaugeVec
	lock                       sync.Mutex
	providerRelays             map[string]uint64
}

func NewConsumerMetricsManager(networkAddress string) *ConsumerMetricsManager {
	if networkAddress == DisabledFlagOption {
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

	// Create a new GaugeVec metric to represent the TotalErrored over time.
	totalErroredMetric := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "lava_consumer_total_errored",
		Help: "The total number of errors encountered by the consumer over time.",
	}, []string{"spec", "apiInterface"})

	blockMetric := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "lava_latest_block",
		Help: "The latest block measured",
	}, []string{"spec"})

	latencyMetric := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "lava_consumer_latency_for_request",
		Help: "The latency of requests requested by the consumer over time.",
	}, []string{"spec", "apiInterface"})

	qosMetric := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "lava_qos_metrics",
		Help: "The QOS metrics per provider for current epoch for the session with the most relays.",
	}, []string{"spec", "apiInterface", "provider_address", "qos_metric"})

	qosExcellenceMetric := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "lava_qos_excellence_metrics",
		Help: "The QOS metrics per provider excellence",
	}, []string{"spec", "provider_address", "qos_metric"})

	latestBlockMetric := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "lava_latest_provider_block",
		Help: "The latest block reported by provider",
	}, []string{"spec", "provider_address", "apiInterface"})
	// Register the metrics with the Prometheus registry.
	prometheus.MustRegister(totalCURequestedMetric)
	prometheus.MustRegister(totalRelaysRequestedMetric)
	prometheus.MustRegister(totalErroredMetric)
	prometheus.MustRegister(blockMetric)
	prometheus.MustRegister(latencyMetric)
	prometheus.MustRegister(qosMetric)
	prometheus.MustRegister(qosExcellenceMetric)
	prometheus.MustRegister(latestBlockMetric)
	http.Handle("/metrics", promhttp.Handler())
	go func() {
		utils.LavaFormatInfo("prometheus endpoint listening", utils.Attribute{Key: "Listen Address", Value: networkAddress})
		http.ListenAndServe(networkAddress, nil)
	}()
	return &ConsumerMetricsManager{
		totalCURequestedMetric:     totalCURequestedMetric,
		totalRelaysRequestedMetric: totalRelaysRequestedMetric,
		totalErroredMetric:         totalErroredMetric,
		blockMetric:                blockMetric,
		latencyMetric:              latencyMetric,
		qosMetric:                  qosMetric,
		qosExcellenceMetric:        qosExcellenceMetric,
		LatestBlockMetric:          latestBlockMetric,
		providerRelays:             map[string]uint64{},
	}
}

func (pme *ConsumerMetricsManager) SetBlock(block int64) {
	if pme == nil {
		return
	}
	pme.blockMetric.WithLabelValues("lava").Set(float64(block))
}

func (pme *ConsumerMetricsManager) SetRelayMetrics(relayMetric *RelayMetrics) {
	if pme == nil {
		return
	}
	pme.latencyMetric.WithLabelValues(relayMetric.ChainID, relayMetric.APIType).Set(float64(relayMetric.Latency))
	pme.totalCURequestedMetric.WithLabelValues(relayMetric.ChainID, relayMetric.APIType).Add(float64(relayMetric.ComputeUnits))
	pme.totalRelaysRequestedMetric.WithLabelValues(relayMetric.ChainID, relayMetric.APIType).Add(1)
	if !relayMetric.Success {
		pme.totalErroredMetric.WithLabelValues(relayMetric.ChainID, relayMetric.APIType).Add(1)
	}
}

func (pme *ConsumerMetricsManager) SetQOSMetrics(chainId string, apiInterface string, providerAddress string, qos *pairingtypes.QualityOfServiceReport, qosExcellence *pairingtypes.QualityOfServiceReport, latestBlock int64, relays uint64) {
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
	// update existing relays
	pme.providerRelays[providerRelaysKey] = relays
	setMetricsForQos := func(qosArg *pairingtypes.QualityOfServiceReport, metric *prometheus.GaugeVec, apiInterfaceArg string) {
		if qosArg == nil {
			return
		}
		availability, err := qosArg.Availability.Float64()
		if err == nil {
			if apiInterfaceArg == "" {
				metric.WithLabelValues(chainId, providerAddress, AvailabilityLabel).Set(availability)
			} else {
				metric.WithLabelValues(chainId, apiInterface, providerAddress, AvailabilityLabel).Set(availability)
			}
		}
		sync, err := qosArg.Sync.Float64()
		if err == nil {
			if apiInterfaceArg == "" {
				metric.WithLabelValues(chainId, providerAddress, SyncLabel).Set(sync)
			} else {
				metric.WithLabelValues(chainId, apiInterface, providerAddress, SyncLabel).Set(sync)
			}
		}
		latency, err := qosArg.Latency.Float64()
		if err == nil {
			if apiInterfaceArg == "" {
				metric.WithLabelValues(chainId, providerAddress, LatencyLabel).Set(latency)
			} else {
				metric.WithLabelValues(chainId, apiInterface, providerAddress, LatencyLabel).Set(latency)
			}
		}
	}
	setMetricsForQos(qos, pme.qosMetric, apiInterface)
	setMetricsForQos(qosExcellence, pme.qosExcellenceMetric, "") // it's one for all of them

	pme.LatestBlockMetric.WithLabelValues(chainId, providerAddress, apiInterface)
}

func (pme *ConsumerMetricsManager) ResetQOSMetrics() {
	if pme == nil {
		return
	}
	pme.lock.Lock()
	defer pme.lock.Unlock()
	pme.qosMetric.Reset()
	pme.qosExcellenceMetric.Reset()
	pme.providerRelays = map[string]uint64{}
}
