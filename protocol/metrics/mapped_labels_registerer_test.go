package metrics

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/require"
)

func TestNewMappedLabelsGaugeVec_WithRegisterer(t *testing.T) {
	reg := prometheus.NewRegistry()
	labels := []string{"chain", "provider"}

	g := NewMappedLabelsGaugeVec(MappedLabelsMetricOpts{
		Name:       "test_gauge_reg",
		Help:       "help",
		Labels:     labels,
		Registerer: reg,
	})

	g.WithLabelValues(map[string]string{"chain": "ETH1", "provider": "p1"}).Set(42)
	require.Equal(t, float64(42), testutil.ToFloat64(g.GaugeVec.WithLabelValues("ETH1", "p1")))
}

func TestNewMappedLabelsGaugeVec_WithRegisterer_Reuse(t *testing.T) {
	reg := prometheus.NewRegistry()
	labels := []string{"chain"}

	g1 := NewMappedLabelsGaugeVec(MappedLabelsMetricOpts{
		Name:       "test_gauge_reuse",
		Help:       "help",
		Labels:     labels,
		Registerer: reg,
	})
	g1.WithLabelValues(map[string]string{"chain": "ETH1"}).Set(10)

	// Register again with same name — should reuse the existing collector.
	g2 := NewMappedLabelsGaugeVec(MappedLabelsMetricOpts{
		Name:       "test_gauge_reuse",
		Help:       "help",
		Labels:     labels,
		Registerer: reg,
	})

	require.Equal(t, float64(10), testutil.ToFloat64(g2.GaugeVec.WithLabelValues("ETH1")))
}

func TestNewMappedLabelsGaugeVec_WithoutRegisterer(t *testing.T) {
	labels := []string{"chain"}

	g := NewMappedLabelsGaugeVec(MappedLabelsMetricOpts{
		Name:   "test_gauge_noreg",
		Help:   "help",
		Labels: labels,
	})

	g.WithLabelValues(map[string]string{"chain": "ETH1"}).Set(7)
	require.Equal(t, float64(7), testutil.ToFloat64(g.GaugeVec.WithLabelValues("ETH1")))
}

func TestNewMappedLabelsCounterVec_WithRegisterer(t *testing.T) {
	reg := prometheus.NewRegistry()
	labels := []string{"chain", "provider"}

	c := NewMappedLabelsCounterVec(MappedLabelsMetricOpts{
		Name:       "test_counter_reg",
		Help:       "help",
		Labels:     labels,
		Registerer: reg,
	})

	c.WithLabelValues(map[string]string{"chain": "ETH1", "provider": "p1"}).Inc()
	require.Equal(t, float64(1), testutil.ToFloat64(c.CounterVec.WithLabelValues("ETH1", "p1")))
}

func TestNewMappedLabelsCounterVec_WithRegisterer_Reuse(t *testing.T) {
	reg := prometheus.NewRegistry()
	labels := []string{"chain"}

	c1 := NewMappedLabelsCounterVec(MappedLabelsMetricOpts{
		Name:       "test_counter_reuse",
		Help:       "help",
		Labels:     labels,
		Registerer: reg,
	})
	c1.WithLabelValues(map[string]string{"chain": "ETH1"}).Add(5)

	c2 := NewMappedLabelsCounterVec(MappedLabelsMetricOpts{
		Name:       "test_counter_reuse",
		Help:       "help",
		Labels:     labels,
		Registerer: reg,
	})

	require.Equal(t, float64(5), testutil.ToFloat64(c2.CounterVec.WithLabelValues("ETH1")))
}
