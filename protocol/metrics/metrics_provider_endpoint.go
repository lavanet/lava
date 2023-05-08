package metrics

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
)

type ProviderMetricsEndpoint struct {
	providerMetrics map[string]*ProviderMetrics
	lock            sync.RWMutex
}

func NewProviderMetricsEndpoint(networkAddress string) *ProviderMetricsEndpoint {
	//TODO add the listening endpoint and an update timer
	return &ProviderMetricsEndpoint{providerMetrics: map[string]*ProviderMetrics{}}
}
func (pme *ProviderMetricsEndpoint) getProviderMetric(specID string, apiInterface string) *ProviderMetrics {
	return pme.providerMetrics[specID+apiInterface]
}

func (pme *ProviderMetricsEndpoint) setProviderMetric(metrics *ProviderMetrics) {
	specID := metrics.SpecID
	apiInterface := metrics.ApiInterface
	pme.providerMetrics[specID+apiInterface] = metrics
}

func (pme *ProviderMetricsEndpoint) AddProviderMetrics(specID string, apiInterface string) *ProviderMetrics {
	pme.lock.Lock()
	defer pme.lock.Unlock()
	if pme.getProviderMetric(specID, apiInterface) == nil {
		totalCUServicedMetric := prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "provider_total_cu_serviced",
			Help: "The total number of CUs serviced by the provider over time.",
		}, []string{specID, apiInterface})

		// Create a new GaugeVec metric to represent the TotalCUPaid over time.
		totalCUPaidMetric := prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "provider_total_cu_paid",
			Help: "The total number of CUs paid to the provider over time.",
		}, []string{specID, apiInterface})

		// Create a new GaugeVec metric to represent the TotalRelaysServiced over time.
		totalRelaysServicedMetric := prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "provider_total_relays_serviced",
			Help: "The total number of relays serviced by the provider over time.",
		}, []string{specID, apiInterface})

		// Create a new GaugeVec metric to represent the TotalErrored over time.
		totalErroredMetric := prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "provider_total_errored",
			Help: "The total number of errors encountered by the provider over time.",
		}, []string{specID, apiInterface})

		// Create a new GaugeVec metric to represent the CurrentConsumers over time.
		currentConsumersMetric := prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "provider_current_consumers",
			Help: "The current number of consumers of the provider.",
		}, []string{specID, apiInterface})
		// Register the metrics with the Prometheus registry.
		prometheus.MustRegister(totalCUServicedMetric)
		prometheus.MustRegister(totalCUPaidMetric)
		prometheus.MustRegister(totalRelaysServicedMetric)
		prometheus.MustRegister(totalErroredMetric)
		prometheus.MustRegister(currentConsumersMetric)
		pme.setProviderMetric(&ProviderMetrics{SpecID: specID, ApiInterface: apiInterface})
	}
	return pme.getProviderMetric(specID, apiInterface)
}
