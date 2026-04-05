package metrics

import (
	"fmt"
	"testing"

	"github.com/lavanet/lava/v5/protocol/common"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestErrorMetrics_EndToEnd verifies the full flow:
// InitErrorMetrics → LogCodedError → Prometheus counter incremented
func TestErrorMetrics_EndToEnd(t *testing.T) {
	// Use a fresh registry to avoid interference from other tests
	reg := prometheus.NewRegistry()
	counter := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "lava_errors_total_e2e",
		Help: "End-to-end test counter",
	}, []string{"error_code", "error_name", "error_category", "retryable", "chain_id"})
	reg.MustRegister(counter)

	// Register callback manually (simulating what InitErrorMetrics does)
	common.SetErrorMetricsCallback(func(errorCode uint32, errorName string, errorCategory string, retryable bool, chainID string) {
		retryStr := "false"
		if retryable {
			retryStr = "true"
		}
		counter.With(prometheus.Labels{
			"error_code":     formatCode(errorCode),
			"error_name":     errorName,
			"error_category": errorCategory,
			"retryable":      retryStr,
			"chain_id":       chainID,
		}).Inc()
	})
	defer common.SetErrorMetricsCallback(nil) // cleanup

	// Simulate errors from the relay pipeline
	common.LogCodedError("node error", nil, common.LavaErrorChainNonceTooLow, "ETH1", -32000, "nonce too low")
	common.LogCodedError("node error", nil, common.LavaErrorChainNonceTooLow, "ETH1", -32000, "nonce too low")
	common.LogCodedError("timeout", nil, common.LavaErrorConnectionTimeout, "SOLANA", 0, "")
	common.LogCodedError("rate limit", nil, common.LavaErrorNodeRateLimited, "ETH1", 429, "")

	// Gather and verify
	families, err := reg.Gather()
	require.NoError(t, err)
	require.Len(t, families, 1)

	family := families[0]
	assert.Equal(t, "lava_errors_total_e2e", *family.Name)
	assert.Len(t, family.Metric, 3) // 3 distinct label combos

	for _, m := range family.Metric {
		labels := labelMap(m.Label)
		switch labels["error_name"] {
		case "CHAIN_NONCE_TOO_LOW":
			assert.Equal(t, "3001", labels["error_code"])
			assert.Equal(t, "external", labels["error_category"])
			assert.Equal(t, "false", labels["retryable"])
			assert.Equal(t, "ETH1", labels["chain_id"])
			assert.Equal(t, float64(2), m.Counter.GetValue(), "should have 2 nonce errors")
		case "PROTOCOL_CONNECTION_TIMEOUT":
			assert.Equal(t, "1001", labels["error_code"])
			assert.Equal(t, "internal", labels["error_category"])
			assert.Equal(t, "true", labels["retryable"])
			assert.Equal(t, "SOLANA", labels["chain_id"])
			assert.Equal(t, float64(1), m.Counter.GetValue())
		case "NODE_RATE_LIMITED":
			assert.Equal(t, "2005", labels["error_code"])
			assert.Equal(t, "external", labels["error_category"])
			assert.Equal(t, "true", labels["retryable"])
			assert.Equal(t, float64(1), m.Counter.GetValue())
		default:
			t.Errorf("unexpected error name: %s", labels["error_name"])
		}
	}
}

func formatCode(code uint32) string {
	return fmt.Sprintf("%d", code)
}
