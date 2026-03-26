package metrics

import "github.com/prometheus/client_golang/prometheus"

type MappedLabelsMetricOpts struct {
	Name       string
	Help       string
	Labels     []string
	Registerer prometheus.Registerer
}

type MappedLabelsMetricBase struct {
	labels []string
}

func (mlmb *MappedLabelsMetricBase) getLabelValues(labelsWithValues map[string]string) []string {
	labelValues := make([]string, len(mlmb.labels))
	for i, label := range mlmb.labels {
		labelValues[i] = labelsWithValues[label]
	}
	return labelValues
}
