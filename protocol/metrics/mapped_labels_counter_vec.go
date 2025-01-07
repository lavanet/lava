package metrics

import "github.com/prometheus/client_golang/prometheus"

// MappedLabelsCounterVec is a wrapper around prometheus.CounterVec that allows for setting labels dynamically.
// We use if for the metrics that have a dynamic number of labels, based on flags given upon startup.
type MappedLabelsCounterVec struct {
	MappedLabelsMetricBase
	*prometheus.CounterVec
}

func NewMappedLabelsCounterVec(opts MappedLabelsMetricOpts) *MappedLabelsCounterVec {
	metric := &MappedLabelsCounterVec{
		MappedLabelsMetricBase: MappedLabelsMetricBase{
			labels: opts.Labels,
		},
		CounterVec: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: opts.Name,
			Help: opts.Help,
		}, opts.Labels),
	}

	prometheus.MustRegister(metric.CounterVec)

	return metric
}

func (mlcv *MappedLabelsCounterVec) WithLabelValues(labelsWithValues map[string]string) prometheus.Counter {
	return mlcv.CounterVec.WithLabelValues(mlcv.getLabelValues(labelsWithValues)...)
}
