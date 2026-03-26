package chainlib

import (
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

type UnsupportedMethodError struct {
	originalError error
	methodName    string
}

func (e *UnsupportedMethodError) Error() string {
	if e.methodName != "" {
		return fmt.Sprintf("unsupported method %q: %v", e.methodName, e.originalError)
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

// SolanaNonRetryableError represents a Solana error that should not be retried.
// Currently covers error code -32009 ("missing in long-term storage") which indicates
// the slot data is permanently unavailable.
// Note: -32007 (ledger jump) IS retryable as another provider may have the data.
type SolanaNonRetryableError struct {
	originalError error
}

func (e *SolanaNonRetryableError) Error() string {
	return fmt.Sprintf("solana non-retryable error: %v", e.originalError)
}

func (e *SolanaNonRetryableError) Unwrap() error {
	return e.originalError
}

// NewSolanaNonRetryableError creates a new SolanaNonRetryableError
func NewSolanaNonRetryableError(originalError error) *SolanaNonRetryableError {
	return &SolanaNonRetryableError{
		originalError: originalError,
	}
}

// ClassifyNodeError classifies a node error into a LavaError using the error registry.
// It extracts error codes and messages from JSON-RPC, gRPC, and HTTP errors,
// then delegates to common.ClassifyError for two-tier classification.
//
// Parameters:
//   - nodeError: the error from the node
//   - chainFamily: the chain family for Tier 2 lookups (use -1 if unknown)
//   - transport: the transport type for Tier 1 generic matcher partitioning
func ClassifyNodeError(nodeError error, chainFamily common.ChainFamily, transport common.TransportType) *common.LavaError {
	if nodeError == nil {
		return nil
	}

	errorCode := 0
	errorMessage := nodeError.Error()

	// Extract error code from HTTP-wrapped JSON-RPC errors
	if jsonMsg := TryRecoverNodeErrorFromClientError(nodeError); jsonMsg != nil && jsonMsg.Error != nil {
		errorCode = jsonMsg.Error.Code
		if jsonMsg.Error.Message != "" {
			errorMessage = jsonMsg.Error.Message
		}
	}

	// Extract gRPC status code
	if st, ok := status.FromError(nodeError); ok {
		errorCode = int(st.Code())
		if st.Message() != "" {
			errorMessage = st.Message()
		}
	}

	// Extract HTTP status code
	if httpError, ok := nodeError.(rpcclient.HTTPError); ok {
		errorCode = httpError.StatusCode
	}

	return common.ClassifyError(nil, chainFamily, transport, errorCode, errorMessage)
}

// IsUnsupportedMethodError checks if an error indicates an unsupported method.
// Uses the error registry's SubCategory classification for a unified check across
// all transports (JSON-RPC, REST, gRPC) and all pattern types (codes, messages).
func IsUnsupportedMethodError(nodeError error) bool {
	if nodeError == nil {
		return false
	}

	// Fast path: check the error message patterns directly (covers most cases)
	if common.IsUnsupportedMethodMessage(nodeError.Error()) {
		return true
	}

	// Classify using the registry — checks codes, messages, and HTTP/gRPC status
	classified := ClassifyNodeError(nodeError, -1, common.TransportJsonRPC)
	if classified != nil && classified.SubCategory.IsUnsupportedMethod() {
		return true
	}

	// Also try REST and gRPC transports for HTTP status codes and gRPC codes
	classified = ClassifyNodeError(nodeError, -1, common.TransportREST)
	if classified != nil && classified.SubCategory.IsUnsupportedMethod() {
		return true
	}
	classified = ClassifyNodeError(nodeError, -1, common.TransportGRPC)
	if classified != nil && classified.SubCategory.IsUnsupportedMethod() {
		return true
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

// IsSolanaNonRetryableError checks if an error indicates a Solana error that should not be retried.
// This checks:
// - JSON-RPC error codes: -32602 (invalid params), -32009 (missing in long-term storage)
// - Error message pattern matching (via common.IsSolanaNonRetryableError)
// Note: -32007 (ledger jump) IS retryable as another provider may have the data.
func IsSolanaNonRetryableError(nodeError error) bool {
	if nodeError == nil {
		return false
	}

	// First check the error message patterns
	if common.IsSolanaNonRetryableError(nodeError.Error()) {
		return true
	}

	// Try to recover JSON-RPC error from HTTP error
	if jsonMsg := TryRecoverNodeErrorFromClientError(nodeError); jsonMsg != nil && jsonMsg.Error != nil {
		// Check for non-retryable JSON-RPC error codes
		// -32602: Invalid params (e.g., invalid slot number)
		// -32009: Missing in long-term storage (slot permanently unavailable)
		if jsonMsg.Error.Code == common.JSONRPCInvalidParamsCode ||
			jsonMsg.Error.Code == common.SolanaMissingInLongTermStorageCode {
			return true
		}
		// Check error message patterns in JSON-RPC error
		if jsonMsg.Error.Message != "" {
			return common.IsSolanaNonRetryableError(jsonMsg.Error.Message)
		}
	}

	return false
}

// IsSolanaNonRetryableErrorType checks if an error is specifically a SolanaNonRetryableError type
func IsSolanaNonRetryableErrorType(err error) bool {
	if err == nil {
		return false
	}
	var solanaNonRetryableError *SolanaNonRetryableError
	return errors.As(err, &solanaNonRetryableError)
}

// ShouldRetryError determines if an error should trigger retry attempts.
// Uses the error registry's Retryable field and SubCategory for classification,
// with fallback to legacy type checks for wrapped errors.
func ShouldRetryError(err error) bool {
	if err == nil {
		return false
	}

	// Check wrapped error types (these wrap the original error with retry intent)
	if IsUnsupportedMethodErrorType(err) {
		return false
	}
	if IsSolanaNonRetryableErrorType(err) {
		return false
	}

	// Classify using the registry
	classified := ClassifyNodeError(err, -1, common.TransportJsonRPC)
	if classified != nil && classified != common.LavaErrorUnknown {
		// Unsupported methods are never retried regardless of Retryable flag
		if classified.SubCategory.IsUnsupportedMethod() {
			return false
		}
		return classified.Retryable
	}

	// Legacy fallback for errors not classified by the registry
	if IsUnsupportedMethodError(err) {
		return false
	}
	if IsSolanaNonRetryableError(err) {
		return false
	}

	// Unknown errors — allow retry
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

	// Allow empty/missing response ID for non-standard JSON-RPC implementations (e.g., XRPL/Ripple)
	// Some chains don't follow JSON-RPC 2.0 spec strictly and omit the ID field in responses
	//
	// TODO: In the future, add a spec-level parameter (e.g., in api_collection or as an add-on flag)
	// to explicitly declare when a chain allows non-standard JSON-RPC responses. This validation
	// function should check that parameter instead of auto-detecting missing IDs.
	// Example: "allow_missing_response_id": true in the spec's collection_data
	// Not implemented now because it requires a software upgrade on-chain (spec schema change)
	// and governance approval to update existing specs.
	if len(replyMsgID) == 0 || string(replyMsgID) == "null" || string(replyMsgID) == "[]" {
		return nil // Skip ID validation when response has no ID
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

	if IsSolanaNonRetryableError(nodeError) {
		return &SolanaNonRetryableError{originalError: nodeError}
	}

	return rne.handleGenericErrors(ctx, nodeError)
}

type JsonRPCErrorHandler struct{ genericErrorHandler }

func (jeh *JsonRPCErrorHandler) HandleNodeError(ctx context.Context, nodeError error) error {
	if IsUnsupportedMethodError(nodeError) {
		return &UnsupportedMethodError{originalError: nodeError}
	}

	if IsSolanaNonRetryableError(nodeError) {
		return &SolanaNonRetryableError{originalError: nodeError}
	}

	return jeh.handleGenericErrors(ctx, nodeError)
}

type TendermintRPCErrorHandler struct{ genericErrorHandler }

func (tendermintErrorHandler *TendermintRPCErrorHandler) HandleNodeError(ctx context.Context, nodeError error) error {
	if IsUnsupportedMethodError(nodeError) {
		return &UnsupportedMethodError{originalError: nodeError}
	}

	if IsSolanaNonRetryableError(nodeError) {
		return &SolanaNonRetryableError{originalError: nodeError}
	}

	return tendermintErrorHandler.handleGenericErrors(ctx, nodeError)
}

type GRPCErrorHandler struct{ genericErrorHandler }

func (geh *GRPCErrorHandler) HandleNodeError(ctx context.Context, nodeError error) error {
	if IsUnsupportedMethodError(nodeError) {
		return &UnsupportedMethodError{originalError: nodeError}
	}

	if IsSolanaNonRetryableError(nodeError) {
		return &SolanaNonRetryableError{originalError: nodeError}
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
