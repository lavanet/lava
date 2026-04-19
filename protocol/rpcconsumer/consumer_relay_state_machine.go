package rpcconsumer

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

// ConsumerRelayStateMachine is an alias for the unified state machine type.
// Kept for backward compatibility with existing tests.
type ConsumerRelayStateMachine = relaycore.UnifiedRelayStateMachine

// ConsumerRelaySender is kept as a type alias for the unified interface
// so that existing test mocks continue to compile.
type ConsumerRelaySender = relaycore.RelaySenderInf

// ConsumerStateMachineConfig returns the StateMachineConfig for Consumer mode
func ConsumerStateMachineConfig() relaycore.StateMachineConfig {
	return relaycore.StateMachineConfig{
		EnableCircuitBreaker:    false,
		CircuitBreakerThreshold: 0,
		EnableTimeoutPriority:   false,
		MaxRetries:              MaximumNumberOfTickerRelayRetries,
		SendRelayAttempts:       SendRelayAttempts,
	}
}

// ConsumerPolicyConfig returns the PolicyConfig for Consumer mode
func ConsumerPolicyConfig() relaypolicy.PolicyConfig {
	return relaypolicy.PolicyConfig{
		MaxRetries:              MaximumNumberOfTickerRelayRetries,
		RelayRetryLimit:         relaycore.RelayRetryLimit,
		DisableBatchRetry:       relaycore.DisableBatchRequestRetry,
		EnableCircuitBreaker:    false,
		CircuitBreakerThreshold: 0,
		SendRelayAttempts:       SendRelayAttempts,
	}
}

// NewRelayStateMachine creates a Consumer-mode unified state machine.
// This is a convenience wrapper that passes the Consumer config and policy.
func NewRelayStateMachine(
	ctx context.Context,
	usedProviders *lavasession.UsedProviders,
	relaySender ConsumerRelaySender,
	protocolMessage chainlib.ProtocolMessage,
	analytics *metrics.RelayMetrics,
	debugRelays bool,
) (RelayStateMachine, error) {
	policy := relaypolicy.NewPolicy(ConsumerPolicyConfig())
	return relaycore.NewUnifiedRelayStateMachine(
		ctx,
		usedProviders,
		relaySender,
		protocolMessage,
		analytics,
		debugRelays,
		ConsumerStateMachineConfig(),
		policy,
	)
}
