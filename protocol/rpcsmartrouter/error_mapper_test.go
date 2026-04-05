package rpcsmartrouter

import (
	"errors"
	"net"
	"syscall"
	"testing"

	"github.com/lavanet/lava/v5/protocol/chainlib/chainproxy/rpcclient"
	"github.com/lavanet/lava/v5/protocol/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClassifyDirectRPCError_Nil(t *testing.T) {
	lavaErr, wrappedErr := classifyDirectRPCError(nil, -1, common.TransportJsonRPC)
	assert.Equal(t, common.LavaErrorUnknown, lavaErr)
	assert.Nil(t, wrappedErr)
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

	lavaErr, _ := classifyDirectRPCError(err, -1, common.TransportJsonRPC)
	require.NotNil(t, lavaErr)
	assert.Equal(t, common.LavaErrorConnectionRefused, lavaErr)
	assert.True(t, lavaErr.Retryable)
	assert.Equal(t, common.CategoryInternal, lavaErr.Category)
}

func TestClassifyDirectRPCError_Timeout(t *testing.T) {
	err := &mockNetError{timeout: true}

	lavaErr, _ := classifyDirectRPCError(err, -1, common.TransportJsonRPC)
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
			lavaErr, _ := classifyDirectRPCError(err, -1, common.TransportJsonRPC)
			require.NotNil(t, lavaErr)
			assert.Equal(t, tt.expectedError, lavaErr)
			assert.Equal(t, common.CategoryExternal, lavaErr.Category)
			assert.True(t, lavaErr.Retryable)
		})
	}
}

func TestClassifyDirectRPCError_UnknownError(t *testing.T) {
	err := errors.New("your mom died")
	lavaErr, _ := classifyDirectRPCError(err, -1, common.TransportJsonRPC)
	require.NotNil(t, lavaErr)
	assert.Equal(t, common.LavaErrorUnknown, lavaErr)
	// Unknown errors are external — they're pass-throughs from the node
	assert.Equal(t, common.CategoryExternal, lavaErr.Category)
}

func TestClassifyDirectRPCError_RateLimitByMessage(t *testing.T) {
	err := errors.New("rate limit exceeded for this endpoint")
	lavaErr, _ := classifyDirectRPCError(err, -1, common.TransportJsonRPC)
	require.NotNil(t, lavaErr)
	assert.Equal(t, common.LavaErrorNodeRateLimited, lavaErr)
}

func TestClassifyDirectRPCError_InternalVsExternal(t *testing.T) {
	// Connection errors are internal (Lava protocol layer)
	timeoutErr := &mockNetError{timeout: true}
	lavaErr, _ := classifyDirectRPCError(timeoutErr, -1, common.TransportJsonRPC)
	assert.True(t, common.IsInternal(lavaErr.Code))

	// HTTP status errors are external (node/chain layer)
	httpErr := errors.New("HTTP status 503: Service Unavailable")
	lavaErr, _ = classifyDirectRPCError(httpErr, -1, common.TransportJsonRPC)
	assert.True(t, common.IsExternal(lavaErr.Code))
}

func TestClassifyError_MatchOrdering(t *testing.T) {
	// "rate limit" message should match before HTTP 429 status
	err := "rate limit 429"
	result := common.ClassifyError(nil, common.ChainFamilyEVM, common.TransportJsonRPC, 0, err)
	assert.Equal(t, common.LavaErrorNodeRateLimited, result)
}

func TestClassifyDirectRPCError_UnsupportedMethod(t *testing.T) {
	// "method not found" has SubCategoryUnsupportedMethod (zero retries, cached response)
	err := errors.New("method not found")
	lavaErr, _ := classifyDirectRPCError(err, -1, common.TransportJsonRPC)
	require.NotNil(t, lavaErr)
	assert.Equal(t, common.LavaErrorNodeMethodNotFound, lavaErr)
	assert.True(t, lavaErr.SubCategory.IsUnsupportedMethod())

	// "method not supported" is retryable — another provider may support it — no SubCategory
	err = errors.New("the method is method not supported on this node")
	lavaErr, _ = classifyDirectRPCError(err, -1, common.TransportJsonRPC)
	require.NotNil(t, lavaErr)
	assert.Equal(t, common.LavaErrorNodeMethodNotSupported, lavaErr)
	assert.False(t, lavaErr.SubCategory.IsUnsupportedMethod())
	assert.True(t, lavaErr.Retryable)
}

