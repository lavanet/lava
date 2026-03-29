package rpcsmartrouter

import (
	"errors"
	"net"
	"syscall"

	"github.com/lavanet/lava/v5/protocol/common"
)

// ---------------------------------------------------------------------------
// classifiedError — wraps an error with its classification metadata
// ---------------------------------------------------------------------------

// classifiedError wraps an original error with its LavaError classification.
// It implements the error interface and supports errors.Unwrap().
type classifiedError struct {
	Original  error
	LavaError *common.LavaError
}

func (ce *classifiedError) Error() string {
	return ce.Original.Error()
}

func (ce *classifiedError) Unwrap() error {
	return ce.Original
}

// extractLavaError extracts the *common.LavaError from a classifiedError,
// or returns nil if the error is not (or does not wrap) a classifiedError.
func extractLavaError(err error) *common.LavaError {
	var ce *classifiedError
	if errors.As(err, &ce) {
		return ce.LavaError
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
	var connError *common.LavaError
	if isConnectionRefused(err) {
		connError = common.LavaErrorConnectionRefused
	} else if isTimeout(err) {
		connError = common.LavaErrorConnectionTimeout
	}

	// Classify using the correct chain family for Tier 2 matchers
	classified := common.ClassifyError(connError, chainFamily, transport, 0, err.Error())

	common.LogCodedError("direct RPC error", err, classified, "", 0, err.Error())

	return classified, &classifiedError{Original: err, LavaError: classified}
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

func isConnectionRefused(err error) bool {
	var netErr *net.OpError
	if errors.As(err, &netErr) {
		var syscallErr *syscall.Errno
		if errors.As(netErr.Err, &syscallErr) {
			return *syscallErr == syscall.ECONNREFUSED
		}
	}
	return false
}

func isTimeout(err error) bool {
	var netErr net.Error
	return errors.As(err, &netErr) && netErr.Timeout()
}
