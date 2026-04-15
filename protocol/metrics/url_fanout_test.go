package metrics

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/require"
)

// newSmartRouterForURLFanoutTest builds a SmartRouterMetricsManager with only the
// fields the fan-out tests exercise. Gauges/counters are registered without a
// Registerer to avoid polluting the process-global Prometheus registry across
// parallel test packages.
func newSmartRouterForURLFanoutTest() *SmartRouterMetricsManager {
	endpointLabels := []string{"spec", "apiInterface", "endpoint_id"}
	return &SmartRouterMetricsManager{
		endpointLatestBlock: NewMappedLabelsGaugeVec(MappedLabelsMetricOpts{
			Name:   "t_sr_endpoint_latest_block",
			Labels: endpointLabels,
		}),
		endpointFetchLatestSuccess: NewMappedLabelsCounterVec(MappedLabelsMetricOpts{
			Name:   "t_sr_endpoint_fetch_latest_success",
			Labels: endpointLabels,
		}),
		endpointFetchLatestFails: NewMappedLabelsCounterVec(MappedLabelsMetricOpts{
			Name:   "t_sr_endpoint_fetch_latest_fails",
			Labels: endpointLabels,
		}),
		endpointFetchBlockSuccess: NewMappedLabelsCounterVec(MappedLabelsMetricOpts{
			Name:   "t_sr_endpoint_fetch_block_success",
			Labels: endpointLabels,
		}),
		endpointFetchBlockFails: NewMappedLabelsCounterVec(MappedLabelsMetricOpts{
			Name:   "t_sr_endpoint_fetch_block_fails",
			Labels: endpointLabels,
		}),
		endpointOverallHealth: NewMappedLabelsGaugeVec(MappedLabelsMetricOpts{
			Name:   "t_sr_endpoint_overall_health",
			Labels: endpointLabels,
		}),
		endpointMetrics:    make(map[string]*EndpointMetrics),
		urlToProviderNames: make(map[string][]string),
	}
}

// TestResolveProviderNames_FansOutAcrossSharedURL is the core fan-out guarantee:
// registering two providers at the same URL must cause resolveProviderNames(URL)
// to return both, so URL-keyed metric emissions reach every provider.
func TestResolveProviderNames_FansOutAcrossSharedURL(t *testing.T) {
	m := newSmartRouterForURLFanoutTest()
	const sharedURL = "https://base.lava.build:443/"

	m.RegisterEndpoint("BASE", "jsonrpc", sharedURL, "lava1")
	m.RegisterEndpoint("BASE", "jsonrpc", sharedURL, "lava2")

	names := m.resolveProviderNames(sharedURL)
	require.ElementsMatch(t, []string{"lava1", "lava2"}, names,
		"both providers on a shared URL must resolve together")
}

// TestResolveProviderNames_DeduplicatesSameProvider covers the common case where
// a provider's config lists the same node-url multiple times (typical for HA pairs
// pointing at a single upstream). RegisterEndpoint is called once per url, but the
// set of names per URL should not grow duplicates for the same provider.
func TestResolveProviderNames_DeduplicatesSameProvider(t *testing.T) {
	m := newSmartRouterForURLFanoutTest()
	const sharedURL = "https://base.blockpi.network/v1/rpc/abc"

	m.RegisterEndpoint("BASE", "jsonrpc", sharedURL, "blockpi1")
	m.RegisterEndpoint("BASE", "jsonrpc", sharedURL, "blockpi1") // duplicate node-url

	names := m.resolveProviderNames(sharedURL)
	require.Equal(t, []string{"blockpi1"}, names,
		"duplicate (URL, providerName) registrations must not produce duplicate entries")
}

// TestResolveProviderNames_FallsBackToInput guards the backward-compatible behavior
// for callers that pass a provider name directly (e.g. the relay hot path, which
// already knows the provider it's hitting). An unknown key must round-trip as a
// single-element slice so existing callers keep working.
func TestResolveProviderNames_FallsBackToInput(t *testing.T) {
	m := newSmartRouterForURLFanoutTest()

	names := m.resolveProviderNames("provider-not-registered-as-url")
	require.Equal(t, []string{"provider-not-registered-as-url"}, names)

	// resolveProviderName (single) must pick the first name for consistency.
	require.Equal(t, "provider-not-registered-as-url",
		m.resolveProviderName("provider-not-registered-as-url"))
}

// TestSetEndpointLatestBlock_FansOutAcrossSharedURL verifies the actual metric
// emission path: a single SetEndpointLatestBlock(URL, block) call must populate
// the gauge for every provider sharing that URL. Before the fan-out fix, only the
// last-registered provider got the metric — the rest stayed at zero.
func TestSetEndpointLatestBlock_FansOutAcrossSharedURL(t *testing.T) {
	m := newSmartRouterForURLFanoutTest()
	const sharedURL = "https://base.lava.build:443/"
	const block int64 = 44615090

	m.RegisterEndpoint("BASE", "jsonrpc", sharedURL, "lava1")
	m.RegisterEndpoint("BASE", "jsonrpc", sharedURL, "lava2")

	m.SetEndpointLatestBlock("BASE", "jsonrpc", sharedURL, block)

	lava1 := testutil.ToFloat64(m.endpointLatestBlock.WithLabelValues(map[string]string{
		"spec": "BASE", "apiInterface": "jsonrpc", "endpoint_id": "lava1",
	}))
	lava2 := testutil.ToFloat64(m.endpointLatestBlock.WithLabelValues(map[string]string{
		"spec": "BASE", "apiInterface": "jsonrpc", "endpoint_id": "lava2",
	}))
	require.Equal(t, float64(block), lava1, "lava1 must receive the block update from the shared URL")
	require.Equal(t, float64(block), lava2, "lava2 must receive the block update from the shared URL")
}

// TestRecordBlockFetch_FansOutAcrossSharedURL covers the other URL-keyed emitter
// (chain tracker success/fail counters). Each call must increment counters for
// every provider on the URL so the fetch success rate reflects reality per-provider.
func TestRecordBlockFetch_FansOutAcrossSharedURL(t *testing.T) {
	m := newSmartRouterForURLFanoutTest()
	const sharedURL = "https://base.blockpi.network/v1/rpc/abc"

	m.RegisterEndpoint("BASE", "jsonrpc", sharedURL, "blockpi1")
	m.RegisterEndpoint("BASE", "jsonrpc", sharedURL, "blockpi2")

	// isLatest=true, success=true → hits endpointFetchLatestSuccess for each provider.
	m.RecordBlockFetch("BASE", "jsonrpc", sharedURL, true, true)

	bp1 := testutil.ToFloat64(m.endpointFetchLatestSuccess.WithLabelValues(map[string]string{
		"spec": "BASE", "apiInterface": "jsonrpc", "endpoint_id": "blockpi1",
	}))
	bp2 := testutil.ToFloat64(m.endpointFetchLatestSuccess.WithLabelValues(map[string]string{
		"spec": "BASE", "apiInterface": "jsonrpc", "endpoint_id": "blockpi2",
	}))
	require.Equal(t, float64(1), bp1, "blockpi1 must be incremented")
	require.Equal(t, float64(1), bp2, "blockpi2 must be incremented")
}
