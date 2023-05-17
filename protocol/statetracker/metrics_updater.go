package statetracker

const (
	CallbackKeyForMetricUpdate = "metric-update"
)

type MetricsManagerInf interface {
	SetBlock(int64)
}

type MetricsUpdater struct {
	consumerMetricsManager MetricsManagerInf
}

func NewMetricsUpdater(consumerMetricsManager MetricsManagerInf) *MetricsUpdater {
	return &MetricsUpdater{consumerMetricsManager: consumerMetricsManager}
}

func (mu *MetricsUpdater) UpdaterKey() string {
	return CallbackKeyForMetricUpdate
}

func (mu *MetricsUpdater) Update(latestBlock int64) {
	if mu.consumerMetricsManager == nil {
		return
	}
	mu.consumerMetricsManager.SetBlock(latestBlock)
}
