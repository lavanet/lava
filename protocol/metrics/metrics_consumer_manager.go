package metrics

import (
	"fmt"
	"net/http"
	"sync"

	"github.com/lavanet/lava/utils"
	pairingtypes "github.com/lavanet/lava/x/pairing/types"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type ConsumerMetricsManager struct {
	totalCURequestedMetric        *prometheus.CounterVec
	totalRelaysRequestedMetric    *prometheus.CounterVec
	totalErroredMetric            *prometheus.CounterVec
	blockMetric                   *prometheus.GaugeVec
	latencyMetric                 *prometheus.GaugeVec
	qosMetric                     *prometheus.GaugeVec
	qosExcellenceMetric           *prometheus.GaugeVec
	LatestBlockMetric             *prometheus.GaugeVec
	LatestProviderRelay           *prometheus.GaugeVec
	virtualEpochMetric            *prometheus.GaugeVec
	endpointsHealthChecksOkMetric prometheus.Gauge
	lock                          sync.Mutex
	protocolVersionMetric         *prometheus.GaugeVec
	providerRelays                map[string]uint64
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
		Name: "lava_consumer_qos_metrics",
		Help: "The QOS metrics per provider for current epoch for the session with the most relays.",
	}, []string{"spec", "apiInterface", "provider_address", "qos_metric"})

	qosExcellenceMetric := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "lava_consumer_qos_excellence_metrics",
		Help: "The QOS metrics per provider excellence",
	}, []string{"spec", "provider_address", "qos_metric"})

	latestBlockMetric := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "lava_consumer_latest_provider_block",
		Help: "The latest block reported by provider",
	}, []string{"spec", "provider_address", "apiInterface"})
	latestProviderRelay := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "lava_consumer_latest_provider_relay_time",
		Help: "The latest time we sent a relay to provider",
	}, []string{"spec", "provider_address", "apiInterface"})
	virtualEpochMetric := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "virtual_epoch",
		Help: "The current virtual epoch measured",
	}, []string{"spec"})
	endpointsHealthChecksOkMetric := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "lava_consumer_endpoints_health_checks_ok",
		Help: "At least one endpoint is healthy",
	})
	endpointsHealthChecksOkMetric.Set(1)
	protocolVersionMetric := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "lava_provider_protocol_version",
		Help: "The current running lavap version for the process. major := version / 1000000, minor := (version / 1000) % 1000, patch := version % 1000",
	}, []string{"version"})
	// Register the metrics with the Prometheus registry.
	prometheus.MustRegister(totalCURequestedMetric)
	prometheus.MustRegister(totalRelaysRequestedMetric)
	prometheus.MustRegister(totalErroredMetric)
	prometheus.MustRegister(blockMetric)
	prometheus.MustRegister(latencyMetric)
	prometheus.MustRegister(qosMetric)
	prometheus.MustRegister(qosExcellenceMetric)
	prometheus.MustRegister(latestBlockMetric)
	prometheus.MustRegister(latestProviderRelay)
	prometheus.MustRegister(virtualEpochMetric)
	prometheus.MustRegister(endpointsHealthChecksOkMetric)
	prometheus.MustRegister(protocolVersionMetric)
	http.Handle("/metrics", promhttp.Handler())
	go func() {
		utils.LavaFormatInfo("prometheus endpoint listening", utils.Attribute{Key: "Listen Address", Value: networkAddress})
		http.ListenAndServe(networkAddress, nil)
	}()
	return &ConsumerMetricsManager{
		totalCURequestedMetric:        totalCURequestedMetric,
		totalRelaysRequestedMetric:    totalRelaysRequestedMetric,
		totalErroredMetric:            totalErroredMetric,
		blockMetric:                   blockMetric,
		latencyMetric:                 latencyMetric,
		qosMetric:                     qosMetric,
		qosExcellenceMetric:           qosExcellenceMetric,
		LatestBlockMetric:             latestBlockMetric,
		LatestProviderRelay:           latestProviderRelay,
		providerRelays:                map[string]uint64{},
		virtualEpochMetric:            virtualEpochMetric,
		endpointsHealthChecksOkMetric: endpointsHealthChecksOkMetric,
		protocolVersionMetric:         protocolVersionMetric,
	}
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
	pme.LatestProviderRelay.WithLabelValues(chainId, providerAddress, apiInterface).SetToCurrentTime()
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

	pme.LatestBlockMetric.WithLabelValues(chainId, providerAddress, apiInterface).Set(float64(latestBlock))
}

func (pme *ConsumerMetricsManager) SetVirtualEpoch(virtualEpoch uint64) {
	if pme == nil {
		return
	}
	pme.virtualEpochMetric.WithLabelValues("lava").Set(float64(virtualEpoch))
}

func (pme *ConsumerMetricsManager) SetEndpointsHealthChecksOkStatus(status bool) {
	var value float64 = 0
	if status {
		value = 1
	}
	pme.endpointsHealthChecksOkMetric.Set(value)
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
