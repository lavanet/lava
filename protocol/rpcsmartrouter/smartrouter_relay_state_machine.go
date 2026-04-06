package rpcsmartrouter

import (
	"context"

	"github.com/lavanet/lava/v5/protocol/chainlib"
	"github.com/lavanet/lava/v5/protocol/lavasession"
	"github.com/lavanet/lava/v5/protocol/metrics"
	"github.com/lavanet/lava/v5/protocol/relaycore"
	"github.com/lavanet/lava/v5/protocol/relaypolicy"
)

// Using interfaces from relaycore
type (
	RelayStateMachine          = relaycore.RelayStateMachine
	ResultsCheckerInf          = relaycore.ResultsCheckerInf
	RelayStateSendInstructions = relaycore.RelayStateSendInstructions
)

// SmartRouterRelayStateMachine is an alias for the unified state machine type.
// Kept for backward compatibility with existing tests.
type SmartRouterRelayStateMachine = relaycore.UnifiedRelayStateMachine

// SmartRouterRelaySender is kept as a type alias for the unified interface
// so that existing test mocks continue to compile.
type SmartRouterRelaySender = relaycore.RelaySenderInf

type tickerMetricSetterInf = relaycore.TickerMetricSetterInf

// SmartRouterStateMachineConfig returns the StateMachineConfig for SmartRouter mode
func SmartRouterStateMachineConfig() relaycore.StateMachineConfig {
	return relaycore.StateMachineConfig{
		EnableCircuitBreaker:         true,
		CircuitBreakerThreshold:      2,
		EnableTimeoutPriority:        true,
		MaxRetries:                   MaximumNumberOfTickerRelayRetries,
		SendRelayAttempts:            SendRelayAttempts,
	}
}

// SmartRouterPolicyConfig returns the PolicyConfig for SmartRouter mode
func SmartRouterPolicyConfig() relaypolicy.PolicyConfig {
	return relaypolicy.PolicyConfig{
		MaxRetries:              MaximumNumberOfTickerRelayRetries,
		RelayRetryLimit:         relaycore.RelayRetryLimit,
		DisableBatchRetry:       relaycore.DisableBatchRequestRetry,
		EnableCircuitBreaker:    true,
		CircuitBreakerThreshold: 2,
		SendRelayAttempts:       SendRelayAttempts,
	}
}

// NewSmartRouterRelayStateMachine creates a SmartRouter-mode unified state machine.
// This is a convenience wrapper that passes the SmartRouter config and policy.
func NewSmartRouterRelayStateMachine(
	ctx context.Context,
	usedProviders *lavasession.UsedProviders,
	relaySender SmartRouterRelaySender,
	protocolMessage chainlib.ProtocolMessage,
	analytics *metrics.RelayMetrics,
	debugRelays bool,
	tickerMetricSetter tickerMetricSetterInf,
) (RelayStateMachine, error) {
	policy := relaypolicy.NewPolicy(SmartRouterPolicyConfig())
	return relaycore.NewUnifiedRelayStateMachine(
		ctx,
		usedProviders,
		relaySender,
		protocolMessage,
		analytics,
		debugRelays,
		tickerMetricSetter,
		SmartRouterStateMachineConfig(),
		policy,
	)
}