func TestClassifyDirectRPCError_JSONRPCBodyExtraction(t *testing.T) {
	// GIVEN an HTTP error whose body contains a Solana-specific JSON-RPC error code
	// WHEN classifyDirectRPCError processes it with ChainFamilySolana
	// THEN the code is extracted from the body and Tier 2 classification fires
	jsonBody := `{"jsonrpc":"2.0","error":{"code":-32009,"message":"Slot 123 was skipped, or missing in long-term storage"},"id":1}`
	httpErr := rpcclient.HTTPError{StatusCode: 200, Status: "200 OK", Body: []byte(jsonBody)}

	lavaErr, _ := classifyDirectRPCError(httpErr, common.ChainFamilySolana, common.TransportJsonRPC)
	require.NotNil(t, lavaErr)
	assert.Equal(t, common.LavaErrorChainSolanaMissingLongTerm, lavaErr, "Solana -32009 should be extracted from HTTP body and classified via Tier 2")
}

func TestClassifyError_UnsupportedMethodByCode(t *testing.T) {
	// JSON-RPC -32601 should classify as unsupported method
	result := common.ClassifyError(nil, common.ChainFamilyEVM, common.TransportJsonRPC, -32601, "some error")
	assert.Equal(t, common.LavaErrorNodeMethodNotFound, result)
	assert.True(t, result.SubCategory.IsUnsupportedMethod())
}

func TestDetectConnectionError_NotRefused(t *testing.T) {
	// ETIMEDOUT is a timeout, not a connection refused
	otherErr := &net.OpError{
		Op:  "dial",
		Net: "tcp",
		Err: syscall.ETIMEDOUT,
	}
	assert.NotEqual(t, common.LavaErrorConnectionRefused, common.DetectConnectionError(otherErr))

	regularErr := errors.New("regular error")
	assert.Nil(t, common.DetectConnectionError(regularErr))
}

func TestDetectConnectionError_Timeout(t *testing.T) {
	timeoutErr := &mockNetError{timeout: true}
	assert.Equal(t, common.LavaErrorConnectionTimeout, common.DetectConnectionError(timeoutErr))

	nonTimeoutErr := &mockNetError{timeout: false}
	assert.Nil(t, common.DetectConnectionError(nonTimeoutErr))

	regularErr := errors.New("regular error")
	assert.Nil(t, common.DetectConnectionError(regularErr))
}

func TestExtractLavaError_FromWrappedError(t *testing.T) {
	origErr := errors.New("nonce too low")
	_, wrappedErr := classifyDirectRPCError(origErr, -1, common.TransportJsonRPC)
	require.NotNil(t, wrappedErr)

	lavaErr := extractLavaError(wrappedErr)
	require.NotNil(t, lavaErr)
	assert.Equal(t, common.LavaErrorChainNonceTooLow, lavaErr)
}

func TestExtractLavaError_FromPlainError(t *testing.T) {
	plainErr := errors.New("plain error")
	lavaErr := extractLavaError(plainErr)
	assert.Nil(t, lavaErr)
}

func TestWrappedError_ErrorsIs(t *testing.T) {
	// LavaWrappedError.Is() enables errors.Is matching against the *LavaError sentinel.
	// This was broken with the old classifiedError whose Unwrap() returned Original,
	// not the LavaError, so errors.Is(err, LavaErrorSomething) never worked.
	origErr := errors.New("nonce too low")
	_, wrappedErr := classifyDirectRPCError(origErr, -1, common.TransportJsonRPC)
	require.NotNil(t, wrappedErr)

	assert.True(t, errors.Is(wrappedErr, common.LavaErrorChainNonceTooLow), "errors.Is should match the LavaError sentinel")
}

