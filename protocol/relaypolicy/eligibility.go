package relaypolicy

import (
	"github.com/lavanet/lava/v5/protocol/common"
	"github.com/lavanet/lava/v5/protocol/lavasession"
)

// DecideEligibility consolidates DP#9 — provider eligibility logic.
// Called from workers in the same location as today. The worker executes the result
// (updates UsedProviders) for timing reasons.
func DecideEligibility(err error, isFirstSyncLoss bool) EligibilityResult {
	if err == nil {
		return EligibilityResult{Action: MarkUnwanted}
	}

	// Never retry unsupported method errors
	if common.IsUnsupportedMethodMessage(err.Error()) {
		return EligibilityResult{Action: MarkUnwanted}
	}

	// Allow retry for first sync loss
	if lavasession.IsSessionSyncLoss(err) && isFirstSyncLoss {
		return EligibilityResult{Action: AllowRetry}
	}

	return EligibilityResult{Action: MarkUnwanted}
}
