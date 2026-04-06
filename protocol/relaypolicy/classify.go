package relaypolicy

import (
	"net/http"
	"strings"

	"github.com/lavanet/lava/v5/protocol/chainlib"
	"github.com/lavanet/lava/v5/protocol/common"
)

// ClassifyNodeError classifies a node error based on the error message, status code,
// and API interface. Called from workers in the same location as the old inline
// unsupported-method classification. Returns a structured classification.
func ClassifyNodeError(errorMessage string, statusCode int, apiInterface string) ErrorClassification {
	isUnsupported := common.IsUnsupportedMethodMessage(errorMessage)

	// Additional check for REST APIs: 404/405 indicate unsupported endpoints
	if !isUnsupported && apiInterface != "" {
		if strings.EqualFold(apiInterface, "rest") {
			if statusCode == http.StatusNotFound || statusCode == http.StatusMethodNotAllowed {
				isUnsupported = true
			}
		}
	}

	return ErrorClassification{
		IsUnsupportedMethod:  isUnsupported,
		IsSolanaNonRetryable: false, // Solana check is done on protocol errors, not node errors
		IsRetryable:          !isUnsupported,
	}
}

// ClassifyProtocolError classifies a protocol-level error. Used for errors that
// never reached the node (connection failures, etc.).
func ClassifyProtocolError(err error) ErrorClassification {
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
