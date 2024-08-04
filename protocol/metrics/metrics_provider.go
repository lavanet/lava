package metrics

import (
	"sync"

	pairingtypes "github.com/lavanet/lava/v2/x/pairing/types"
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
	lock                      sync.Mutex
	totalCUServicedMetric     *prometheus.CounterVec
	totalCUPaidMetric         *prometheus.CounterVec
	totalRelaysServicedMetric *prometheus.CounterVec
	totalErroredMetric        *prometheus.CounterVec
	consumerQoSMetric         *prometheus.GaugeVec
}

func (pm *ProviderMetrics) AddRelay(consumerAddress string, cu uint64, qos *pairingtypes.QualityOfServiceReport) {
	if pm == nil {
		return
	}
	pm.lock.Lock()
	defer pm.lock.Unlock()
	pm.totalCUServicedMetric.WithLabelValues(pm.specID, pm.apiInterface).Add(float64(cu))
	pm.totalRelaysServicedMetric.WithLabelValues(pm.specID, pm.apiInterface).Add(1)
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

func NewProviderMetrics(specID, apiInterface string, totalCUServicedMetric *prometheus.CounterVec,
	totalCUPaidMetric *prometheus.CounterVec,
	totalRelaysServicedMetric *prometheus.CounterVec,
	totalErroredMetric *prometheus.CounterVec,
	consumerQoSMetric *prometheus.GaugeVec,
) *ProviderMetrics {
	pm := &ProviderMetrics{
		specID:                    specID,
		apiInterface:              apiInterface,
		lock:                      sync.Mutex{},
		totalCUServicedMetric:     totalCUServicedMetric,
		totalCUPaidMetric:         totalCUPaidMetric,
		totalRelaysServicedMetric: totalRelaysServicedMetric,
		totalErroredMetric:        totalErroredMetric,
		consumerQoSMetric:         consumerQoSMetric,
	}
	return pm
}
