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

// NewUnsupportedMethodError creates an error wrapping a LavaError with unsupported method classification.
// The methodName is included in the context for logging.
func NewUnsupportedMethodError(_ error, methodName string) error {
	context := "unsupported method"
	if methodName != "" {
		context = fmt.Sprintf("unsupported method %q", methodName)
	}
	return common.NewLavaError(common.LavaErrorNodeMethodNotFound, context)
}

// NewSolanaNonRetryableError creates an error wrapping a LavaError with non-retryable classification.
func NewSolanaNonRetryableError(err error) error {
	return common.NewLavaError(common.LavaErrorChainSolanaMissingLongTerm, err.Error())
}

// ExtractNodeErrorDetails extracts the numeric error code and canonical message from a node error.
// It handles three error shapes in priority order:
//  1. HTTP-wrapped JSON-RPC body  — extracts .error.code / .error.message
//  2. gRPC status                 — extracts status code and description
//  3. Raw HTTP status code        — extracts the HTTP status integer
//
// Falls back to (0, err.Error()) when none of the above apply.
func ExtractNodeErrorDetails(nodeError error) (errorCode int, errorMessage string) {
	errorCode = 0
	errorMessage = nodeError.Error()

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

	return errorCode, errorMessage
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
	classified, _, _ := ClassifyNodeErrorWithDetails(nodeError, chainFamily, transport)
	return classified
}

// ClassifyNodeErrorWithDetails is like ClassifyNodeError but also returns the numeric
// error code and inner error message extracted from the raw node error. Callers that
// emit structured logs should prefer this so they don't lose the precise code/message
// that classification already computed. Returns (nil, 0, "") when nodeError is nil.
func ClassifyNodeErrorWithDetails(nodeError error, chainFamily common.ChainFamily, transport common.TransportType) (*common.LavaError, int, string) {
	if nodeError == nil {
		return nil, 0, ""
	}
	connError := common.DetectConnectionError(nodeError)
	errorCode, errorMessage := ExtractNodeErrorDetails(nodeError)
	return common.ClassifyError(connError, chainFamily, transport, errorCode, errorMessage), errorCode, errorMessage
}

// IsUnsupportedMethodError checks if an error indicates an unsupported method.
// Uses the error registry's SubCategory classification for a unified check across
// all transports (JSON-RPC, REST, gRPC) and all pattern types (codes, messages).
func IsUnsupportedMethodError(nodeError error) bool {
	if nodeError == nil {
		return false
	}

	// Classify using the registry — checks codes, messages, and HTTP/gRPC status
	for _, transport := range []common.TransportType{common.TransportJsonRPC, common.TransportREST, common.TransportGRPC} {
		classified := ClassifyNodeError(nodeError, -1, transport)
		if classified != nil && classified.SubCategory.IsUnsupportedMethod() {
			return true
		}
	}

	return false
}

// unwrapLavaError extracts the *LavaError from a LavaWrappedError, or returns nil.
func unwrapLavaError(err error) *common.LavaError {
	var wrapped *common.LavaWrappedError
	if errors.As(err, &wrapped) {
		return wrapped.LavaErr
	}
	return nil
}

// ExtractLavaError returns the *LavaError embedded in a LavaWrappedError, or LavaErrorUnknown.
// Use this when an error has already been classified (e.g., returned from handleAndClassify)
// and you need to retrieve its classification for structured logging.
func ExtractLavaError(err error) *common.LavaError {
	if le := unwrapLavaError(err); le != nil {
		return le
	}
	return common.LavaErrorUnknown
}

// IsUnsupportedMethodErrorType checks if an error wraps a LavaError with unsupported method SubCategory.
func IsUnsupportedMethodErrorType(err error) bool {
	if le := unwrapLavaError(err); le != nil {
		return le.SubCategory.IsUnsupportedMethod()
	}
	return false
}

// IsNonRetryableUserFacingErrorType checks if an error wraps a LavaError whose
// subcategory is any "don't retry, don't charge CU" kind — currently
// SubCategoryUnsupportedMethod or SubCategoryUserError. The retry state
// machine uses this to short-circuit both classes uniformly.
func IsNonRetryableUserFacingErrorType(err error) bool {
	if le := unwrapLavaError(err); le != nil {
		return le.SubCategory.IsNonRetryableUserFacing()
	}
	return false
}

// IsSolanaNonRetryableError checks if an error indicates a Solana error that should not be retried.
// Covers -32009 (missing in long-term storage) and -32602 (invalid params).
// Note: -32007 (ledger jump) IS retryable as another provider may have the data.
func IsSolanaNonRetryableError(nodeError error) bool {
	if nodeError == nil {
		return false
	}
	classified := ClassifyNodeError(nodeError, common.ChainFamilySolana, common.TransportJsonRPC)
	switch classified {
	case common.LavaErrorChainSolanaMissingLongTerm, common.LavaErrorUserInvalidParams:
		return true
	}
	return false
}

// IsSolanaNonRetryableErrorType checks if an error wraps a non-retryable LavaError.
func IsSolanaNonRetryableErrorType(err error) bool {
	if le := unwrapLavaError(err); le != nil {
		return !le.Retryable
	}
	return false
}

// ShouldRetryError determines if an error should trigger retry attempts.
// Convenience wrapper that uses default chain family and transport.
// Prefer ShouldRetryErrorWithContext when chain/transport info is available.
func ShouldRetryError(err error) bool {
	return ShouldRetryErrorWithContext(err, -1, common.TransportJsonRPC)
}

