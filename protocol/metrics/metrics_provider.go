package metrics

import (
	"sync"

	pairingtypes "github.com/lavanet/lava/x/pairing/types"
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
	totalCUServiced           uint64
	totalCUPaid               uint64
	totalRelaysServiced       uint64
	totalErrored              uint64
	currentConsumers          uint64
	lock                      sync.Mutex
	totalCUServicedMetric     *prometheus.GaugeVec
	totalCUPaidMetric         *prometheus.GaugeVec
	totalRelaysServicedMetric *prometheus.GaugeVec
	totalErroredMetric        *prometheus.GaugeVec
	consumerQoSMetric         *prometheus.GaugeVec
}

func (pm *ProviderMetrics) AddRelay(consumerAddress string, cu uint64, qos *pairingtypes.QualityOfServiceReport) {
	if pm == nil {
		return
	}
	pm.lock.Lock()
	defer pm.lock.Unlock()
	pm.totalCUServiced += cu
	pm.totalRelaysServiced += 1
	pm.totalCUServicedMetric.WithLabelValues(pm.specID, pm.apiInterface).Set(float64(pm.totalCUServiced))
	pm.totalRelaysServicedMetric.WithLabelValues(pm.specID, pm.apiInterface).Set(float64(pm.totalRelaysServiced))
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
	pm.totalCUPaid += cu
	pm.totalCUPaidMetric.WithLabelValues(pm.specID).Set(float64(pm.totalCUPaid))
}

func (pm *ProviderMetrics) AddError() {
	if pm == nil {
		return
	}
	pm.lock.Lock()
	defer pm.lock.Unlock()
	pm.totalErrored += 1
	pm.totalErroredMetric.WithLabelValues(pm.specID, pm.apiInterface).Set(float64(pm.totalErrored))
}

func NewProviderMetrics(specID string, apiInterface string, totalCUServicedMetric *prometheus.GaugeVec,
	totalCUPaidMetric *prometheus.GaugeVec,
	totalRelaysServicedMetric *prometheus.GaugeVec,
	totalErroredMetric *prometheus.GaugeVec,
	consumerQoSMetric *prometheus.GaugeVec,
) *ProviderMetrics {
	pm := &ProviderMetrics{
		specID:                    specID,
		apiInterface:              apiInterface,
		totalCUServiced:           0,
		totalCUPaid:               0,
		totalRelaysServiced:       0,
		totalErrored:              0,
		currentConsumers:          0,
		lock:                      sync.Mutex{},
		totalCUServicedMetric:     totalCUServicedMetric,
		totalCUPaidMetric:         totalCUPaidMetric,
		totalRelaysServicedMetric: totalRelaysServicedMetric,
		totalErroredMetric:        totalErroredMetric,
		consumerQoSMetric:         consumerQoSMetric,
	}
	return pm
}
