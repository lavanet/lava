package metrics

import "github.com/prometheus/client_golang/prometheus"

type MappedLabelsGaugeVec struct {
	*prometheus.GaugeVec
	labels []string
}

type MappedLabelsGaugeVecOpts struct {
	Name   string
	Help   string
	Labels []string
}

func NewMappedLabelsGaugeVec(opts MappedLabelsGaugeVecOpts) *MappedLabelsGaugeVec {
	metric := &MappedLabelsGaugeVec{
		labels: opts.Labels,
		GaugeVec: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: opts.Name,
			Help: opts.Help,
		}, opts.Labels),
	}

	prometheus.MustRegister(metric.GaugeVec)

	return metric
}

func (mlgv *MappedLabelsGaugeVec) getLabelValues(labelsWithValues map[string]string) []string {
	labelValues := make([]string, len(mlgv.labels))
	for i, label := range mlgv.labels {
		labelValues[i] = labelsWithValues[label]
	}
	return labelValues
}

func (mlgv *MappedLabelsGaugeVec) WithLabelValues(labelsWithValues map[string]string) prometheus.Gauge {
	return mlgv.GaugeVec.WithLabelValues(mlgv.getLabelValues(labelsWithValues)...)
}
