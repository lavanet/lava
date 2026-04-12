package relaypolicy

import (
	"net/http"
	"strings"

	"github.com/lavanet/lava/v5/protocol/common"
)

// ClassifyNodeError classifies a node error based on the error message, status code,
// and API interface. Called from the consumer worker to set the IsUnsupportedMethod
// flag on the relay result before storage.
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
		IsUnsupportedMethod: isUnsupported,
	}
}
