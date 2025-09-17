package metrics

import (
	"sync"

	pairingtypes "github.com/lavanet/lava/v5/x/pairing/types"
	"github.com/prometheus/client_golang/prometheus"
)

const (
	AvailabilityLabel = "availability"
	SyncLabel         = "sync/freshness"
	LatencyLabel      = "latency"
)

type ProviderMetrics struct {
	specID                    string
	apiInterface              string
	endpoint                  string
	lock                      sync.Mutex
	totalCUServicedMetric     *prometheus.CounterVec
	totalCUPaidMetric         *prometheus.CounterVec
	totalRelaysServicedMetric *MappedLabelsCounterVec
	totalErroredMetric        *prometheus.CounterVec
	consumerQoSMetric         *prometheus.GaugeVec
	loadRateMetric            *prometheus.GaugeVec
	providerLatencyMetric     *prometheus.GaugeVec
}

func (pm *ProviderMetrics) AddRelay(consumerAddress string, cu uint64, qos *pairingtypes.QualityOfServiceReport) {
	if pm == nil {
		return
	}
	pm.lock.Lock()
	defer pm.lock.Unlock()
	pm.totalCUServicedMetric.WithLabelValues(pm.specID, pm.apiInterface).Add(float64(cu))
	labels := map[string]string{"spec": pm.specID, "apiInterface": pm.apiInterface, "provider_endpoint": pm.endpoint}
	pm.totalRelaysServicedMetric.WithLabelValues(labels).Add(1)
	if qos == nil {
		return
	}
	availability, err := qos.Availability.Float64()
	if err == nil {
		pm.consumerQoSMetric.WithLabelValues(pm.specID, consumerAddress, AvailabilityLabel).Set(availability)
	}
	sync, err := qos.Sync.Float64()
	if err == nil {
		pm.consumerQoSMetric.WithLabelValues(pm.specID, consumerAddress, SyncLabel).Set(sync)
	}
	latency, err := qos.Latency.Float64()
	if err == nil {
		pm.consumerQoSMetric.WithLabelValues(pm.specID, consumerAddress, LatencyLabel).Set(latency)
	}
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
	pm.lock.Lock()
	defer pm.lock.Unlock()
	pm.providerLatencyMetric.WithLabelValues(pm.specID, pm.apiInterface).Set(latencyMs)
}

func (pm *ProviderMetrics) AddPayment(cu uint64) {
	if pm == nil {
		return
	}
	pm.lock.Lock()
	defer pm.lock.Unlock()
	pm.totalCUPaidMetric.WithLabelValues(pm.specID).Add(float64(cu))
}

func (pm *ProviderMetrics) AddError() {
	if pm == nil {
		return
	}
	pm.lock.Lock()
	defer pm.lock.Unlock()
	pm.totalErroredMetric.WithLabelValues(pm.specID, pm.apiInterface).Add(1)
}

func NewProviderMetrics(specID, apiInterface, endpoint string, totalCUServicedMetric *prometheus.CounterVec,
	totalCUPaidMetric *prometheus.CounterVec,
	totalRelaysServicedMetric *MappedLabelsCounterVec,
	totalErroredMetric *prometheus.CounterVec,
	consumerQoSMetric *prometheus.GaugeVec,
	loadRateMetric *prometheus.GaugeVec,
	providerLatencyMetric *prometheus.GaugeVec,
) *ProviderMetrics {
	pm := &ProviderMetrics{
		specID:                    specID,
		apiInterface:              apiInterface,
		endpoint:                  endpoint,
		lock:                      sync.Mutex{},
		totalCUServicedMetric:     totalCUServicedMetric,
		totalCUPaidMetric:         totalCUPaidMetric,
		totalRelaysServicedMetric: totalRelaysServicedMetric,
		totalErroredMetric:        totalErroredMetric,
		consumerQoSMetric:         consumerQoSMetric,
		loadRateMetric:            loadRateMetric,
		providerLatencyMetric:     providerLatencyMetric,
	}
	return pm
}
