package relaypolicy

import "github.com/lavanet/lava/v5/protocol/relaycore"

// Re-export relaycore types for convenience
type (
	Action          = relaycore.Action
	SendResult      = relaycore.SendResult
	ArchiveAction   = relaycore.ArchiveAction
	MutationOutput  = relaycore.MutationOutput
	DecisionOutput  = relaycore.DecisionOutput
	DecisionInput   = relaycore.DecisionInput
	ResultsSummary  = relaycore.ResultsSummary
)

// Constants re-exported from relaycore
const (
	Retry = relaycore.ActionRetry
	Stop  = relaycore.ActionStop

	SendSuccess = relaycore.SendSuccess
	SendRetry   = relaycore.SendRetry
	SendStop    = relaycore.SendStop

	NoChange      = relaycore.ArchiveNoChange
	AddArchive    = relaycore.ArchiveAdd
	RemoveArchive = relaycore.ArchiveRemove
)

// EligibilityAction represents the provider eligibility decision.
type EligibilityAction int

const (
	MarkUnwanted EligibilityAction = iota
	AllowRetry
)

// PolicyConfig configures the policy engine. Consumer and SmartRouter may use different values.
type PolicyConfig struct {
	MaxRetries              int  // hard ceiling on retry attempts
	RelayRetryLimit         int  // error tolerance (total errors before giving up)
	DisableBatchRetry       bool // whether batch requests can be retried
	EnableCircuitBreaker    bool // provider exhaustion detection (SmartRouter only)
	CircuitBreakerThreshold int  // consecutive pairing errors before tripping
	SendRelayAttempts       int  // consecutive batch errors before giving up
}

// EligibilityResult is returned by DecideEligibility.
type EligibilityResult struct {
	Action EligibilityAction
}

// ErrorClassification is returned by ClassifyError.
type ErrorClassification struct {
	IsUnsupportedMethod  bool
	IsSolanaNonRetryable bool
	IsRetryable          bool
}
