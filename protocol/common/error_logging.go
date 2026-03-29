package common

import (
	"fmt"
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

// LogCodedError logs an error with a classified LavaError, auto-populating structured fields
// and invoking the metrics callback if registered.
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
	codedAttrs = append(codedAttrs, attributes...)

	// Invoke metrics callback if registered
	errorMetricsMu.RLock()
	cb := errorMetricsCallback
	errorMetricsMu.RUnlock()
	if cb != nil {
		cb(lavaError.Code, lavaError.Name, lavaError.Category.String(), lavaError.Retryable, chainID)
	}

	return utils.LavaFormatError(description, err, codedAttrs...)
}

// ClassifyAndLogNodeError is a convenience for CheckResponseError implementations.
// It classifies an error by transport, logs it with LogCodedError, and returns the classification.
func ClassifyAndLogNodeError(transport TransportType, errorCode int, errorMessage string) *LavaError {
	classified := ClassifyError(nil, -1, transport, errorCode, errorMessage)
	LogCodedError("node error", fmt.Errorf("%s", errorMessage), classified, "", errorCode, errorMessage)
	return classified
}
