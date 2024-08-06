package metrics

import (
	"net/http"

	"github.com/lavanet/lava/v2/utils"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type HealthMetrics struct {
	failedRuns      *prometheus.CounterVec
	successfulRuns  *prometheus.CounterVec
	failureAlerts   *prometheus.GaugeVec
	healthyChecks   *prometheus.GaugeVec
	unhealthyChecks *prometheus.GaugeVec
	latestBlocks    *prometheus.GaugeVec
}

func NewHealthMetrics(networkAddress string) *HealthMetrics {
	if networkAddress == DisabledFlagOption {
		utils.LavaFormatWarning("prometheus endpoint inactive, option is disabled", nil)
		return nil
	}

	latestBlocks := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "lava_health_latest_blocks",
		Help: "The latest blocks queried on all checks",
	}, []string{"identifier", "entity"})

	failureAlerts := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "lava_health_failure_alerts",
		Help: "The current amount of active alerts",
	}, []string{"identifier"})

	healthyChecks := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "lava_healthy_entities",
		Help: "The current amount of healthy checks",
	}, []string{"identifier"})

	unhealthyChecks := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "lava_unhealthy_entities",
		Help: "The current amount of healthy checks",
	}, []string{"identifier"})

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
	prometheus.MustRegister(failureAlerts)
	prometheus.MustRegister(healthyChecks)
	prometheus.MustRegister(unhealthyChecks)
	prometheus.MustRegister(latestBlocks)
	http.Handle("/metrics", promhttp.Handler())
	go func() {
		utils.LavaFormatInfo("prometheus endpoint listening", utils.Attribute{Key: "Listen Address", Value: networkAddress})
		http.ListenAndServe(networkAddress, nil)
	}()
	return &HealthMetrics{
		failedRuns:      failedRuns,
		successfulRuns:  successfulRuns,
		failureAlerts:   failureAlerts,
		healthyChecks:   healthyChecks,
		unhealthyChecks: unhealthyChecks,
		latestBlocks:    latestBlocks,
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

func (pme *HealthMetrics) SetLatestBlockData(label string, data map[string]uint64) {
	if pme == nil {
		return
	}
	for entity, value := range data {
		pme.latestBlocks.WithLabelValues(label, entity).Set(float64(value))
	}
}

func (pme *HealthMetrics) SetAlertResults(label string, fails uint64, unhealthy uint64, healthy uint64) {
	if pme == nil {
		return
	}
	pme.failureAlerts.WithLabelValues(label).Set(float64(fails))
	pme.unhealthyChecks.WithLabelValues(label).Set(float64(unhealthy))
	pme.healthyChecks.WithLabelValues(label).Set(float64(healthy))
}
