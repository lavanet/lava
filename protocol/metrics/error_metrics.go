package metrics

import (
	"sync"

	"github.com/lavanet/lava/v5/protocol/common"
	"github.com/lavanet/lava/v5/utils"
	"github.com/prometheus/client_golang/prometheus"
)

var errorMetricsOnce sync.Once

// InitErrorMetrics initializes the Prometheus error counter and registers
// it as the metrics callback in the error logging system.
// Call once during application startup (e.g., in consumer/provider server init).
func InitErrorMetrics() {
	errorMetricsOnce.Do(func() {
		// Label set: {error_name, error_category, retryable, chain_id}.
		//
		// Cardinality budget:
		//   - error_name:     ~100 registered codes
		//   - error_category: 2 values (internal, external)
		//   - retryable:      2 values
		//   - chain_id:       ~50 live chains
		// Theoretical max: 100 × 2 × 2 × 50 = 20,000 series.
		// Practical max is much lower because (a) category and retryable are
		// 1:1 derivable from error_name — each LavaError pins both at
		// registration — and (b) most (name, chain) pairs never fire.
		// Realistic steady-state: a few hundred series.
		//
		// Why we keep category and retryable as labels despite being
		// derivable from error_name: filter ergonomics. Dashboards that
		// want "rate of retryable errors by chain" should not have to join
		// against a separate mapping table — the label is cheap and the
		// cardinality is not actually multiplied by it.
		//
		// Why error_code is NOT a label: it would be a second 1:1-with-name
		// dimension that adds no filter power. error_code still appears in
		// structured logs for operator search; the Prometheus series only
		// needs one identity key.
		counter := prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "lava_errors_total",
			Help: "Total classified errors by name, category, retryability, and chain.",
		}, []string{"error_name", "error_category", "retryable", "chain_id"})

		// Best-effort registration. Three cases:
		//  1. Success → use the counter we just created.
		//  2. Already registered (e.g. from a previous test run or a
		//     duplicate InitErrorMetrics call) → reuse the existing
		//     collector so both callers share the same counter.
		//  3. Any OTHER registration failure (collector with same name but
		//     mismatched labels, Prometheus internal error, etc.) → log a
		//     warning so operators know lava_errors_total will not be
		//     scrapable. The callback is still wired up so the counter
		//     keeps accumulating in-process and can be reached via direct
		//     introspection, but /metrics will be missing the series.
		if err := prometheus.Register(counter); err != nil {
			if existing, ok := err.(prometheus.AlreadyRegisteredError); ok {
				if reused, ok := existing.ExistingCollector.(*prometheus.CounterVec); ok {
					counter = reused
				}
			} else {
				utils.LavaFormatWarning("failed to register lava_errors_total Prometheus counter; metric will not appear in /metrics scrapes", err)
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
