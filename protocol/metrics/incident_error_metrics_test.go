package metrics

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/require"
)

var errorLabels = []string{"spec", "apiInterface", "provider_address", "method"}

func newConsumerForErrorTest() *ConsumerMetricsManager {
	return &ConsumerMetricsManager{
		incidentNodeErrorsTotalMetric:     prometheus.NewCounterVec(prometheus.CounterOpts{Name: "t_c_node_errors_total"}, errorLabels),
		incidentProtocolErrorsTotalMetric: prometheus.NewCounterVec(prometheus.CounterOpts{Name: "t_c_protocol_errors_total"}, errorLabels),
	}
}

// ---- Consumer node error tests ----

func TestConsumerSetRelayNodeErrorMetric_Increments(t *testing.T) {
	cmm := newConsumerForErrorTest()
	labels := []string{"ETH1", "jsonrpc", "provider1", "eth_blockNumber"}

	cmm.SetRelayNodeErrorMetric("ETH1", "jsonrpc", "provider1", "eth_blockNumber")

	require.Equal(t, float64(1), testutil.ToFloat64(cmm.incidentNodeErrorsTotalMetric.WithLabelValues(labels...)))
}

func TestConsumerSetRelayNodeErrorMetric_AccumulatesAcrossCalls(t *testing.T) {
	cmm := newConsumerForErrorTest()
	labels := []string{"ETH1", "jsonrpc", "provider1", "eth_blockNumber"}

	cmm.SetRelayNodeErrorMetric("ETH1", "jsonrpc", "provider1", "eth_blockNumber")
	cmm.SetRelayNodeErrorMetric("ETH1", "jsonrpc", "provider1", "eth_blockNumber")
	cmm.SetRelayNodeErrorMetric("ETH1", "jsonrpc", "provider1", "eth_blockNumber")

	require.Equal(t, float64(3), testutil.ToFloat64(cmm.incidentNodeErrorsTotalMetric.WithLabelValues(labels...)))
}

func TestConsumerSetRelayNodeErrorMetric_NilManager(t *testing.T) {
	var cmm *ConsumerMetricsManager
	require.NotPanics(t, func() {
		cmm.SetRelayNodeErrorMetric("ETH1", "jsonrpc", "provider1", "eth_blockNumber")
	})
}

// ---- Consumer protocol error tests ----

func TestConsumerSetProtocolError_Increments(t *testing.T) {
	cmm := newConsumerForErrorTest()
	labels := []string{"ETH1", "jsonrpc", "provider1", "eth_blockNumber"}

	cmm.SetProtocolError("ETH1", "jsonrpc", "provider1", "eth_blockNumber")

	require.Equal(t, float64(1), testutil.ToFloat64(cmm.incidentProtocolErrorsTotalMetric.WithLabelValues(labels...)))
}

func TestConsumerSetProtocolError_AccumulatesAcrossCalls(t *testing.T) {
	cmm := newConsumerForErrorTest()
	labels := []string{"ETH1", "jsonrpc", "provider1", "eth_blockNumber"}

	cmm.SetProtocolError("ETH1", "jsonrpc", "provider1", "eth_blockNumber")
	cmm.SetProtocolError("ETH1", "jsonrpc", "provider1", "eth_blockNumber")

	require.Equal(t, float64(2), testutil.ToFloat64(cmm.incidentProtocolErrorsTotalMetric.WithLabelValues(labels...)))
}

func TestConsumerSetProtocolError_NilManager(t *testing.T) {
	var cmm *ConsumerMetricsManager
	require.NotPanics(t, func() {
		cmm.SetProtocolError("ETH1", "jsonrpc", "provider1", "eth_blockNumber")
	})
}
