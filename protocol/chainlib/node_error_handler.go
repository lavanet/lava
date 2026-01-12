package chainlib

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/url"
	"regexp"
	"strings"

	"github.com/goccy/go-json"

	"github.com/lavanet/lava/v5/protocol/chainlib/chainproxy/rpcInterfaceMessages"
	"github.com/lavanet/lava/v5/protocol/chainlib/chainproxy/rpcclient"
	"github.com/lavanet/lava/v5/protocol/common"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/itchyny/gojq"
	"github.com/lavanet/lava/v5/utils"
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

type UnsupportedMethodError struct {
	originalError error
	methodName    string
}

func (e *UnsupportedMethodError) Error() string {
	if e.methodName != "" {
		return fmt.Sprintf("unsupported method '%s': %v", e.methodName, e.originalError)
	}
	return fmt.Sprintf("unsupported method: %v", e.originalError)
}

func (e *UnsupportedMethodError) Unwrap() error {
	return e.originalError
}

// WithMethod sets the method name for the error
func (e *UnsupportedMethodError) WithMethod(method string) *UnsupportedMethodError {
	e.methodName = method
	return e
}

// GetMethodName returns the method name associated with this error
func (e *UnsupportedMethodError) GetMethodName() string {
	return e.methodName
}

// NewUnsupportedMethodError creates a new UnsupportedMethodError with optional method name
func NewUnsupportedMethodError(originalError error, methodName string) *UnsupportedMethodError {
	return &UnsupportedMethodError{
		originalError: originalError,
		methodName:    methodName,
	}
}

// GetUnsupportedMethodPatterns returns all error patterns that indicate an unsupported method
// This is useful for documentation and testing purposes
func GetUnsupportedMethodPatterns() map[string][]string {
	return map[string][]string{
		"json-rpc": {
			JSONRPCMethodNotFound,
			JSONRPCMethodNotSupported,
			JSONRPCUnknownMethod,
			JSONRPCMethodDoesNotExist,
			JSONRPCInvalidMethod,
			JSONRPCErrorCode,
		},
		"rest": {
			RESTEndpointNotFound,
			RESTRouteNotFound,
			RESTPathNotFound,
			RESTMethodNotAllowed,
		},
		"grpc": {
			GRPCMethodNotImplemented,
			GRPCUnimplemented,
			GRPCNotImplemented,
			GRPCServiceNotFound,
		},
	}
}

// IsUnsupportedMethodErrorMessage checks if an error message indicates an unsupported method
// This is a convenience function that accepts a string directly
func IsUnsupportedMethodErrorMessage(errorMessage string) bool {
	return IsUnsupportedMethodErrorMessageBytes([]byte(errorMessage))
}

// IsUnsupportedMethodErrorMessageBytes checks if an error message (as bytes) indicates an unsupported method.
// This is more efficient than IsUnsupportedMethodErrorMessage when working with []byte data
// as it avoids string conversions and uses pre-computed byte patterns with a single-pass lowercase conversion.
func IsUnsupportedMethodErrorMessageBytes(errorMessage []byte) bool {
	// Convert to lowercase once (single O(n) pass)
	errorMsgLower := bytes.ToLower(errorMessage)

	// Check all patterns with early exit on first match
	for _, pattern := range unsupportedMethodPatternBytes {
		if bytes.Contains(errorMsgLower, pattern) {
			return true
		}
	}

	return false
}

// IsUnsupportedMethodError checks if an error indicates an unsupported method
func IsUnsupportedMethodError(nodeError error) bool {
	if nodeError == nil {
		return false
	}

	// First check the error message patterns
	if IsUnsupportedMethodErrorMessage(nodeError.Error()) {
		return true
	}

	// Check for HTTP status codes that indicate unsupported endpoints
	if httpError, ok := nodeError.(rpcclient.HTTPError); ok {
		return httpError.StatusCode == HTTPStatusNotFound || httpError.StatusCode == HTTPStatusMethodNotAllowed
	}

	// Check for gRPC status codes
	if st, ok := status.FromError(nodeError); ok {
		// Check for both Unimplemented and Unknown codes that might indicate unsupported methods
		if st.Code() == codes.Unimplemented {
			return true
		}
		// Also check Unknown code with unsupported method message
		if st.Code() == codes.Unknown && IsUnsupportedMethodErrorMessage(st.Message()) {
			return true
		}
	}

	// Try to recover JSON-RPC error from HTTP error
	if jsonMsg := TryRecoverNodeErrorFromClientError(nodeError); jsonMsg != nil && jsonMsg.Error != nil {
		// JSON-RPC error code -32601 is "Method not found"
		if jsonMsg.Error.Code == JSONRPCMethodNotFoundCode {
			return true
		}
		// Check error message patterns in JSON-RPC error
		if jsonMsg.Error.Message != "" {
			return IsUnsupportedMethodErrorMessage(jsonMsg.Error.Message)
		}
	}

	return false
}

