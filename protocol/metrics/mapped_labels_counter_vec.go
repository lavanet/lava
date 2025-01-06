package metrics

import "github.com/prometheus/client_golang/prometheus"

// MappedLabelsCounterVec is a wrapper around prometheus.CounterVec that allows for setting labels dynamically.
// We use if for the metrics that have a dynamic number of labels, based on flags given upon startup.
type MappedLabelsCounterVec struct {
	*prometheus.CounterVec
	labels []string
}

type MappedLabelsCounterVecOpts struct {
	Name   string
	Help   string
	Labels []string
}

func NewMappedLabelsCounterVec(opts MappedLabelsCounterVecOpts) *MappedLabelsCounterVec {
	metric := &MappedLabelsCounterVec{
		labels: opts.Labels,
		CounterVec: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: opts.Name,
			Help: opts.Help,
		}, opts.Labels),
	}

	prometheus.MustRegister(metric.CounterVec)

	return metric
}

func (mlgv *MappedLabelsCounterVec) getLabelValues(labelsWithValues map[string]string) []string {
	labelValues := make([]string, len(mlgv.labels))
	for i, label := range mlgv.labels {
		labelValues[i] = labelsWithValues[label]
	}
	return labelValues
}

func (mlgv *MappedLabelsCounterVec) WithLabelValues(labelsWithValues map[string]string) prometheus.Counter {
	return mlgv.CounterVec.WithLabelValues(mlgv.getLabelValues(labelsWithValues)...)
}