// ShouldRetryErrorWithContext determines if an error should trigger retry attempts,
// using chain family and transport for accurate Tier 2 classification.
func ShouldRetryErrorWithContext(err error, chainFamily common.ChainFamily, transport common.TransportType) bool {
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

	// Classify using the registry with chain-specific and transport-specific matchers
	classified := ClassifyNodeError(err, chainFamily, transport)
	if classified != nil && classified != common.LavaErrorUnknown {
		// Unsupported methods are never retried regardless of Retryable flag
		if classified.SubCategory.IsUnsupportedMethod() {
			return false
		}
		return classified.Retryable
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

// handleGenericErrors handles connection-level and deadline errors, masking sensitive info.
// It returns nil for any error that doesn't match a known connection pattern (not a deadline,
// net.OpError, url.Error, DNSError, or secret-containing message). The nil return is intentional:
// the caller (handleAndClassify) falls back to returning the original error unchanged when nil
// is received, preserving pre-existing behavior for unrecognised error shapes.
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

// handleAndClassify is the shared error handling path for all transports.
// It classifies the error, emits metrics, logs it exactly once, then delegates
// to transport-specific handling (unsupported method wrapping, generic errors, etc.).
//
// Single-log invariant: each failure produces at most one structured log line
// from this path. The logic below ensures this even though handleGenericErrors
// has its own LavaFormatProduction logging for IP-masked connection errors:
//
//  1. Classified (non-Unknown) → LogCodedError emits metric + our structured
//     log. handleGenericErrors is NOT called. Return the wrapped LavaError.
//  2. Unknown + handleGenericErrors logged → emit metric only (via
//     EmitErrorMetric), skip our own log so we don't double-log.
//  3. Unknown + handleGenericErrors did NOT log (returned nil) → LogCodedError
//     emits metric + our structured log as the sole observability entry.
func handleAndClassify(ctx context.Context, nodeError error, transport common.TransportType, chainFamily common.ChainFamily, chainID string, geh *genericErrorHandler) error {
	// Classify. Use the details variant so the log/metric carries the real
	// JSON-RPC / gRPC / HTTP code and the inner error message — not the outer
	// wrapper string.
	classified, errorCode, errorMessage := ClassifyNodeErrorWithDetails(nodeError, chainFamily, transport)

	// Case 1: fully classified. Log + metric here; handleGenericErrors is not
	// reached, so there is no second log to worry about.
	if classified != common.LavaErrorUnknown {
		common.LogCodedError("provider node error", nodeError, classified, chainID, errorCode, errorMessage)
		return common.NewLavaError(classified, nodeError.Error())
	}

	// Unknown: fall through to handleGenericErrors, which applies IP masking
	// and emits its own LavaFormatProduction log for recognised connection
	// patterns. Returns nil for unrecognised shapes; callers treat a nil
	// return from HandleNodeError as "fall back to the raw error" (see e.g.
	// jsonRPC.go: `if parsedError != nil { return parsedError } return err`).
	parsed := geh.handleGenericErrors(ctx, nodeError)
	if parsed != nil {
		// handleGenericErrors already logged via LavaFormatProduction — emit
		// the metric directly to avoid a second structured log line from
		// LogCodedError. The Unknown rate is still visible in metrics.
		common.EmitErrorMetric(classified, chainID)
		return parsed
	}

	// Unknown AND handleGenericErrors did not log. Emit the single structured
	// log entry for this failure so the unknown error is still observable,
	// but discard LogCodedError's return so callers still see nil and fall
	// back to the raw error as before (preserving the pre-fix contract).
	_ = common.LogCodedError("provider node error", nodeError, classified, chainID, errorCode, errorMessage)
	return nil
}

type RestErrorHandler struct {
	genericErrorHandler
	chainFamily common.ChainFamily
	chainID     string
}

func (rne *RestErrorHandler) HandleNodeError(ctx context.Context, nodeError error) error {
	return handleAndClassify(ctx, nodeError, common.TransportREST, rne.chainFamily, rne.chainID, &rne.genericErrorHandler)
}

type JsonRPCErrorHandler struct {
	genericErrorHandler
	chainFamily common.ChainFamily
	chainID     string
}

func (jeh *JsonRPCErrorHandler) HandleNodeError(ctx context.Context, nodeError error) error {
	return handleAndClassify(ctx, nodeError, common.TransportJsonRPC, jeh.chainFamily, jeh.chainID, &jeh.genericErrorHandler)
}

type TendermintRPCErrorHandler struct {
	genericErrorHandler
	chainFamily common.ChainFamily
	chainID     string
}

func (tendermintErrorHandler *TendermintRPCErrorHandler) HandleNodeError(ctx context.Context, nodeError error) error {
	return handleAndClassify(ctx, nodeError, common.TransportJsonRPC, tendermintErrorHandler.chainFamily, tendermintErrorHandler.chainID, &tendermintErrorHandler.genericErrorHandler)
}

type GRPCErrorHandler struct {
	genericErrorHandler
	chainFamily common.ChainFamily
	chainID     string
}

func (geh *GRPCErrorHandler) HandleNodeError(ctx context.Context, nodeError error) error {
	return handleAndClassify(ctx, nodeError, common.TransportGRPC, geh.chainFamily, geh.chainID, &geh.genericErrorHandler)
}

type ErrorHandler interface {
	HandleNodeError(context.Context, error) error
	HandleStatusError(int, bool) error
	HandleJSONFormatError([]byte) error
	ValidateRequestAndResponseIds(json.RawMessage, json.RawMessage) error
}
