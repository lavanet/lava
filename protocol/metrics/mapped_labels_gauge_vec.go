package metrics

import (
	"errors"

	"github.com/prometheus/client_golang/prometheus"
)

// MappedLabelsGaugeVec is a wrapper around prometheus.GaugeVec that allows for setting labels dynamically.
// We use if for the metrics that have a dynamic number of labels, based on flags given upon startup.
type MappedLabelsGaugeVec struct {
	MappedLabelsMetricBase
	*prometheus.GaugeVec
}

func NewMappedLabelsGaugeVec(opts MappedLabelsMetricOpts) *MappedLabelsGaugeVec {
	metric := &MappedLabelsGaugeVec{
		MappedLabelsMetricBase: MappedLabelsMetricBase{
			labels: opts.Labels,
		},
		GaugeVec: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: opts.Name,
			Help: opts.Help,
		}, opts.Labels),
	}

	if opts.Registerer != nil {
		if err := opts.Registerer.Register(metric.GaugeVec); err != nil {
			are := &prometheus.AlreadyRegisteredError{}
			if errors.As(err, are) {
				if existing, ok := are.ExistingCollector.(*prometheus.GaugeVec); ok {
					metric.GaugeVec = existing
				} else {
					panic(are)
				}
			} else {
				panic(err)
			}
		}
	}

	return metric
}

func (mlgv *MappedLabelsGaugeVec) WithLabelValues(labelsWithValues map[string]string) prometheus.Gauge {
	return mlgv.GaugeVec.WithLabelValues(mlgv.getLabelValues(labelsWithValues)...)
}
