package rpcsmartrouter

import (
	"errors"

	"github.com/lavanet/lava/v5/protocol/chainlib"
	"github.com/lavanet/lava/v5/protocol/common"
)

// extractLavaError extracts the *common.LavaError from a LavaWrappedError,
// or returns nil if the error is not (or does not wrap) a LavaWrappedError.
func extractLavaError(err error) *common.LavaError {
	var wrapped *common.LavaWrappedError
	if errors.As(err, &wrapped) {
		return wrapped.LavaErr
	}
	return nil
}

// ---------------------------------------------------------------------------
// Classification helpers
// ---------------------------------------------------------------------------

// classifyDirectRPCError classifies a direct RPC error into a LavaError for
// internal use (logging, metrics, endpoint health). The original error is never
// modified — the router is a transparent hop for the user.
// Returns both the classification and a classifiedError that wraps the original.
func classifyDirectRPCError(err error, chainFamily common.ChainFamily, transport common.TransportType) (*common.LavaError, error) {
	if err == nil {
		return common.LavaErrorUnknown, nil
	}

	// Connection-level errors — detected before inspecting the message
	connError := common.DetectConnectionError(err)

	// Extract JSON-RPC/gRPC/HTTP error code and canonical message, then classify
	errorCode, errorMessage := chainlib.ExtractNodeErrorDetails(err)
	classified := common.ClassifyError(connError, chainFamily, transport, errorCode, errorMessage)

	return classified, common.NewLavaError(classified, err.Error())
}

// classifyAndWrap is a convenience that calls classifyDirectRPCError and returns
// only the wrapped error (discarding the *LavaError for call sites that don't need it).
func classifyAndWrap(err error, chainFamily common.ChainFamily, transport common.TransportType) error {
	if err == nil {
		return nil
	}
	_, wrapped := classifyDirectRPCError(err, chainFamily, transport)
	return wrapped
}

// classifyEndpointHealth decides whether an endpoint should be marked unhealthy
// and/or backed off based on the classified error.
//
// Rules:
//   - CategoryInternal (timeout, connection refused, DNS) → unhealthy + backoff
//   - CategoryExternal + Retryable (5xx, syncing) → backoff + unhealthy (except rate limit)
//   - CategoryExternal + Retryable + RateLimited → backoff only (endpoint is healthy, just busy)
//   - CategoryExternal + !Retryable (4xx, unsupported) → neither (error is the user's)
func classifyEndpointHealth(classified *common.LavaError) (shouldMarkUnhealthy bool, needsBackoff bool) {
	if classified == nil {
		return false, false
	}
	switch {
	case classified.Category == common.CategoryInternal:
		return true, true
	case classified.Category == common.CategoryExternal && classified.Retryable:
		if classified == common.LavaErrorNodeRateLimited {
			return false, true // healthy but busy
		}
		return true, true
	default:
		return false, false
	}
}
