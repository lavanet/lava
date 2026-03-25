package rpcsmartrouter

import (
	"errors"
	"net"
	"syscall"
	"testing"

	"github.com/lavanet/lava/v5/protocol/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClassifyDirectRPCError_Nil(t *testing.T) {
	result := ClassifyDirectRPCError(nil)
	assert.Nil(t, result)
}

func TestClassifyDirectRPCError_ConnectionRefused(t *testing.T) {
	errno := syscall.ECONNREFUSED
	err := &net.OpError{
		Op:  "dial",
		Net: "tcp",
		Err: &net.OpError{
			Op:  "dial",
			Net: "tcp",
			Err: &errno,
		},
	}

	lavaErr := ClassifyDirectRPCError(err)
	require.NotNil(t, lavaErr)
	assert.Equal(t, common.LavaErrorConnectionRefused, lavaErr)
	assert.True(t, lavaErr.Retryable)
	assert.Equal(t, common.CategoryInternal, lavaErr.Category)
}

func TestClassifyDirectRPCError_Timeout(t *testing.T) {
	err := &mockNetError{timeout: true}

	lavaErr := ClassifyDirectRPCError(err)
	require.NotNil(t, lavaErr)
	assert.Equal(t, common.LavaErrorConnectionTimeout, lavaErr)
	assert.True(t, lavaErr.Retryable)
	assert.Equal(t, common.CategoryInternal, lavaErr.Category)
}

func TestClassifyDirectRPCError_HTTPStatusCodes(t *testing.T) {
	tests := []struct {
		name          string
		errorMsg      string
		expectedError *common.LavaError
	}{
		{
			name:          "HTTP 429 rate limit",
			errorMsg:      "HTTP status 429: Too Many Requests",
			expectedError: common.LavaErrorNodeRateLimited,
		},
		{
			name:          "HTTP 503 service unavailable",
			errorMsg:      "HTTP status 503: Service Unavailable",
			expectedError: common.LavaErrorNodeServiceUnavailable,
		},
		{
			name:          "HTTP 500 internal server error",
			errorMsg:      "HTTP status 500: Internal Server Error",
			expectedError: common.LavaErrorNodeInternalError,
		},
		{
			name:          "HTTP 502 bad gateway",
			errorMsg:      "HTTP status 502: Bad Gateway",
			expectedError: common.LavaErrorNodeBadGateway,
		},
		{
			name:          "HTTP 504 gateway timeout",
			errorMsg:      "HTTP status 504: Gateway Timeout",
			expectedError: common.LavaErrorNodeGatewayTimeout,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := errors.New(tt.errorMsg)
			lavaErr := ClassifyDirectRPCError(err)
			require.NotNil(t, lavaErr)
			assert.Equal(t, tt.expectedError, lavaErr)
			assert.Equal(t, common.CategoryExternal, lavaErr.Category)
			assert.True(t, lavaErr.Retryable)
		})
	}
}

func TestClassifyDirectRPCError_UnknownError(t *testing.T) {
	err := errors.New("your mom died")
	lavaErr := ClassifyDirectRPCError(err)
	require.NotNil(t, lavaErr)
	assert.Equal(t, common.LavaErrorUnknown, lavaErr)
	// Unknown errors are external — they're pass-throughs from the node
	assert.Equal(t, common.CategoryExternal, lavaErr.Category)
}

func TestClassifyDirectRPCError_RateLimitByMessage(t *testing.T) {
	err := errors.New("rate limit exceeded for this endpoint")
	lavaErr := ClassifyDirectRPCError(err)
	require.NotNil(t, lavaErr)
	assert.Equal(t, common.LavaErrorNodeRateLimited, lavaErr)
}

func TestClassifyDirectRPCError_InternalVsExternal(t *testing.T) {
	// Connection errors are internal (Lava protocol layer)
	timeoutErr := &mockNetError{timeout: true}
	lavaErr := ClassifyDirectRPCError(timeoutErr)
	assert.True(t, common.IsInternal(lavaErr.Code))

	// HTTP status errors are external (node/chain layer)
	httpErr := errors.New("HTTP status 503: Service Unavailable")
	lavaErr = ClassifyDirectRPCError(httpErr)
	assert.True(t, common.IsExternal(lavaErr.Code))
}

func TestClassifyError_MatchOrdering(t *testing.T) {
	// "rate limit" message should match before HTTP 429 status
	err := "rate limit 429"
	result := common.ClassifyError(nil, common.ChainFamilyEVM, common.TransportJsonRPC, 0, err)
	assert.Equal(t, common.LavaErrorNodeRateLimited, result)
}

func TestClassifyDirectRPCError_UnsupportedMethod(t *testing.T) {
	tests := []struct {
		name     string
		errorMsg string
		expected *common.LavaError
	}{
		{"method not found message", "method not found", common.LavaErrorNodeMethodNotFound},
		{"method not supported message", "the method is method not supported on this node", common.LavaErrorNodeMethodNotSupported},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := errors.New(tt.errorMsg)
			lavaErr := ClassifyDirectRPCError(err)
			require.NotNil(t, lavaErr)
			assert.Equal(t, tt.expected, lavaErr)
			assert.True(t, lavaErr.SubCategory.IsUnsupportedMethod())
		})
	}
}

func TestClassifyError_UnsupportedMethodByCode(t *testing.T) {
	// JSON-RPC -32601 should classify as unsupported method
	result := common.ClassifyError(nil, common.ChainFamilyEVM, common.TransportJsonRPC, -32601, "some error")
	assert.Equal(t, common.LavaErrorNodeMethodNotFound, result)
	assert.True(t, result.SubCategory.IsUnsupportedMethod())
}

func TestIsConnectionRefused(t *testing.T) {
	otherErr := &net.OpError{
		Op:  "dial",
		Net: "tcp",
		Err: syscall.ETIMEDOUT,
	}
	assert.False(t, isConnectionRefused(otherErr))

	regularErr := errors.New("regular error")
	assert.False(t, isConnectionRefused(regularErr))
}

func TestIsTimeout(t *testing.T) {
	timeoutErr := &mockNetError{timeout: true}
	assert.True(t, isTimeout(timeoutErr))

	nonTimeoutErr := &mockNetError{timeout: false}
	assert.False(t, isTimeout(nonTimeoutErr))

	regularErr := errors.New("regular error")
	assert.False(t, isTimeout(regularErr))
}

// mockNetError implements net.Error for testing
type mockNetError struct {
	timeout   bool
	temporary bool
}

func (e *mockNetError) Error() string   { return "mock net error" }
func (e *mockNetError) Timeout() bool   { return e.timeout }
func (e *mockNetError) Temporary() bool { return e.temporary }
