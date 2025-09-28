package metrics

import (
	"sync"
	"time"

	pairingtypes "github.com/lavanet/lava/v5/x/pairing/types"
	"github.com/prometheus/client_golang/prometheus"
)

const (
	AvailabilityLabel = "availability"
	SyncLabel         = "sync/freshness"
	LatencyLabel      = "latency"
)

type ProviderMetrics struct {
	specID                          string
	apiInterface                    string
	endpoint                        string
	lock                            sync.Mutex
	totalCUServicedMetric           *prometheus.CounterVec
	totalCUPaidMetric               *prometheus.CounterVec
	totalRelaysServicedMetric       *MappedLabelsCounterVec
	totalRequestsPerFunctionMetric  *MappedLabelsCounterVec
	inFlightPerFunctionMetric       *MappedLabelsGaugeVec
	totalErrorsPerFunctionMetric    *MappedLabelsCounterVec
	requestLatencyPerFunctionMetric *MappedLabelsGaugeVec
	totalErroredMetric              *prometheus.CounterVec
	consumerQoSMetric               *prometheus.GaugeVec
	loadRateMetric                  *prometheus.GaugeVec
}

func (pm *ProviderMetrics) AddRelay(consumerAddress string, cu uint64, qos *pairingtypes.QualityOfServiceReport, function string) {
	if pm == nil {
		return
	}
	pm.lock.Lock()
	defer pm.lock.Unlock()
	pm.totalCUServicedMetric.WithLabelValues(pm.specID, pm.apiInterface).Add(float64(cu))
	totalRelayedLabels := map[string]string{"spec": pm.specID, "apiInterface": pm.apiInterface, "provider_endpoint": pm.endpoint}
	totalReqPerFuncLabels := map[string]string{"spec": pm.specID, "apiInterface": pm.apiInterface, "function": function}
	pm.totalRelaysServicedMetric.WithLabelValues(totalRelayedLabels).Add(1)
	pm.totalRequestsPerFunctionMetric.WithLabelValues(totalReqPerFuncLabels).Add(1)
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
	pm.lock.Lock()
	defer pm.lock.Unlock()
	labels := map[string]string{"spec": pm.specID, "apiInterface": pm.apiInterface, "function": function}
	pm.requestLatencyPerFunctionMetric.WithLabelValues(labels).Set(float64(latency.Milliseconds()))
}

func (pm *ProviderMetrics) AddFunctionError(function string) {
	if pm == nil {
		return
	}
	pm.lock.Lock()
	defer pm.lock.Unlock()
	pm.totalErroredMetric.WithLabelValues(pm.specID, pm.apiInterface).Add(1)
	labels := map[string]string{"spec": pm.specID, "apiInterface": pm.apiInterface, "function": function}
	pm.totalRequestsPerFunctionMetric.WithLabelValues(labels).Add(1)
	pm.totalErrorsPerFunctionMetric.WithLabelValues(labels).Add(1)
}

func NewProviderMetrics(specID, apiInterface, endpoint string, totalCUServicedMetric *prometheus.CounterVec,
	totalCUPaidMetric *prometheus.CounterVec,
	totalRelaysServicedMetric *MappedLabelsCounterVec,
	totalRequestsPerFunctionMetric *MappedLabelsCounterVec,
	inFlightPerFunctionMetric *MappedLabelsGaugeVec,
	totalErrorsPerFunctionMetric *MappedLabelsCounterVec,
	requestLatencyPerFunctionMetric *MappedLabelsGaugeVec,
	totalErroredMetric *prometheus.CounterVec,
	consumerQoSMetric *prometheus.GaugeVec,
	loadRateMetric *prometheus.GaugeVec,
) *ProviderMetrics {
	pm := &ProviderMetrics{
		specID:                          specID,
		apiInterface:                    apiInterface,
		endpoint:                        endpoint,
		lock:                            sync.Mutex{},
		totalCUServicedMetric:           totalCUServicedMetric,
		totalCUPaidMetric:               totalCUPaidMetric,
		totalRelaysServicedMetric:       totalRelaysServicedMetric,
		totalRequestsPerFunctionMetric:  totalRequestsPerFunctionMetric,
		inFlightPerFunctionMetric:       inFlightPerFunctionMetric,
		totalErrorsPerFunctionMetric:    totalErrorsPerFunctionMetric,
		requestLatencyPerFunctionMetric: requestLatencyPerFunctionMetric,
		totalErroredMetric:              totalErroredMetric,
		consumerQoSMetric:               consumerQoSMetric,
		loadRateMetric:                  loadRateMetric,
	}
	return pm
}
