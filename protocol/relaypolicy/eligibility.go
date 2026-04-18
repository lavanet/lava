package relaypolicy

import "github.com/lavanet/lava/v5/protocol/common"

// Re-export eligibility types from common for convenience
type (
	EligibilityAction = common.EligibilityAction
	EligibilityResult = common.EligibilityResult
)

const (
	MarkUnwanted = common.EligibilityMarkUnwanted
	AllowRetry   = common.EligibilityAllowRetry
)

// DecideEligibility delegates to common.DecideEligibility.
// Kept in relaypolicy for API consistency with the policy package.
func DecideEligibility(isUnsupportedMethod bool, isSyncLoss bool, isFirstSyncLoss bool) EligibilityResult {
	return common.DecideEligibility(isUnsupportedMethod, isSyncLoss, isFirstSyncLoss)
}