// mockNetError implements net.Error for testing
type mockNetError struct {
	timeout   bool
	temporary bool
}

func (e *mockNetError) Error() string   { return "mock net error" }
func (e *mockNetError) Timeout() bool   { return e.timeout }
func (e *mockNetError) Temporary() bool { return e.temporary }

// ---------------------------------------------------------------------------
// Endpoint health classification tests (GIVEN–WHEN–THEN)
// ---------------------------------------------------------------------------

func TestClassifyEndpointHealth_InternalError(t *testing.T) {
	// GIVEN a CategoryInternal error (transport timeout, connection refused, DNS failure)
	// WHEN the relay fails
	// THEN the endpoint is marked unhealthy AND backoff is requested
	internalErrors := []*common.LavaError{
		common.LavaErrorConnectionTimeout,
		common.LavaErrorConnectionRefused,
		common.LavaErrorDNSFailure,
		common.LavaErrorConnectionReset,
		common.LavaErrorContextDeadline,
	}
	for _, le := range internalErrors {
		unhealthy, backoff := classifyEndpointHealth(le)
		assert.True(t, unhealthy, "%s should mark unhealthy", le.Name)
		assert.True(t, backoff, "%s should request backoff", le.Name)
	}
}

func TestClassifyEndpointHealth_ExternalRetryable(t *testing.T) {
	// GIVEN a CategoryExternal + Retryable error (5xx, node syncing)
	// WHEN the relay fails
	// THEN backoff is requested AND endpoint is marked unhealthy
	retryableErrors := []*common.LavaError{
		common.LavaErrorNodeInternalError,
		common.LavaErrorNodeServiceUnavailable,
		common.LavaErrorNodeBadGateway,
		common.LavaErrorNodeGatewayTimeout,
		common.LavaErrorNodeSyncing,
	}
	for _, le := range retryableErrors {
		unhealthy, backoff := classifyEndpointHealth(le)
		assert.True(t, unhealthy, "%s should mark unhealthy", le.Name)
		assert.True(t, backoff, "%s should request backoff", le.Name)
	}
}

func TestClassifyEndpointHealth_RateLimited(t *testing.T) {
	// GIVEN a rate-limited error (CategoryExternal + Retryable but rate-limited)
	// WHEN the relay fails
	// THEN backoff is requested but endpoint is NOT marked unhealthy (it's healthy, just busy)
	unhealthy, backoff := classifyEndpointHealth(common.LavaErrorNodeRateLimited)
	assert.False(t, unhealthy, "rate-limited should NOT mark unhealthy")
	assert.True(t, backoff, "rate-limited should request backoff")
}

func TestClassifyEndpointHealth_ExternalNonRetryable(t *testing.T) {
	// GIVEN a CategoryExternal + non-retryable error (4xx, unsupported method, nonce too low)
	// WHEN the relay fails
	// THEN neither mark unhealthy nor backoff (error is the user's or permanent)
	nonRetryableErrors := []*common.LavaError{
		common.LavaErrorNodeMethodNotFound,
		common.LavaErrorNodeEndpointNotFound,
		common.LavaErrorChainNonceTooLow,
		common.LavaErrorChainExecutionReverted,
		common.LavaErrorUserInvalidParams,
	}
	for _, le := range nonRetryableErrors {
		unhealthy, backoff := classifyEndpointHealth(le)
		assert.False(t, unhealthy, "%s should NOT mark unhealthy", le.Name)
		assert.False(t, backoff, "%s should NOT request backoff", le.Name)
	}
}

func TestClassifyEndpointHealth_Nil(t *testing.T) {
	// GIVEN a nil classification
	// WHEN health is evaluated
	// THEN neither mark unhealthy nor backoff
	unhealthy, backoff := classifyEndpointHealth(nil)
	assert.False(t, unhealthy)
	assert.False(t, backoff)
}
