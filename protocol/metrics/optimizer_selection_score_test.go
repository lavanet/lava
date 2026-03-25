package metrics

import (
	"context"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/require"
)

type stubOptimizerQoSClient struct {
	reports []OptimizerQoSReportToSend
}

func (s *stubOptimizerQoSClient) GetReportsToSend() []OptimizerQoSReportToSend {
	return s.reports
}

func newSmartRouterForOptimizerTest(client *ConsumerOptimizerQoSClient) *SmartRouterMetricsManager {
	scoreLabels := []string{"spec", "apiInterface", "endpoint_id", "score_type"}
	optimizerLabels := []string{"spec", "endpoint_id", "score_type"}
	return &SmartRouterMetricsManager{
		endpointSelectionScore:  prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: "t_endpoint_sel"}, scoreLabels),
		optimizerSelectionScore: prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: "t_optimizer_sel"}, optimizerLabels),
		optimizerQoSClient:      client,
	}
}

func TestStartSelectionStatsUpdater_WritesToOptimizerMetric(t *testing.T) {
	coqc := NewConsumerOptimizerQoSClient("addr", "", 1)
	coqc.SetReportsToSend([]OptimizerQoSReportToSend{
		{
			ChainId:               "ETH1",
			ProviderAddress:       "provider1",
			SelectionAvailability: 0.9,
			SelectionLatency:      0.8,
			SelectionSync:         0.7,
			SelectionStake:        0.6,
			SelectionComposite:    0.85,
		},
	})

	m := newSmartRouterForOptimizerTest(coqc)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	m.StartSelectionStatsUpdater(ctx, 10*time.Millisecond)

	require.Eventually(t, func() bool {
		return testutil.ToFloat64(m.optimizerSelectionScore.WithLabelValues("ETH1", "provider1", "composite")) == 0.85
	}, time.Second, 5*time.Millisecond)

	require.Equal(t, 0.9, testutil.ToFloat64(m.optimizerSelectionScore.WithLabelValues("ETH1", "provider1", "availability")))
	require.Equal(t, 0.8, testutil.ToFloat64(m.optimizerSelectionScore.WithLabelValues("ETH1", "provider1", "latency")))
	require.Equal(t, 0.7, testutil.ToFloat64(m.optimizerSelectionScore.WithLabelValues("ETH1", "provider1", "sync")))
	require.Equal(t, 0.6, testutil.ToFloat64(m.optimizerSelectionScore.WithLabelValues("ETH1", "provider1", "stake")))
}

func TestStartSelectionStatsUpdater_DoesNotWriteToEndpointMetric(t *testing.T) {
	coqc := NewConsumerOptimizerQoSClient("addr", "", 1)
	coqc.SetReportsToSend([]OptimizerQoSReportToSend{
		{
			ChainId:            "ETH1",
			ProviderAddress:    "provider1",
			SelectionComposite: 0.85,
		},
	})

	m := newSmartRouterForOptimizerTest(coqc)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	m.StartSelectionStatsUpdater(ctx, 10*time.Millisecond)

	require.Eventually(t, func() bool {
		return testutil.ToFloat64(m.optimizerSelectionScore.WithLabelValues("ETH1", "provider1", "composite")) == 0.85
	}, time.Second, 5*time.Millisecond)

	// endpointSelectionScore should NOT have been touched
	require.Equal(t, float64(0), testutil.ToFloat64(m.endpointSelectionScore.WithLabelValues("ETH1", "", "provider1", "composite")))
}

func TestSetProviderSelected_WritesToEndpointMetric(t *testing.T) {
	m := newSmartRouterForOptimizerTest(nil)

	scores := []ProviderSelectionScores{
		{
			ProviderAddress: "provider1",
			Availability:    0.95,
			Latency:         0.88,
			Sync:            0.77,
			Stake:           0.66,
			Composite:       0.9,
		},
	}

	m.SetProviderSelected("ETH1", "jsonrpc", "", scores, 0)

	require.Equal(t, 0.95, testutil.ToFloat64(m.endpointSelectionScore.WithLabelValues("ETH1", "jsonrpc", "provider1", "availability")))
	require.Equal(t, 0.88, testutil.ToFloat64(m.endpointSelectionScore.WithLabelValues("ETH1", "jsonrpc", "provider1", "latency")))
	require.Equal(t, 0.77, testutil.ToFloat64(m.endpointSelectionScore.WithLabelValues("ETH1", "jsonrpc", "provider1", "sync")))
	require.Equal(t, 0.66, testutil.ToFloat64(m.endpointSelectionScore.WithLabelValues("ETH1", "jsonrpc", "provider1", "stake")))
	require.Equal(t, 0.9, testutil.ToFloat64(m.endpointSelectionScore.WithLabelValues("ETH1", "jsonrpc", "provider1", "composite")))
}

func TestStartSelectionStatsUpdater_NilSafe(t *testing.T) {
	var m *SmartRouterMetricsManager
	require.NotPanics(t, func() {
		m.StartSelectionStatsUpdater(context.Background(), time.Second)
	})
}

func TestSetProviderSelected_NilSafe(t *testing.T) {
	var m *SmartRouterMetricsManager
	require.NotPanics(t, func() {
		m.SetProviderSelected("ETH1", "jsonrpc", "", nil, 0)
	})
}
