package metrics

import (
	"context"
	"time"

	pairingtypes "github.com/lavanet/lava/v5/x/pairing/types"
)

// NoOpConsumerMetrics is returned by SafeMetrics when the caller passes nil.
// Every method is a harmless no-op so downstream code never needs nil guards.
var _ ConsumerMetricsManagerInf = NoOpConsumerMetrics{}

type NoOpConsumerMetrics struct{}

func (NoOpConsumerMetrics) SetRelayMetrics(*RelayMetrics, error)                           {}
func (NoOpConsumerMetrics) RecordEndToEndLatency(string, string, string, float64)          {}
func (NoOpConsumerMetrics) RecordProviderLatency(string, string, string, string, float64)  {}
func (NoOpConsumerMetrics) RecordCacheResult(string, string, string, bool, float64)        {}
func (NoOpConsumerMetrics) SetRelayNodeErrorMetric(string, string, string, string)         {}
func (NoOpConsumerMetrics) SetProtocolError(string, string, string, string)                {}
func (NoOpConsumerMetrics) RecordIncidentRetry(string, string, string, uint64, bool)       {}
func (NoOpConsumerMetrics) RecordIncidentConsistency(string, string, string, bool)         {}
func (NoOpConsumerMetrics) RecordIncidentHedgeResult(string, string, string, uint64, bool) {}
func (NoOpConsumerMetrics) SetCrossValidationMetric(string, string, string, bool, []string, []string) {
}
func (NoOpConsumerMetrics) UpdateHealthCheckStatus(bool)                          {}
func (NoOpConsumerMetrics) UpdateHealthcheckStatusBreakdown(string, string, bool) {}
func (NoOpConsumerMetrics) SetProviderLiveness(string, string, string, bool)      {}
func (NoOpConsumerMetrics) SetProviderSelected(string, string, string, []ProviderSelectionScores, float64) {
}
func (NoOpConsumerMetrics) SetBlockedProvider(string, string, string, string, bool)  {}
func (NoOpConsumerMetrics) SetProbeOutlierBlock(string, string, string, int64, int64, int64) {}
func (NoOpConsumerMetrics) SetMajorityBaselineConsensusFailure(string, string, int)           {}
func (NoOpConsumerMetrics) SetAlignLatestBlockOutcome(string, string, string)                 {}
func (NoOpConsumerMetrics) SetLatestBlockOutlierRejected(string)                              {}
func (NoOpConsumerMetrics) SetChainStateLatestBlock(string, int64)                            {}
func (NoOpConsumerMetrics) SetMajorityBaselineGauge(string, string, int64)                    {}
func (NoOpConsumerMetrics) SetAlignLatestBlockGap(string, string, int64)                      {}
func (NoOpConsumerMetrics) SetSyncScoringOutlierSkipped(string, string, string)               {}
func (NoOpConsumerMetrics) SetSharedStatePropagation(string)                                  {}
func (NoOpConsumerMetrics) SetQOSMetrics(string, string, string, string, *pairingtypes.QualityOfServiceReport, *pairingtypes.QualityOfServiceReport, int64, uint64, time.Duration, bool) {
}
func (NoOpConsumerMetrics) ResetSessionRelatedMetrics()                                    {}
func (NoOpConsumerMetrics) ResetBlockedProvidersMetrics(string, string, map[string]string) {}
func (NoOpConsumerMetrics) SetWsSubscriptionRequestMetric(string, string)                  {}
func (NoOpConsumerMetrics) SetFailedWsSubscriptionRequestMetric(string, string)            {}
func (NoOpConsumerMetrics) SetWebSocketConnectionActive(string, string, bool)              {}
func (NoOpConsumerMetrics) SetVersion(string)                                              {}
func (NoOpConsumerMetrics) StartSelectionStatsUpdater(context.Context, time.Duration)      {}

// SafeMetrics returns m if non-nil, otherwise a NoOpConsumerMetrics.
// Use this in constructors to avoid storing a nil interface.
func SafeMetrics(m ConsumerMetricsManagerInf) ConsumerMetricsManagerInf {
	if m == nil {
		return NoOpConsumerMetrics{}
	}
	return m
}

