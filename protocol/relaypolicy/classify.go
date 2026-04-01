package relaypolicy

import (
	"github.com/lavanet/lava/v5/protocol/chainlib"
)

// ClassifyError consolidates DP#6/DP#7/DP#8 into a single classification function.
// Called from workers in the same location as today. Returns a structured classification
// instead of scattered boolean checks.
func ClassifyError(err error) ErrorClassification {
	if err == nil {
		return ErrorClassification{IsRetryable: true}
	}

	isUnsupported := chainlib.IsUnsupportedMethodError(err) || chainlib.IsUnsupportedMethodErrorType(err)
	isSolanaNonRetryable := chainlib.IsSolanaNonRetryableError(err) || chainlib.IsSolanaNonRetryableErrorType(err)

	return ErrorClassification{
		IsUnsupportedMethod:  isUnsupported,
		IsSolanaNonRetryable: isSolanaNonRetryable,
		IsRetryable:          !isUnsupported && !isSolanaNonRetryable,
	}
}
