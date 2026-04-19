package relaypolicy

import "github.com/lavanet/lava/v5/protocol/common"

// Re-export eligibility types from common for convenience.
type (
	EligibilityAction = common.EligibilityAction
	EligibilityResult = common.EligibilityResult
)

const (
	MarkUnwanted = common.EligibilityMarkUnwanted
	AllowRetry   = common.EligibilityAllowRetry
)

// DecideEligibility is the canonical eligibility decision function.
// Inject this into UsedProviders via SetEligibilityFunc to route
// eligibility decisions through the policy package.
func DecideEligibility(isUnsupportedMethod, isSyncLoss, isFirstSyncLoss bool) EligibilityResult {
	return common.DecideEligibility(isUnsupportedMethod, isSyncLoss, isFirstSyncLoss)
}
