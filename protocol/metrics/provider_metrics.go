package metrics

import (
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

const (
	// Selection stats labels
	SelectionAvailabilityLabel = "availability"
	SelectionLatencyLabel      = "latency"
	SelectionSyncLabel         = "sync"
	SelectionStakeLabel        = "stake"
	SelectionCompositeLabel    = "composite"
)

type ProviderMetrics struct {
	specID                          string
	apiInterface                    string
	lock                            sync.Mutex
	totalCUServicedMetric           *prometheus.CounterVec
	totalCUPaidMetric               *prometheus.CounterVec
	totalRelaysServicedMetric       *MappedLabelsCounterVec
	totalErroredMetric              *MappedLabelsCounterVec
	inFlightPerFunctionMetric       *MappedLabelsGaugeVec
	requestLatencyPerFunctionMetric *prometheus.HistogramVec
	loadRateMetric                  *prometheus.GaugeVec
	providerLatencyMetric           *prometheus.HistogramVec
	providerEndToEndLatencyMetric   *prometheus.HistogramVec
}

func (pm *ProviderMetrics) AddRelay(cu uint64, function string) {
	if pm == nil {
		return
	}
	pm.lock.Lock()
	defer pm.lock.Unlock()
	pm.totalCUServicedMetric.WithLabelValues(pm.specID, pm.apiInterface).Add(float64(cu))
	labels := map[string]string{"spec": pm.specID, "apiInterface": pm.apiInterface, "function": function}
	pm.totalRelaysServicedMetric.WithLabelValues(labels).Add(1)
}

func (pm *ProviderMetrics) AddInFlightRelay(function string) {
	if pm == nil {
		return
	}
	pm.lock.Lock()
	defer pm.lock.Unlock()
	labels := map[string]string{"spec": pm.specID, "apiInterface": pm.apiInterface, "function": function}
	pm.inFlightPerFunctionMetric.WithLabelValues(labels).Add(1)
}

func (pm *ProviderMetrics) SubInFlightRelay(function string) {
	if pm == nil {
		return
	}
	pm.lock.Lock()
	defer pm.lock.Unlock()
	labels := map[string]string{"spec": pm.specID, "apiInterface": pm.apiInterface, "function": function}
	pm.inFlightPerFunctionMetric.WithLabelValues(labels).Sub(1)
}

func (pm *ProviderMetrics) SetLoadRate(loadRate float64) {
	if pm == nil {
		return
	}
	pm.loadRateMetric.WithLabelValues(pm.specID).Set(loadRate)
}

func (pm *ProviderMetrics) SetLatency(latencyMs float64) {
	if pm == nil {
		return
	}
	pm.providerLatencyMetric.WithLabelValues(pm.specID, pm.apiInterface).Observe(latencyMs)
}

func (pm *ProviderMetrics) SetEndToEndLatency(latencyMs float64) {
	if pm == nil {
		return
	}
	pm.providerEndToEndLatencyMetric.WithLabelValues(pm.specID, pm.apiInterface).Observe(latencyMs)
}

func (pm *ProviderMetrics) AddPayment(cu uint64) {
	if pm == nil {
		return
	}
	pm.lock.Lock()
	defer pm.lock.Unlock()
	pm.totalCUPaidMetric.WithLabelValues(pm.specID).Add(float64(cu))
}

func (pm *ProviderMetrics) AddFunctionLatency(function string, latency time.Duration) {
	if pm == nil {
		return
	}
	pm.requestLatencyPerFunctionMetric.WithLabelValues(pm.specID, pm.apiInterface, function).Observe(float64(latency.Milliseconds()))
}

func (pm *ProviderMetrics) AddFunctionError(function string) {
	if pm == nil {
		return
	}
	pm.lock.Lock()
	defer pm.lock.Unlock()
	labels := map[string]string{"spec": pm.specID, "apiInterface": pm.apiInterface, "function": function}
	pm.totalRelaysServicedMetric.WithLabelValues(labels).Add(1)
	pm.totalErroredMetric.WithLabelValues(labels).Add(1)
}

func NewProviderMetrics(specID, apiInterface string,
	totalCUServicedMetric *prometheus.CounterVec,
	totalCUPaidMetric *prometheus.CounterVec,
	totalRelaysServicedMetric *MappedLabelsCounterVec,
	totalErroredMetric *MappedLabelsCounterVec,
	inFlightPerFunctionMetric *MappedLabelsGaugeVec,
	requestLatencyPerFunctionMetric *prometheus.HistogramVec,
	loadRateMetric *prometheus.GaugeVec,
	providerLatencyMetric *prometheus.HistogramVec,
	providerEndToEndLatencyMetric *prometheus.HistogramVec,
) *ProviderMetrics {
	return &ProviderMetrics{
		specID:                          specID,
		apiInterface:                    apiInterface,
		lock:                            sync.Mutex{},
		totalCUServicedMetric:           totalCUServicedMetric,
		totalCUPaidMetric:               totalCUPaidMetric,
		totalRelaysServicedMetric:       totalRelaysServicedMetric,
		totalErroredMetric:              totalErroredMetric,
		inFlightPerFunctionMetric:       inFlightPerFunctionMetric,
		requestLatencyPerFunctionMetric: requestLatencyPerFunctionMetric,
		loadRateMetric:                  loadRateMetric,
		providerLatencyMetric:           providerLatencyMetric,
		providerEndToEndLatencyMetric:   providerEndToEndLatencyMetric,
	}
}
