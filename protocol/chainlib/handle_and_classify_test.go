package chainlib

import (
	"context"
	"errors"
	"testing"

	"github.com/lavanet/lava/v5/protocol/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHandleAndClassify_UnsupportedMethod(t *testing.T) {
	// GIVEN a "method not found" error
	// WHEN handleAndClassify processes it
	// THEN it returns an UnsupportedMethodError wrapper
	err := errors.New("method not found")
	result := handleAndClassify(context.Background(), err, common.TransportJsonRPC, &genericErrorHandler{})
	require.NotNil(t, result)
	assert.True(t, IsUnsupportedMethodErrorType(result))
}

func TestHandleAndClassify_NonRetryableError(t *testing.T) {
	// GIVEN a non-retryable node error (e.g., nonce too low)
	// WHEN handleAndClassify processes it
	// THEN it returns a SolanaNonRetryableError wrapper (non-retryable indicator)
	err := errors.New("nonce too low")
	result := handleAndClassify(context.Background(), err, common.TransportJsonRPC, &genericErrorHandler{})
	require.NotNil(t, result)
	assert.True(t, IsSolanaNonRetryableErrorType(result))
}

func TestHandleAndClassify_RetryableError(t *testing.T) {
	// GIVEN a retryable error (e.g., rate limit)
	// WHEN handleAndClassify processes it
	// THEN it does NOT wrap with UnsupportedMethodError or SolanaNonRetryableError
	err := errors.New("rate limit exceeded")
	result := handleAndClassify(context.Background(), err, common.TransportJsonRPC, &genericErrorHandler{})
	// retryable errors go through handleGenericErrors which may return nil
	// (no connection error detected) — the important thing is no wrapping
	assert.False(t, IsUnsupportedMethodErrorType(result))
	assert.False(t, IsSolanaNonRetryableErrorType(result))
}

func TestHandleAndClassify_UnknownError(t *testing.T) {
	// GIVEN a completely unknown error
	// WHEN handleAndClassify processes it
	// THEN it falls through to handleGenericErrors (no special wrapping)
	err := errors.New("something completely unexpected")
	result := handleAndClassify(context.Background(), err, common.TransportJsonRPC, &genericErrorHandler{})
	assert.False(t, IsUnsupportedMethodErrorType(result))
	assert.False(t, IsSolanaNonRetryableErrorType(result))
}

func TestHandleAndClassify_JsonRPCMethodNotFound(t *testing.T) {
	// GIVEN a "method not supported" error (message-based match)
	// WHEN classified with TransportJsonRPC
	// THEN it's detected as unsupported method via SubCategory
	err := errors.New("method not supported")
	result := handleAndClassify(context.Background(), err, common.TransportJsonRPC, &genericErrorHandler{})
	require.NotNil(t, result)
	assert.True(t, IsUnsupportedMethodErrorType(result))
}
