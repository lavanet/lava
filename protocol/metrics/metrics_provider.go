package metrics

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
)

type ProviderMetrics struct {
	SpecID                    string
	ApiInterface              string
	TotalCUServiced           uint64
	TotalCUPaid               uint64
	TotalRelaysServiced       uint64
	TotalErrored              uint64
	CurrentConsumers          uint64
	lock                      sync.Mutex
	totalCUServicedMetric     *prometheus.GaugeVec
	totalCUPaidMetric         *prometheus.GaugeVec
	totalRelaysServicedMetric *prometheus.GaugeVec
	totalErroredMetric        *prometheus.GaugeVec
	currentConsumersMetric    *prometheus.GaugeVec
}

func (pm *ProviderMetrics) addRelay(cu uint64) {
	pm.lock.Lock()
	defer pm.lock.Unlock()
	pm.TotalCUServiced += cu
	pm.TotalRelaysServiced += 1
}

func (pm *ProviderMetrics) addPayment(cu uint64) {
	pm.lock.Lock()
	defer pm.lock.Unlock()
	pm.TotalCUPaid += cu
}

func (pm *ProviderMetrics) sendMetrics() {
	pm.lock.Lock()
	defer pm.lock.Unlock()
	pm.totalCUServicedMetric.WithLabelValues(pm.SpecID, pm.ApiInterface).Set(float64(pm.TotalCUServiced))
	pm.totalCUPaidMetric.WithLabelValues(pm.SpecID, pm.ApiInterface).Set(float64(pm.TotalCUPaid))
	pm.totalRelaysServicedMetric.WithLabelValues(pm.SpecID, pm.ApiInterface).Set(float64(pm.TotalRelaysServiced))
	pm.totalErroredMetric.WithLabelValues(pm.SpecID, pm.ApiInterface).Set(float64(pm.TotalErrored))
	pm.currentConsumersMetric.WithLabelValues(pm.SpecID, pm.ApiInterface).Set(float64(pm.CurrentConsumers))
}
