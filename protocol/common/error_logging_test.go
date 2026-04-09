package common

import (
	"errors"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLogCodedError_NilLavaError(t *testing.T) {
	// Should not panic and should use LavaErrorUnknown
	err := LogCodedError("test", errors.New("boom"), nil, "", 0, "")
	require.NotNil(t, err)
}

func TestLogCodedError_WithLavaError(t *testing.T) {
	err := LogCodedError("test timeout", errors.New("connection timed out"),
		LavaErrorConnectionTimeout, "ETH1", 0, "connection timed out")
	require.NotNil(t, err)
}

func TestLogCodedError_WithChainError(t *testing.T) {
	err := LogCodedError("node error", errors.New("nonce too low"),
		LavaErrorChainNonceTooLow, "ETH1", -32000, "nonce too low")
	require.NotNil(t, err)
}

func TestLogCodedError_MetricsCallbackInvoked(t *testing.T) {
	var mu sync.Mutex
	var captured struct {
		code     uint32
		name     string
		category string
		retry    bool
		chain    string
	}

	// Register test callback
	SetErrorMetricsCallback(func(code uint32, name string, category string, retryable bool, chainID string) {
		mu.Lock()
		defer mu.Unlock()
		captured.code = code
		captured.name = name
		captured.category = category
		captured.retry = retryable
		captured.chain = chainID
	})
	defer SetErrorMetricsCallback(nil) // cleanup

	LogCodedError("test", errors.New("nonce too low"),
		LavaErrorChainNonceTooLow, "ETH1", -32000, "nonce too low")

	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, LavaErrorChainNonceTooLow.Code, captured.code)
	assert.Equal(t, "CHAIN_NONCE_TOO_LOW", captured.name)
	assert.Equal(t, "external", captured.category)
	assert.False(t, captured.retry)
	assert.Equal(t, "ETH1", captured.chain)
}

func TestLogCodedError_NoCallbackNoPanic(t *testing.T) {
	SetErrorMetricsCallback(nil)
	// Should not panic
	err := LogCodedError("test", errors.New("boom"), LavaErrorUnknown, "", 0, "")
	require.NotNil(t, err)
}
