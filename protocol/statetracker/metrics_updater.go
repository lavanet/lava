package statetracker

import (
	"github.com/lavanet/lava/protocol/metrics"
)

const (
	CallbackKeyForMetricUpdate = "metric-update"
)

type MetricsUpdater struct {
	consumerMetricsManager *metrics.ConsumerMetricsManager
}

func NewMetricsUpdater(consumerMetricsManager *metrics.ConsumerMetricsManager) *MetricsUpdater {
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
