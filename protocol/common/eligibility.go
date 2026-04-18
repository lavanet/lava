package common

// EligibilityAction represents the provider eligibility decision.
type EligibilityAction int

const (
	EligibilityMarkUnwanted EligibilityAction = iota
	EligibilityAllowRetry
)

// EligibilityResult is returned by DecideEligibility.
type EligibilityResult struct {
	Action EligibilityAction
}

// DecideEligibility makes the provider eligibility decision.
// Pure function — no external dependencies. The caller computes the inputs.
//
//   - isUnsupportedMethod: true if the error is an unsupported method (no provider will succeed)
//   - isSyncLoss: true if the error is a session sync loss
//   - isFirstSyncLoss: true if this provider hasn't had a sync loss before
func DecideEligibility(isUnsupportedMethod bool, isSyncLoss bool, isFirstSyncLoss bool) EligibilityResult {
	if isUnsupportedMethod {
		return EligibilityResult{Action: EligibilityMarkUnwanted}
	}
	if isSyncLoss && isFirstSyncLoss {
		return EligibilityResult{Action: EligibilityAllowRetry}
	}
	return EligibilityResult{Action: EligibilityMarkUnwanted}
}
