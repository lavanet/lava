package common

import (
	"bytes"
	"strings"

	sdkerrors "cosmossdk.io/errors"
)

var (
	ContextDeadlineExceededError                = sdkerrors.New("ContextDeadlineExceeded Error", 300, "context deadline exceeded")
	StatusCodeError504                          = sdkerrors.New("Disallowed StatusCode Error", 504, "Disallowed status code error (504)")
	StatusCodeError429                          = sdkerrors.New("Disallowed StatusCode Error", 429, "Disallowed status code error (429)")
	StatusCodeErrorStrict                       = sdkerrors.New("Disallowed StatusCode Error", 800, "Disallowed status code error (800)")
	APINotSupportedError                        = sdkerrors.New("APINotSupported Error", 900, "api not supported")
	SubscriptionNotFoundError                   = sdkerrors.New("SubscriptionNotFoundError Error", 901, "subscription not found")
	ProviderFinalizationDataAccountabilityError = sdkerrors.New("ProviderFinalizationDataAccountability Error", 3365, "provider returned invalid finalization data, with accountability")
)


// Error pattern constants for unsupported method detection
const (
	// JSON-RPC error patterns
	JSONRPCMethodNotFound     = "method not found"
	JSONRPCMethodNotSupported = "method not supported"
	JSONRPCUnknownMethod      = "unknown method"
	JSONRPCMethodDoesNotExist = "method does not exist"
	JSONRPCInvalidMethod      = "invalid method"
	JSONRPCErrorCode          = "-32601" // JSON-RPC 2.0 method not found error code

	// REST API error patterns
	RESTEndpointNotFound = "endpoint not found"
	RESTRouteNotFound    = "route not found"
	RESTPathNotFound     = "path not found"
	RESTMethodNotAllowed = "method not allowed"

	// gRPC error patterns
	GRPCMethodNotImplemented = "method not implemented"
	GRPCUnimplemented        = "unimplemented"
	GRPCNotImplemented       = "not implemented"
	GRPCServiceNotFound      = "service not found"

	// HTTP status codes for unsupported endpoints
	HTTPStatusNotFound         = 404
	HTTPStatusMethodNotAllowed = 405

	// JSON-RPC error code for method not found
	JSONRPCMethodNotFoundCode = -32601
)

// IsUnsupportedMethodMessage checks if an error message indicates an unsupported method
// This performs basic string pattern matching on error messages across all supported protocols.
// It is the single source of truth for pattern matching to ensure consistency across the codebase.
//
// For more comprehensive checks including HTTP status codes and gRPC status codes,
// use chainlib.IsUnsupportedMethodError which wraps this function with additional protocol-specific checks.
//
// Returns true if the error message contains any known unsupported method pattern.
func IsUnsupportedMethodMessage(errorMessage string) bool {
	if errorMessage == "" {
		return false
	}

	errorMsg := strings.ToLower(errorMessage)

	// Check all patterns
	patterns := []string{
		JSONRPCMethodNotFound,
		JSONRPCMethodNotSupported,
		JSONRPCUnknownMethod,
		JSONRPCMethodDoesNotExist,
		JSONRPCInvalidMethod,
		RESTEndpointNotFound,
		RESTRouteNotFound,
		RESTPathNotFound,
		RESTMethodNotAllowed,
		GRPCMethodNotImplemented,
		GRPCUnimplemented,
		GRPCNotImplemented,
		GRPCServiceNotFound,
		JSONRPCErrorCode,
	}

	for _, pattern := range patterns {
		if strings.Contains(errorMsg, pattern) {
			return true
		}
	}

	return false
}

func IsUnsupportedMethodErrorMessageBytes(errorMessage []byte) bool {
	errorMsgLower := bytes.ToLower(errorMessage)


// Pre-computed byte patterns for efficient matching (initialized once at package load)
// These are the lowercase versions of the pattern constants above
var unsupportedMethodPatternBytes = [][]byte{
	// JSON-RPC patterns (most common, ordered by frequency)
	[]byte(JSONRPCMethodNotFound),     // "method not found"
	[]byte(JSONRPCMethodNotSupported), // "method not supported"
	[]byte(JSONRPCUnknownMethod),      // "unknown method"
	[]byte(JSONRPCMethodDoesNotExist), // "method does not exist"
	[]byte(JSONRPCInvalidMethod),      // "invalid method"
	[]byte(JSONRPCErrorCode),          // "-32601"
	// REST patterns
	[]byte(RESTEndpointNotFound), // "endpoint not found"
	[]byte(RESTRouteNotFound),    // "route not found"
	[]byte(RESTPathNotFound),     // "path not found"
	[]byte(RESTMethodNotAllowed), // "method not allowed"
	// gRPC patterns
	[]byte(GRPCMethodNotImplemented), // "method not implemented"
	[]byte(GRPCUnimplemented),        // "unimplemented"
	[]byte(GRPCNotImplemented),       // "not implemented"
	[]byte(GRPCServiceNotFound),      // "service not found"
}

	for _, pattern := range unsupportedMethodPatternBytes{
		if bytes.Contains(errorMsgLower, pattern) {
			return true
		}
	}
	return false
}