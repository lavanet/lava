package relaypolicy

import (
	"github.com/lavanet/lava/v5/protocol/common"
)

// ClassifyNodeError classifies a node error using the structured error registry.
// Wraps common.ClassifyNodeErrorForRetry with chain family and transport resolution.
// Called from the consumer worker to set IsNonRetryable, IsUnsupportedMethod,
// and IsUserError flags on the relay result before storage.
func ClassifyNodeError(chainID string, apiInterface string, statusCode int, errorMessage string, replyData []byte) ErrorClassification {
	family := common.ChainFamilyUnknown
	if f, ok := common.GetChainFamily(chainID); ok {
		family = f
	}
	transport := common.ApiInterfaceToTransport(apiInterface)

	// JSON-RPC node errors return HTTP 200 with the real error code in the
	// response body. Feed the body code to the registry so code-based matchers hit.
	errorCode := statusCode
	if transport == common.TransportJsonRPC && replyData != nil {
		if jsonrpcCode := common.ExtractJSONRPCErrorCode(replyData); jsonrpcCode != 0 {
			errorCode = jsonrpcCode
		}
	}

	c := common.ClassifyNodeErrorForRetry(family, transport, errorCode, errorMessage)
	return ErrorClassification{
		IsNonRetryable:      c.IsNonRetryable,
		IsUnsupportedMethod: c.IsUnsupportedMethod,
		IsUserError:         c.IsUserError,
	}
}