// ConsumerMetricsManagerInf is the interface satisfied by both ConsumerMetricsManager
// (for the real rpcconsumer) and SmartRouterMetricsManager (for the smart router).
// Downstream components (RPCConsumerLogs, ConsumerSessionManager,
// DirectWSSubscriptionManager) accept this interface so each process can supply
// its own implementation without leaking metrics from the other.
type ConsumerMetricsManagerInf interface {
	// --- Relay tracking (RPCConsumerLogs) ---
	SetRelayMetrics(relayMetric *RelayMetrics, err error)

	// --- Latency ---
	RecordEndToEndLatency(chainId string, apiInterface string, method string, latencyMs float64)
	RecordProviderLatency(chainId string, apiInterface string, providerAddress string, method string, latencyMs float64)

	// --- Cache ---
	RecordCacheResult(chainId, apiInterface, method string, hit bool, latencyMs float64)

	// --- Errors (RPCConsumerLogs) ---
	SetRelayNodeErrorMetric(chainId string, apiInterface string, providerAddress string, method string)
	SetProtocolError(chainId string, apiInterface string, providerAddress string, method string)

	// --- Incidents (appendHeadersToRelayResult / RPCConsumerLogs) ---
	RecordIncidentRetry(chainId string, apiInterface string, method string, count uint64, success bool)
	RecordIncidentConsistency(chainId string, apiInterface string, method string, success bool)
	RecordIncidentHedgeResult(chainId string, apiInterface string, method string, count uint64, success bool)

	// --- Cross-validation (RPCConsumerLogs) ---
	SetCrossValidationMetric(chainId, apiInterface, method string, success bool, agreeingProviders, disagreeingProviders []string)

	// --- Health (RelaysMonitorAggregator) ---
	UpdateHealthCheckStatus(status bool)
	UpdateHealthcheckStatusBreakdown(chainId, apiInterface string, status bool)

	// --- Provider state (ConsumerSessionManager) ---
	SetProviderLiveness(chainId string, providerAddress string, providerEndpoint string, isAlive bool)
	SetProviderSelected(chainId string, apiInterface string, providerAddress string, allProviderScores []ProviderSelectionScores, rngValue float64)
	SetBlockedProvider(chainId, apiInterface, providerAddress, providerEndpoint string, isBlocked bool)
	SetProbeOutlierBlock(chainId, apiInterface, providerAddress string, reportedBlock, floor, threshold int64)
	SetMajorityBaselineConsensusFailure(chainId, apiInterface string, validProbeCount int)
	// seenBlock/latestSyncData unification observability — see unification-demo-env-setup.md §7
	SetAlignLatestBlockOutcome(chainId, apiInterface, outcome string)
	SetLatestBlockOutlierRejected(chainId string)
	SetChainStateLatestBlock(chainId string, block int64)
	SetMajorityBaselineGauge(chainId, apiInterface string, value int64)
	SetAlignLatestBlockGap(chainId, apiInterface string, gap int64)
	SetSyncScoringOutlierSkipped(chainId, providerAddress, source string)
	SetSharedStatePropagation(chainId string)
	SetQOSMetrics(chainId string, apiInterface string, providerAddress string, providerEndpoint string, qos *pairingtypes.QualityOfServiceReport, reputation *pairingtypes.QualityOfServiceReport, latestBlock int64, relays uint64, relayLatency time.Duration, sessionSuccessful bool)

	// --- Session (ConsumerSessionManager) ---
	ResetSessionRelatedMetrics()
	ResetBlockedProvidersMetrics(chainId, apiInterface string, providers map[string]string)

	// --- WebSocket (DirectWSSubscriptionManager) ---
	SetWsSubscriptionRequestMetric(chainId string, apiInterface string)
	SetFailedWsSubscriptionRequestMetric(chainId string, apiInterface string)
	SetWebSocketConnectionActive(chainId string, apiInterface string, add bool)

	// --- Misc (RPCConsumerLogs / rpcsmartrouter.go) ---
	SetVersion(version string)
	StartSelectionStatsUpdater(ctx context.Context, updateInterval time.Duration)
}