// IsUnsupportedMethodErrorType checks if an error is specifically an UnsupportedMethodError type
func IsUnsupportedMethodErrorType(err error) bool {
	if err == nil {
		return false
	}
	var unsupportedMethodError *UnsupportedMethodError
	return errors.As(err, &unsupportedMethodError)
}

// ShouldRetryError determines if an error should trigger retry attempts
// Returns false for unsupported method errors to prevent unnecessary retries
func ShouldRetryError(err error) bool {
	if err == nil {
		return false
	}

	// Never retry unsupported method errors
	if IsUnsupportedMethodErrorType(err) {
		return false
	}

	// Never retry if the error message indicates an unsupported method
	if IsUnsupportedMethodError(err) {
		return false
	}

	// For other errors, allow retry logic to decide
	return true
}

type genericErrorHandler struct{}

func (geh *genericErrorHandler) handleConnectionError(err error) error {
	// Generic error message
	genericMsg := "Provider Side Failed Sending Message"

	switch {
	case err == net.ErrWriteToConnected:
		return utils.LavaFormatProduction(genericMsg+", Reason: Write to connected connection", nil)
	case err == net.ErrClosed:
		return utils.LavaFormatProduction(genericMsg+", Reason: Operation on closed connection", nil)
	case err == io.EOF:
		return utils.LavaFormatProduction(genericMsg+", Reason: End of input stream reached", nil)
	case strings.Contains(err.Error(), "http: server gave HTTP response to HTTPS client"):
		return utils.LavaFormatProduction(genericMsg+", Reason: misconfigured http endpoint as https", nil)
	}

	if opErr, ok := err.(*net.OpError); ok {
		switch {
		case opErr.Timeout():
			return utils.LavaFormatProduction(genericMsg+", Reason: Network operation timed out", nil)
		case strings.Contains(opErr.Error(), "connection refused"):
			return utils.LavaFormatProduction(genericMsg+", Reason: Connection refused", nil)
		default:
			// Handle other OpError cases without exposing specific details
			return utils.LavaFormatProduction(genericMsg+", Reason: Network operation error", nil)
		}
	}
	if urlErr, ok := err.(*url.Error); ok {
		switch {
		case urlErr.Timeout():
			return utils.LavaFormatProduction(genericMsg+", Reason: url.Error issue", nil)
		case strings.Contains(urlErr.Error(), "connection refused"):
			return utils.LavaFormatProduction(genericMsg+", Reason: Connection refused", nil)
		}
	}

	if _, ok := err.(*net.DNSError); ok {
		return utils.LavaFormatProduction(genericMsg+", Reason: DNS resolution failed", nil)
	}

	// Mask IP addresses and potential secrets in the error message, and check if any secret was found
	maskedError, foundSecret := maskSensitiveInfo(err.Error())
	if foundSecret {
		// Log or handle the case when a secret was found, if necessary
		utils.LavaFormatProduction(genericMsg+maskedError, nil)
	}
	return nil
}

func maskSensitiveInfo(errMsg string) (string, bool) {
	foundSecret := false

	// Mask IP addresses
	ipRegex := regexp.MustCompile(`\b(?:\d{1,3}\.){3}\d{1,3}\b`)
	if ipRegex.MatchString(errMsg) {
		foundSecret = true
		errMsg = ipRegex.ReplaceAllString(errMsg, "[IP_ADDRESS]")
	}

	return errMsg, foundSecret
}

func (geh *genericErrorHandler) handleGenericErrors(ctx context.Context, nodeError error) error {
	if nodeError == context.DeadlineExceeded || ctx.Err() == context.DeadlineExceeded {
		return utils.LavaFormatProduction("Provider Failed Sending Message", common.ContextDeadlineExceededError)
	}
	retError := geh.handleConnectionError(nodeError)
	if retError != nil {
		// printing the original error as  it was masked for the consumer to not see the private information such as ip address etc..
		utils.LavaFormatProduction("Original Node Error", nodeError)
	}
	return retError
}

