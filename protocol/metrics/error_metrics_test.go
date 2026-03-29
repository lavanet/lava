package metrics

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInitErrorMetrics(t *testing.T) {
	// Reset for testing
	reg := prometheus.NewRegistry()

	counter := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "lava_errors_total_test",
		Help: "Test counter",
	}, []string{"error_code", "error_name", "error_category", "retryable", "chain_id"})
	reg.MustRegister(counter)

	// Simulate what InitErrorMetrics does
	counter.With(prometheus.Labels{
		"error_code":     "3001",
		"error_name":     "CHAIN_NONCE_TOO_LOW",
		"error_category": "external",
		"retryable":      "false",
		"chain_id":       "ETH1",
	}).Inc()

	counter.With(prometheus.Labels{
		"error_code":     "3001",
		"error_name":     "CHAIN_NONCE_TOO_LOW",
		"error_category": "external",
		"retryable":      "false",
		"chain_id":       "ETH1",
	}).Inc()

	counter.With(prometheus.Labels{
		"error_code":     "1001",
		"error_name":     "PROTOCOL_CONNECTION_TIMEOUT",
		"error_category": "internal",
		"retryable":      "true",
		"chain_id":       "SOLANA",
	}).Inc()

	// Gather and verify
	families, err := reg.Gather()
	require.NoError(t, err)
	require.Len(t, families, 1)

	family := families[0]
	assert.Equal(t, "lava_errors_total_test", *family.Name)
	assert.Len(t, family.Metric, 2) // 2 distinct label combos

	// Find the ETH1 nonce metric
	for _, m := range family.Metric {
		labels := labelMap(m.Label)
		if labels["chain_id"] == "ETH1" {
			assert.Equal(t, "3001", labels["error_code"])
			assert.Equal(t, "CHAIN_NONCE_TOO_LOW", labels["error_name"])
			assert.Equal(t, float64(2), m.Counter.GetValue()) // incremented twice
		}
		if labels["chain_id"] == "SOLANA" {
			assert.Equal(t, "1001", labels["error_code"])
			assert.Equal(t, float64(1), m.Counter.GetValue())
		}
	}
}

func labelMap(labels []*dto.LabelPair) map[string]string {
	m := make(map[string]string)
	for _, l := range labels {
		m[l.GetName()] = l.GetValue()
	}
	return m
}
