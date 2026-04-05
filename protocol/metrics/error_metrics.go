package metrics

import (
	"sync"

	"github.com/lavanet/lava/v5/protocol/common"
	"github.com/prometheus/client_golang/prometheus"
)

var errorMetricsOnce sync.Once

// InitErrorMetrics initializes the Prometheus error counter and registers
// it as the metrics callback in the error logging system.
// Call once during application startup (e.g., in consumer/provider server init).
func InitErrorMetrics() {
	errorMetricsOnce.Do(func() {
		// Labels: error_name (not error_code — redundant, higher cardinality),
		// error_category, retryable, chain_id.
		counter := prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "lava_errors_total",
			Help: "Total classified errors by name, category, retryability, and chain.",
		}, []string{"error_name", "error_category", "retryable", "chain_id"})

		// Best-effort registration — if already registered (e.g., in tests), reuse.
		if err := prometheus.Register(counter); err != nil {
			if existing, ok := err.(prometheus.AlreadyRegisteredError); ok {
				if reused, ok := existing.ExistingCollector.(*prometheus.CounterVec); ok {
					counter = reused
				}
			}
		}

		common.SetErrorMetricsCallback(func(_ uint32, errorName string, errorCategory string, retryable bool, chainID string) {
			retryStr := "false"
			if retryable {
				retryStr = "true"
			}
			counter.With(prometheus.Labels{
				"error_name":     errorName,
				"error_category": errorCategory,
				"retryable":      retryStr,
				"chain_id":       chainID,
			}).Inc()
		})
	})
}
