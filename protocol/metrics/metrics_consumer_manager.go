package metrics

import (
	"net/http"

	"github.com/lavanet/lava/utils"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type ConsumerMetricsManager struct {
	totalCURequestedMetric     *prometheus.CounterVec
	totalRelaysRequestedMetric *prometheus.CounterVec
	totalErroredMetric         *prometheus.CounterVec
	blockMetric                *prometheus.GaugeVec
	latencyMetric              *prometheus.GaugeVec
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

	// Register the metrics with the Prometheus registry.
	prometheus.MustRegister(totalCURequestedMetric)
	prometheus.MustRegister(totalRelaysRequestedMetric)
	prometheus.MustRegister(totalErroredMetric)
	prometheus.MustRegister(blockMetric)
	prometheus.MustRegister(latencyMetric)
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
