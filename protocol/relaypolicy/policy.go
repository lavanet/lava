package relaypolicy

import "github.com/lavanet/lava/v5/protocol/relaycore"

// Verify Policy implements RelayPolicyInf at compile time.
var _ relaycore.RelayPolicyInf = (*Policy)(nil)

// Policy is the central retry decision engine. Pure Go, no external libraries.
type Policy struct {
	config                   PolicyConfig
	consecutiveBatchErrors   int
	consecutivePairingErrors int
}

func NewPolicy(config PolicyConfig) *Policy {
	return &Policy{config: config}
}

// Decide makes all post-relay retry decisions. Called from the state machine's
// gotResults and ticker.C cases. Replaces DP#1, DP#2, DP#3, DP#5, and archive mutation.
func (p *Policy) Decide(input DecisionInput) DecisionOutput {
	// 1. MODE CHECKS
	if input.Selection == relaycore.CrossValidation {
		return DecisionOutput{Action: Stop, Reason: "CrossValidation"}
	}
	if input.Selection == relaycore.Stateful {
		return DecisionOutput{Action: Stop, Reason: "Stateful"}
	}

	// 2. PERMANENT FAILURE CHECKS — non-retryable user-facing errors
	if input.Summary.HasUnsupportedMethod {
		return DecisionOutput{Action: Stop, Reason: "UnsupportedMethod"}
	}
	if input.Summary.HasUserError {
		return DecisionOutput{Action: Stop, Reason: "UserError"}
	}
	if input.Summary.HasPermanentProtocolError {
		return DecisionOutput{Action: Stop, Reason: "PermanentProtocolError"}
	}

	// 3. LIMIT CHECKS
	if input.AttemptNumber >= p.config.MaxRetries {
		return DecisionOutput{Action: Stop, Reason: "MaxRetriesReached"}
	}
	if input.IsBatch && p.config.DisableBatchRetry {
		return DecisionOutput{Action: Stop, Reason: "BatchDisabled"}
	}

	// 4. RESULT CHECKS — epoch mismatch always retries
	if input.Summary.HasEpochMismatch && input.Summary.SuccessCount == 0 {
		return DecisionOutput{Action: Retry, Reason: "EpochMismatch"}
	}

	// 5. HASH ERROR CHECK
	if input.Summary.HashErr != nil {
		return DecisionOutput{Action: Stop, Reason: "HashComputationFailed"}
	}

	totalErrors := input.Summary.NodeErrors + input.Summary.SpecialNodeErrors + input.Summary.ProtocolErrors
	if totalErrors > p.config.RelayRetryLimit {
		return DecisionOutput{Action: Stop, Reason: "ErrorToleranceExceeded"}
	}

	// 6. ARCHIVE MUTATION
	mutation := p.decideMutation(input)

	// 7. DEFAULT: RETRY
	return DecisionOutput{Action: Retry, Mutation: mutation, Reason: "Default"}
}

// decideMutation determines archive and cache side effects for the current attempt.
func (p *Policy) decideMutation(input DecisionInput) MutationOutput {
	if input.ArchiveStatus == nil {
		return MutationOutput{}
	}

	isUpgraded := input.ArchiveStatus.IsUpgraded()
	isArchive := input.ArchiveStatus.IsArchive()

	// If upgraded and 2+ node errors, cache hashes and remove archive
	if isUpgraded && input.NodeErrors >= 2 {
		return MutationOutput{
			ArchiveAction: RemoveArchive,
			CacheHashes:   true,
		}
	}

	// Add archive on first retry (attempt 1)
	if !isArchive && input.AttemptNumber == 1 {
		return MutationOutput{ArchiveAction: AddArchive}
	}

	// Remove archive on second retry (attempt 2) if upgraded
	if isUpgraded && input.AttemptNumber == 2 {
		return MutationOutput{ArchiveAction: RemoveArchive}
	}

	return MutationOutput{}
}

// OnSendRelayResult handles pre-relay send decisions. Called from the state machine's
// batchUpdate case. Merges DP#11 (circuit breaker) and DP#12 (batch send retry).
func (p *Policy) OnSendRelayResult(err error, isPairingListEmpty bool) SendResult {
	if err == nil {
		p.consecutiveBatchErrors = 0
		p.consecutivePairingErrors = 0
		return SendSuccess
	}

	p.consecutiveBatchErrors++

	// Circuit breaker
	if p.config.EnableCircuitBreaker && isPairingListEmpty {
		p.consecutivePairingErrors++
		if p.consecutivePairingErrors >= p.config.CircuitBreakerThreshold {
			return SendStop
		}
	} else if p.config.EnableCircuitBreaker {
		p.consecutivePairingErrors = 0
	}

	// Batch send retry limit
	if p.consecutiveBatchErrors > p.config.SendRelayAttempts {
		return SendStop
	}

	return SendRetry
}

// GetConsecutiveBatchErrors returns the current consecutive batch error count (for logging).
func (p *Policy) GetConsecutiveBatchErrors() int {
	return p.consecutiveBatchErrors
}
