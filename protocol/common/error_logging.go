package common

import (
	"encoding/json"
	"sync"

	"github.com/lavanet/lava/v5/utils"
)

// ErrorMetricsCallback is a function that gets called on every classified error for metrics tracking.
// Registered via SetErrorMetricsCallback from the metrics package during initialization.
type ErrorMetricsCallback func(errorCode uint32, errorName string, errorCategory string, retryable bool, chainID string)

var (
	errorMetricsCallback ErrorMetricsCallback
	errorMetricsMu       sync.RWMutex
)

// SetErrorMetricsCallback registers a callback for error metrics (e.g., Prometheus counter).
// Called once during application initialization from the metrics package.
func SetErrorMetricsCallback(cb ErrorMetricsCallback) {
	errorMetricsMu.Lock()
	defer errorMetricsMu.Unlock()
	errorMetricsCallback = cb
}

// buildCodedAttrs assembles the structured log attributes for a classified error.
func buildCodedAttrs(lavaError *LavaError, chainID string, chainErrorCode int, chainErrorMessage string, attributes ...utils.Attribute) []utils.Attribute {
	codedAttrs := []utils.Attribute{
		{Key: "error_code", Value: lavaError.Code},
		{Key: "error_name", Value: lavaError.Name},
		{Key: "error_category", Value: lavaError.Category.String()},
		{Key: "retryable", Value: lavaError.Retryable},
	}
	if chainErrorCode != 0 {
		codedAttrs = append(codedAttrs, utils.Attribute{Key: "chain_error_code", Value: chainErrorCode})
	}
	if chainErrorMessage != "" {
		codedAttrs = append(codedAttrs, utils.Attribute{Key: "chain_error_message", Value: chainErrorMessage})
	}
	if chainID != "" {
		codedAttrs = append(codedAttrs, utils.Attribute{Key: "chain_id", Value: chainID})
	}
	return append(codedAttrs, attributes...)
}

// emitErrorMetric fires the registered metrics callback for a classified error.
func emitErrorMetric(lavaError *LavaError, chainID string) {
	errorMetricsMu.RLock()
	cb := errorMetricsCallback
	errorMetricsMu.RUnlock()
	if cb != nil {
		cb(lavaError.Code, lavaError.Name, lavaError.Category.String(), lavaError.Retryable, chainID)
	}
}

// LogCodedError logs an error at Error level with a classified LavaError, auto-populating
// structured fields and invoking the metrics callback if registered.
//
// Parameters:
//   - description: human-readable context (e.g., "relay to provider failed")
//   - err: the original error
//   - lavaError: the classified LavaError (nil defaults to LavaErrorUnknown)
//   - chainID: the chain ID for metrics labels (may be empty)
//   - chainErrorCode: the original error code from the node (0 if not applicable)
//   - chainErrorMessage: the original error message from the node (may be empty)
//   - attributes: additional structured log attributes
func LogCodedError(description string, err error, lavaError *LavaError, chainID string, chainErrorCode int, chainErrorMessage string, attributes ...utils.Attribute) error {
	if lavaError == nil {
		lavaError = LavaErrorUnknown
	}
	emitErrorMetric(lavaError, chainID)
	return utils.LavaFormatError(description, err, buildCodedAttrs(lavaError, chainID, chainErrorCode, chainErrorMessage, attributes...)...)
}

// LogCodedWarning logs an error at Warning level with a classified LavaError, auto-populating
// structured fields and invoking the metrics callback if registered.
//
// Use this for expected/non-critical errors that should be tracked in metrics but should not
// appear as errors in logs (e.g., session-out-of-sync, invalid consumer requests, relay
// capacity limits that are normal in production).
func LogCodedWarning(description string, err error, lavaError *LavaError, chainID string, chainErrorCode int, chainErrorMessage string, attributes ...utils.Attribute) error {
	if lavaError == nil {
		lavaError = LavaErrorUnknown
	}
	emitErrorMetric(lavaError, chainID)
	return utils.LavaFormatWarning(description, err, buildCodedAttrs(lavaError, chainID, chainErrorCode, chainErrorMessage, attributes...)...)
}

// ClassifyNodeErrorByTransport classifies a node error by transport and returns the classification.
// It does NOT log — callers higher in the stack (e.g. ResultsManager) log with chain_id context.
func ClassifyNodeErrorByTransport(transport TransportType, errorCode int, errorMessage string) *LavaError {
	return ClassifyError(nil, -1, transport, errorCode, errorMessage)
}

// ExtractJSONRPCErrorCode extracts the error code from a JSON-RPC error response body.
// Returns 0 if the body is not a valid JSON-RPC error.
func ExtractJSONRPCErrorCode(data []byte) int {
	var result struct {
		Error *struct {
			Code int `json:"code"`
		} `json:"error,omitempty"`
	}
	if json.Unmarshal(data, &result) == nil && result.Error != nil {
		return result.Error.Code
	}
	return 0
}
