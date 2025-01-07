package metrics

type MappedLabelsMetricOpts struct {
	Name   string
	Help   string
	Labels []string
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
