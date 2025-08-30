package updaters

const (
	CallbackKeyForMetricUpdate = "metric-update"
)

type MetricsManagerInf interface {
	SetBlock(int64)
}

type MetricsUpdater struct {
	metricsManager MetricsManagerInf
}

func NewMetricsUpdater(metricsManager MetricsManagerInf) *MetricsUpdater {
	return &MetricsUpdater{metricsManager: metricsManager}
}

func (mu *MetricsUpdater) UpdaterKey() string {
	return CallbackKeyForMetricUpdate
}

func (mu *MetricsUpdater) Reset(latestBlock int64) {
	mu.Update(latestBlock) // no new functionality on reset
}

func (mu *MetricsUpdater) Update(latestBlock int64) {
	if mu.metricsManager == nil {
		return
	}
	mu.metricsManager.SetBlock(latestBlock)
}
