package metrics

import (
	"net/http"

	"github.com/lavanet/lava/utils"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type HealthMetrics struct {
	failedRuns     *prometheus.CounterVec
	successfulRuns *prometheus.CounterVec
	// protocolVersionMetric      *prometheus.GaugeVec
}

func NewHealthMetrics(networkAddress string) *HealthMetrics {
	if networkAddress == DisabledFlagOption {
		utils.LavaFormatWarning("prometheus endpoint inactive, option is disabled", nil)
		return nil
	}
	failedRuns := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "lava_health_failed_runs",
		Help: "The total of runs failed",
	}, []string{"identifier"})

	successfulRuns := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "lava_health_successful_runs",
		Help: "The total of runs succeeded",
	}, []string{"identifier"})
	// Register the metrics with the Prometheus registry.
	prometheus.MustRegister(failedRuns)
	prometheus.MustRegister(successfulRuns)
	http.Handle("/metrics", promhttp.Handler())
	go func() {
		utils.LavaFormatInfo("prometheus endpoint listening", utils.Attribute{Key: "Listen Address", Value: networkAddress})
		http.ListenAndServe(networkAddress, nil)
	}()
	return &HealthMetrics{
		failedRuns:     failedRuns,
		successfulRuns: successfulRuns,
	}
}

func (pme *HealthMetrics) SetFailedRun(label string) {
	if pme == nil {
		return
	}
	pme.failedRuns.WithLabelValues(label).Add(1)
}

func (pme *HealthMetrics) SetSuccess(label string) {
	if pme == nil {
		return
	}
	pme.successfulRuns.WithLabelValues(label).Add(1)
}
