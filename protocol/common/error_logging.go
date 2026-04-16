package common

import (
	"encoding/json"
	"sync/atomic"

	"github.com/lavanet/lava/v5/utils"
)

// ErrorMetricsCallback is a function that gets called on every classified error for metrics tracking.
// Registered via SetErrorMetricsCallback from the metrics package during initialization.
type ErrorMetricsCallback func(errorCode uint32, errorName string, errorCategory string, retryable bool, chainID string)

// errorMetricsCallbackPtr holds the registered metrics callback. It is loaded on
// every classified error (high-QPS path) and stored only at init / in tests, so
// it uses an atomic pointer to avoid the RWMutex reader-lock CAS on the hot path.
var errorMetricsCallbackPtr atomic.Pointer[ErrorMetricsCallback]

// SetErrorMetricsCallback registers a callback for error metrics (e.g., Prometheus counter).
// Called once during application initialization from the metrics package.
// Passing nil clears the callback (used by tests).
func SetErrorMetricsCallback(cb ErrorMetricsCallback) {
	if cb == nil {
		errorMetricsCallbackPtr.Store(nil)
		return
	}
	errorMetricsCallbackPtr.Store(&cb)
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

// EmitErrorMetric fires the registered metrics callback for a classified error
// without emitting any log line. Prefer LogCodedError/LogCodedWarning for the
// common case; use this directly only when the caller has already logged (or
// is about to log) the same failure via a different path and needs to ensure
// the Prometheus counter is still incremented exactly once.
//
// Uses an atomic pointer load — no locks on the hot classification path.
func EmitErrorMetric(lavaError *LavaError, chainID string) {
	cb := errorMetricsCallbackPtr.Load()
	if cb == nil {
		return
	}
	(*cb)(lavaError.Code, lavaError.Name, lavaError.Category.String(), lavaError.Retryable, chainID)
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
	EmitErrorMetric(lavaError, chainID)
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
	EmitErrorMetric(lavaError, chainID)
	return utils.LavaFormatWarning(description, err, buildCodedAttrs(lavaError, chainID, chainErrorCode, chainErrorMessage, attributes...)...)
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
