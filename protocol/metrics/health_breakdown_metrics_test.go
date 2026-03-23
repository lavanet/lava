package metrics

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/require"
)

func newSmartRouterForHealthBreakdownTest() *SmartRouterMetricsManager {
	labels := []string{"spec", "apiInterface"}
	return &SmartRouterMetricsManager{
		routerOverallHealthBreakdown:   prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: "t_sr_router_health_breakdown"}, labels),
		endpointOverallHealthBreakdown: prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: "t_sr_endpoint_health_breakdown"}, labels),
	}
}

// ---- lava_rpcsmartrouter_overall_health_breakdown ----

func TestSmartRouterUpdateHealthcheckStatusBreakdown_Healthy(t *testing.T) {
	m := newSmartRouterForHealthBreakdownTest()

	m.UpdateHealthcheckStatusBreakdown("ETH1", "jsonrpc", true)

	require.Equal(t, float64(1), testutil.ToFloat64(m.routerOverallHealthBreakdown.WithLabelValues("ETH1", "jsonrpc")))
}

func TestSmartRouterUpdateHealthcheckStatusBreakdown_Unhealthy(t *testing.T) {
	m := newSmartRouterForHealthBreakdownTest()

	m.UpdateHealthcheckStatusBreakdown("ETH1", "jsonrpc", false)

	require.Equal(t, float64(0), testutil.ToFloat64(m.routerOverallHealthBreakdown.WithLabelValues("ETH1", "jsonrpc")))
}

func TestSmartRouterUpdateHealthcheckStatusBreakdown_Transition(t *testing.T) {
	m := newSmartRouterForHealthBreakdownTest()

	m.UpdateHealthcheckStatusBreakdown("ETH1", "jsonrpc", true)
	require.Equal(t, float64(1), testutil.ToFloat64(m.routerOverallHealthBreakdown.WithLabelValues("ETH1", "jsonrpc")))

	m.UpdateHealthcheckStatusBreakdown("ETH1", "jsonrpc", false)
	require.Equal(t, float64(0), testutil.ToFloat64(m.routerOverallHealthBreakdown.WithLabelValues("ETH1", "jsonrpc")))
}

func TestSmartRouterUpdateHealthcheckStatusBreakdown_IndependentChains(t *testing.T) {
	m := newSmartRouterForHealthBreakdownTest()

	m.UpdateHealthcheckStatusBreakdown("ETH1", "jsonrpc", true)
	m.UpdateHealthcheckStatusBreakdown("NEAR", "jsonrpc", false)

	require.Equal(t, float64(1), testutil.ToFloat64(m.routerOverallHealthBreakdown.WithLabelValues("ETH1", "jsonrpc")))
	require.Equal(t, float64(0), testutil.ToFloat64(m.routerOverallHealthBreakdown.WithLabelValues("NEAR", "jsonrpc")))
}

func TestSmartRouterUpdateHealthcheckStatusBreakdown_NilSafe(t *testing.T) {
	var m *SmartRouterMetricsManager
	require.NotPanics(t, func() { m.UpdateHealthcheckStatusBreakdown("ETH1", "jsonrpc", true) })
}

// ---- lava_rpc_endpoint_overall_health_breakdown ----

func TestSmartRouterSetEndpointOverallHealthBreakdown_Healthy(t *testing.T) {
	m := newSmartRouterForHealthBreakdownTest()

	m.SetEndpointOverallHealthBreakdown("ETH1", "jsonrpc", true)

	require.Equal(t, float64(1), testutil.ToFloat64(m.endpointOverallHealthBreakdown.WithLabelValues("ETH1", "jsonrpc")))
}

func TestSmartRouterSetEndpointOverallHealthBreakdown_Unhealthy(t *testing.T) {
	m := newSmartRouterForHealthBreakdownTest()

	m.SetEndpointOverallHealthBreakdown("ETH1", "jsonrpc", false)

	require.Equal(t, float64(0), testutil.ToFloat64(m.endpointOverallHealthBreakdown.WithLabelValues("ETH1", "jsonrpc")))
}

func TestSmartRouterSetEndpointOverallHealthBreakdown_Transition(t *testing.T) {
	m := newSmartRouterForHealthBreakdownTest()

	m.SetEndpointOverallHealthBreakdown("ETH1", "jsonrpc", true)
	require.Equal(t, float64(1), testutil.ToFloat64(m.endpointOverallHealthBreakdown.WithLabelValues("ETH1", "jsonrpc")))

	m.SetEndpointOverallHealthBreakdown("ETH1", "jsonrpc", false)
	require.Equal(t, float64(0), testutil.ToFloat64(m.endpointOverallHealthBreakdown.WithLabelValues("ETH1", "jsonrpc")))
}

func TestSmartRouterSetEndpointOverallHealthBreakdown_IndependentChains(t *testing.T) {
	m := newSmartRouterForHealthBreakdownTest()

	m.SetEndpointOverallHealthBreakdown("ETH1", "jsonrpc", true)
	m.SetEndpointOverallHealthBreakdown("NEAR", "jsonrpc", false)

	require.Equal(t, float64(1), testutil.ToFloat64(m.endpointOverallHealthBreakdown.WithLabelValues("ETH1", "jsonrpc")))
	require.Equal(t, float64(0), testutil.ToFloat64(m.endpointOverallHealthBreakdown.WithLabelValues("NEAR", "jsonrpc")))
}

func TestSmartRouterSetEndpointOverallHealthBreakdown_NilSafe(t *testing.T) {
	var m *SmartRouterMetricsManager
	require.NotPanics(t, func() { m.SetEndpointOverallHealthBreakdown("ETH1", "jsonrpc", true) })
}