func (geh *genericErrorHandler) handleCodeErrors(code codes.Code) error {
	if code == codes.DeadlineExceeded {
		return utils.LavaFormatProduction("Provider Failed Sending Message", common.ContextDeadlineExceededError)
	}
	switch code {
	case codes.PermissionDenied, codes.Canceled, codes.Aborted, codes.DataLoss, codes.Unauthenticated, codes.Unavailable:
		return utils.LavaFormatProduction("Provider Side Failed Sending Message, Reason: "+code.String(), nil)
	}
	return nil
}

func (geh *genericErrorHandler) HandleStatusError(statusCode int, strict bool) error {
	return rpcclient.ValidateStatusCodes(statusCode, strict)
}

func (geh *genericErrorHandler) HandleJSONFormatError(replyData []byte) error {
	_, err := gojq.Parse(string(replyData))
	if err != nil {
		return utils.LavaFormatError("Rest reply is not in JSON format", err, utils.Attribute{Key: "reply.Data", Value: string(replyData)})
	}
	return nil
}

func (geh *genericErrorHandler) ValidateRequestAndResponseIds(nodeMessageID json.RawMessage, replyMsgID json.RawMessage) error {
	reqId, idErr := rpcInterfaceMessages.IdFromRawMessage(nodeMessageID)
	if idErr != nil {
		return fmt.Errorf("failed parsing ID %s", idErr.Error())
	}
	respId, idErr := rpcInterfaceMessages.IdFromRawMessage(replyMsgID)
	if idErr != nil {
		return fmt.Errorf("failed parsing ID %s", idErr.Error())
	}
	if reqId != respId {
		return fmt.Errorf("ID mismatch error")
	}
	return nil
}

func TryRecoverNodeErrorFromClientError(nodeErr error) *rpcclient.JsonrpcMessage {
	// try to parse node error as json message
	httpError, ok := nodeErr.(rpcclient.HTTPError)
	if ok {
		jsonMessage := &rpcclient.JsonrpcMessage{}
		err := json.Unmarshal(httpError.Body, jsonMessage)
		if err == nil {
			utils.LavaFormatDebug("Successfully recovered HTTPError to node message", utils.LogAttr("jsonMessage", jsonMessage))
			return jsonMessage
		}
	}
	return nil
}

type RestErrorHandler struct{ genericErrorHandler }

// Validating if the error is related to the provider connection or not
// returning nil if its not one of the expected connectivity error types
func (rne *RestErrorHandler) HandleNodeError(ctx context.Context, nodeError error) error {
	if IsUnsupportedMethodError(nodeError) {
		return &UnsupportedMethodError{originalError: nodeError}
	}

	return rne.handleGenericErrors(ctx, nodeError)
}

type JsonRPCErrorHandler struct{ genericErrorHandler }

func (jeh *JsonRPCErrorHandler) HandleNodeError(ctx context.Context, nodeError error) error {
	if IsUnsupportedMethodError(nodeError) {
		return &UnsupportedMethodError{originalError: nodeError}
	}

	return jeh.handleGenericErrors(ctx, nodeError)
}

type TendermintRPCErrorHandler struct{ genericErrorHandler }

func (tendermintErrorHandler *TendermintRPCErrorHandler) HandleNodeError(ctx context.Context, nodeError error) error {
	if IsUnsupportedMethodError(nodeError) {
		return &UnsupportedMethodError{originalError: nodeError}
	}

	return tendermintErrorHandler.handleGenericErrors(ctx, nodeError)
}

type GRPCErrorHandler struct{ genericErrorHandler }

func (geh *GRPCErrorHandler) HandleNodeError(ctx context.Context, nodeError error) error {
	if IsUnsupportedMethodError(nodeError) {
		return &UnsupportedMethodError{originalError: nodeError}
	}

	st, ok := status.FromError(nodeError)
	if ok {
		// Get the error message from the gRPC status
		return geh.handleCodeErrors(st.Code())
	}
	return geh.handleGenericErrors(ctx, nodeError)
}

type ErrorHandler interface {
	HandleNodeError(context.Context, error) error
	HandleStatusError(int, bool) error
	HandleJSONFormatError([]byte) error
	ValidateRequestAndResponseIds(json.RawMessage, json.RawMessage) error
}
