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

func (NoOpConsumerMetrics) SetRelayMetrics(*RelayMetrics, error)              {}
func (NoOpConsumerMetrics) SetRelaySentToProviderMetric(string, string)       {}
func (NoOpConsumerMetrics) SetRelaySentByNewBatchTickerMetric(string, string) {}
func (NoOpConsumerMetrics) SetRequestPerProvider(string, string)              {}
func (NoOpConsumerMetrics) SetRelayProcessingLatencyBeforeProvider(time.Duration, string, string) {
}

func (NoOpConsumerMetrics) SetRelayProcessingLatencyAfterProvider(time.Duration, string, string) {
}
func (NoOpConsumerMetrics) SetEndToEndLatency(string, string, time.Duration) {}
func (NoOpConsumerMetrics) SetRelayNodeErrorMetric(string, string)           {}
func (NoOpConsumerMetrics) SetNodeErrorRecoveredSuccessfullyMetric(string, string, string) {
}

func (NoOpConsumerMetrics) SetProtocolErrorRecoveredSuccessfullyMetric(string, string, string) {
}
func (NoOpConsumerMetrics) SetProtocolError(string, string) {}
func (NoOpConsumerMetrics) SetCrossValidationMetric(string, string, string, string, int, int, []string, []string) {
}
func (NoOpConsumerMetrics) UpdateHealthCheckStatus(bool)                          {}
func (NoOpConsumerMetrics) UpdateHealthcheckStatusBreakdown(string, string, bool) {}
func (NoOpConsumerMetrics) SetProviderLiveness(string, string, string, bool)      {}
func (NoOpConsumerMetrics) SetProviderSelected(string, string, []ProviderSelectionScores, float64) {
}
func (NoOpConsumerMetrics) SetBlockedProvider(string, string, string, string, bool) {}
func (NoOpConsumerMetrics) SetQOSMetrics(string, string, string, string, *pairingtypes.QualityOfServiceReport, *pairingtypes.QualityOfServiceReport, int64, uint64, time.Duration, bool) {
}
func (NoOpConsumerMetrics) ResetSessionRelatedMetrics()                                    {}
func (NoOpConsumerMetrics) ResetBlockedProvidersMetrics(string, string, map[string]string) {}
func (NoOpConsumerMetrics) SetWsSubscriptionRequestMetric(string, string)                  {}
func (NoOpConsumerMetrics) SetFailedWsSubscriptionRequestMetric(string, string)            {}
func (NoOpConsumerMetrics) SetDuplicatedWsSubscriptionRequestMetric(string, string)        {}
func (NoOpConsumerMetrics) SetWsSubscriptionDisconnectRequestMetric(string, string, string) {
}
func (NoOpConsumerMetrics) SetWebSocketConnectionActive(string, string, bool)         {}
func (NoOpConsumerMetrics) SetLoLResponse(bool)                                       {}
func (NoOpConsumerMetrics) SetVersion(string)                                         {}
func (NoOpConsumerMetrics) StartSelectionStatsUpdater(context.Context, time.Duration) {}

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
	SetRelaySentToProviderMetric(chainId string, apiInterface string)
	SetRelaySentByNewBatchTickerMetric(chainId string, apiInterface string)
	SetRequestPerProvider(chainId string, providerAddress string)

	// --- Latency (RPCConsumerLogs) ---
	SetRelayProcessingLatencyBeforeProvider(latency time.Duration, chainId string, apiInterface string)
	SetRelayProcessingLatencyAfterProvider(latency time.Duration, chainId string, apiInterface string)
	SetEndToEndLatency(chainId string, apiInterface string, latency time.Duration)

	// --- Errors (RPCConsumerLogs) ---
	SetRelayNodeErrorMetric(chainId string, apiInterface string)
	SetNodeErrorRecoveredSuccessfullyMetric(chainId string, apiInterface string, attempt string)
	SetProtocolErrorRecoveredSuccessfullyMetric(chainId string, apiInterface string, attempt string)
	SetProtocolError(chainId string, providerAddress string)

	// --- Cross-validation (RPCConsumerLogs) ---
	SetCrossValidationMetric(chainId, apiInterface, method, status string, maxParticipants, agreementThreshold int, allProvidersSorted, agreeingProvidersSorted []string)

	// --- Health (RelaysMonitorAggregator) ---
	UpdateHealthCheckStatus(status bool)
	UpdateHealthcheckStatusBreakdown(chainId, apiInterface string, status bool)

	// --- Provider state (ConsumerSessionManager) ---
	SetProviderLiveness(chainId string, providerAddress string, providerEndpoint string, isAlive bool)
	SetProviderSelected(chainId string, providerAddress string, allProviderScores []ProviderSelectionScores, rngValue float64)
	SetBlockedProvider(chainId, apiInterface, providerAddress, providerEndpoint string, isBlocked bool)
	SetQOSMetrics(chainId string, apiInterface string, providerAddress string, providerEndpoint string, qos *pairingtypes.QualityOfServiceReport, reputation *pairingtypes.QualityOfServiceReport, latestBlock int64, relays uint64, relayLatency time.Duration, sessionSuccessful bool)

	// --- Session (ConsumerSessionManager) ---
	ResetSessionRelatedMetrics()
	ResetBlockedProvidersMetrics(chainId, apiInterface string, providers map[string]string)

	// --- WebSocket (DirectWSSubscriptionManager) ---
	SetWsSubscriptionRequestMetric(chainId string, apiInterface string)
	SetFailedWsSubscriptionRequestMetric(chainId string, apiInterface string)
	SetDuplicatedWsSubscriptionRequestMetric(chainId string, apiInterface string)
	SetWsSubscriptionDisconnectRequestMetric(chainId string, apiInterface string, disconnectReason string)
	SetWebSocketConnectionActive(chainId string, apiInterface string, add bool)

	// --- Misc (RPCConsumerLogs / rpcsmartrouter.go) ---
	SetLoLResponse(success bool)
	SetVersion(version string)
	StartSelectionStatsUpdater(ctx context.Context, updateInterval time.Duration)
}
