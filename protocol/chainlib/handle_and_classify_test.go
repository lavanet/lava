package chainlib

import (
	"context"
	"errors"
	"io"
	"sync"
	"testing"

	"github.com/lavanet/lava/v5/protocol/chainlib/chainproxy/rpcclient"
	"github.com/lavanet/lava/v5/protocol/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestHandleAndClassify_UnsupportedMethod(t *testing.T) {
	// GIVEN a "method not found" error
	// WHEN handleAndClassify processes it
	// THEN it returns an UnsupportedMethodError wrapper
	err := errors.New("method not found")
	result := handleAndClassify(context.Background(), err, common.TransportJsonRPC, common.ChainFamilyEVM, "", &genericErrorHandler{})
	require.NotNil(t, result)
	assert.True(t, IsUnsupportedMethodErrorType(result))
}

func TestHandleAndClassify_NonRetryableError(t *testing.T) {
	// GIVEN a non-retryable node error (e.g., nonce too low)
	// WHEN handleAndClassify processes it
	// THEN it returns a SolanaNonRetryableError wrapper (non-retryable indicator)
	err := errors.New("nonce too low")
	result := handleAndClassify(context.Background(), err, common.TransportJsonRPC, common.ChainFamilyEVM, "", &genericErrorHandler{})
	require.NotNil(t, result)
	assert.True(t, IsSolanaNonRetryableErrorType(result))
}

func TestHandleAndClassify_RetryableError(t *testing.T) {
	// GIVEN a retryable error (e.g., rate limit)
	// WHEN handleAndClassify processes it
	// THEN it is wrapped (classified) but not as unsupported-method or non-retryable
	err := errors.New("rate limit exceeded")
	result := handleAndClassify(context.Background(), err, common.TransportJsonRPC, common.ChainFamilyEVM, "", &genericErrorHandler{})
	require.NotNil(t, result)
	assert.True(t, errors.Is(result, common.LavaErrorNodeRateLimited))
	assert.False(t, IsUnsupportedMethodErrorType(result))
	assert.False(t, IsSolanaNonRetryableErrorType(result))
}

func TestHandleAndClassify_UnknownError(t *testing.T) {
	// GIVEN a completely unknown error
	// WHEN handleAndClassify processes it
	// THEN it falls through to handleGenericErrors (no special wrapping)
	err := errors.New("something completely unexpected")
	result := handleAndClassify(context.Background(), err, common.TransportJsonRPC, common.ChainFamilyEVM, "", &genericErrorHandler{})
	assert.False(t, IsUnsupportedMethodErrorType(result))
	assert.False(t, IsSolanaNonRetryableErrorType(result))
}

func TestClassifyNodeError_NilError(t *testing.T) {
	result := ClassifyNodeError(nil, -1, common.TransportJsonRPC)
	assert.Nil(t, result)
}

func TestClassifyNodeError_PlainError(t *testing.T) {
	result := ClassifyNodeError(errors.New("nonce too low"), common.ChainFamilyEVM, common.TransportJsonRPC)
	require.NotNil(t, result)
	assert.Equal(t, common.LavaErrorChainNonceTooLow, result)
}

func TestClassifyNodeError_GRPCStatusError(t *testing.T) {
	// gRPC Unimplemented should classify as unsupported method
	grpcErr := status.Error(codes.Unimplemented, "method not implemented")
	result := ClassifyNodeError(grpcErr, -1, common.TransportGRPC)
	require.NotNil(t, result)
	assert.True(t, result.SubCategory.IsUnsupportedMethod())
}

func TestClassifyNodeError_HTTPError(t *testing.T) {
	// HTTPError with JSON-RPC body containing error code
	httpErr := rpcclient.HTTPError{
		StatusCode: 200,
		Body:       []byte(`{"jsonrpc":"2.0","id":1,"error":{"code":-32601,"message":"Method not found"}}`),
	}
	result := ClassifyNodeError(httpErr, -1, common.TransportJsonRPC)
	require.NotNil(t, result)
	assert.Equal(t, common.LavaErrorNodeMethodNotFound, result)
}

// TestExtractNodeErrorDetails_JSONRPCBodyBeatsHTTPStatus locks in the priority
// rule documented on ExtractNodeErrorDetails: a JSON-RPC error body inside an
// rpcclient.HTTPError MUST win over the wrapping HTTP status. Before the fix,
// the HTTP status unconditionally overwrote the JSON-RPC code, which caused
// classification to lose the structured error code.
func TestExtractNodeErrorDetails_JSONRPCBodyBeatsHTTPStatus(t *testing.T) {
	// Use code 3 (generic JSON-RPC server error reserved range) with an
	// opaque message so the classifier cannot fall back to a message matcher.
	// If priority is broken the returned code would be 400 (HTTP status),
	// which matches no matcher and classifies as Unknown.
	httpErr := rpcclient.HTTPError{
		StatusCode: 400,
		Body:       []byte(`{"jsonrpc":"2.0","id":1,"error":{"code":-32700,"message":"opaque"}}`),
	}
	code, msg := ExtractNodeErrorDetails(httpErr)
	assert.Equal(t, -32700, code, "JSON-RPC body code must beat HTTP status")
	assert.Equal(t, "opaque", msg, "JSON-RPC body message must beat outer error string")

	// End-to-end: classification must honour the extracted JSON-RPC code.
	result := ClassifyNodeError(httpErr, -1, common.TransportJsonRPC)
	require.NotNil(t, result)
	assert.Equal(t, common.LavaErrorUserParseError, result,
		"-32700 must classify as USER_PARSE_ERROR; if the HTTP status overwrote the code we'd get UNKNOWN")
}

// TestExtractNodeErrorDetails_HTTPStatusFallback covers the third-priority
// branch: a raw HTTPError with no recoverable JSON-RPC body falls back to the
// HTTP status code.
func TestExtractNodeErrorDetails_HTTPStatusFallback(t *testing.T) {
	httpErr := rpcclient.HTTPError{
		StatusCode: 503,
		Body:       []byte(`<html>Service Unavailable</html>`),
	}
	code, _ := ExtractNodeErrorDetails(httpErr)
	assert.Equal(t, 503, code, "HTTP status must be used when no JSON-RPC body is recoverable")
}

func TestUnwrapLavaError_FromWrapped(t *testing.T) {
	err := common.NewLavaError(common.LavaErrorChainNonceTooLow, "test")
	le := unwrapLavaError(err)
	require.NotNil(t, le)
	assert.Equal(t, common.LavaErrorChainNonceTooLow, le)
}

func TestUnwrapLavaError_FromPlainError(t *testing.T) {
	le := unwrapLavaError(errors.New("plain"))
	assert.Nil(t, le)
}

func TestUnwrapLavaError_Nil(t *testing.T) {
	le := unwrapLavaError(nil)
	assert.Nil(t, le)
}

func TestHandleAndClassify_JsonRPCMethodNotFound(t *testing.T) {
	// GIVEN a "method not found" error (message-based match)
	// WHEN classified with TransportJsonRPC
	// THEN it's detected as unsupported method via SubCategory
	err := errors.New("method not found")
	result := handleAndClassify(context.Background(), err, common.TransportJsonRPC, common.ChainFamilyEVM, "", &genericErrorHandler{})
	require.NotNil(t, result)
	assert.True(t, IsUnsupportedMethodErrorType(result))
}

func TestHandleAndClassify_MethodNotSupported(t *testing.T) {
	// GIVEN a "method not supported" error (retryable — disabled on this node, try another)
	// WHEN handleAndClassify processes it
	// THEN it is wrapped so callers can observe the Retryable=true signal via errors.Is
	err := errors.New("method not supported")
	result := handleAndClassify(context.Background(), err, common.TransportJsonRPC, common.ChainFamilyEVM, "", &genericErrorHandler{})
	require.NotNil(t, result, "retryable classified errors must be wrapped, not silently dropped")
	assert.True(t, errors.Is(result, common.LavaErrorNodeMethodNotSupported))
	assert.False(t, IsUnsupportedMethodErrorType(result), "method not supported is retryable, not a hard unsupported-method")
	assert.True(t, ShouldRetryError(result), "Retryable=true must be surfaced to the retry logic")
}

// metricCounter captures calls to the error metrics callback so tests can
// assert the exact number of emissions for a single handleAndClassify call.
type metricCounter struct {
	mu    sync.Mutex
	count int
	last  struct {
		code     uint32
		name     string
		category string
	}
}

func (m *metricCounter) callback() common.ErrorMetricsCallback {
	return func(code uint32, name string, category string, retryable bool, chainID string) {
		m.mu.Lock()
		defer m.mu.Unlock()
		m.count++
		m.last.code = code
		m.last.name = name
		m.last.category = category
	}
}

func (m *metricCounter) snapshot() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.count
}

// TestHandleAndClassify_SingleMetricPerCall verifies the single-emit invariant:
// every path through handleAndClassify emits exactly one metric (including the
// Unknown + handleGenericErrors branch, which previously double-logged via both
// LogCodedError and LavaFormatProduction).
func TestHandleAndClassify_SingleMetricPerCall(t *testing.T) {
	tests := []struct {
		name         string
		err          error
		expectedCode uint32
	}{
		{
			name:         "classified non-Unknown",
			err:          errors.New("method not found"),
			expectedCode: common.LavaErrorNodeMethodNotFound.Code,
		},
		{
			name:         "classified retryable",
			err:          errors.New("rate limit exceeded"),
			expectedCode: common.LavaErrorNodeRateLimited.Code,
		},
		{
			name:         "Unknown fall-through (handleGenericErrors returns nil)",
			err:          errors.New("something completely unexpected"),
			expectedCode: common.LavaErrorUnknown.Code,
		},
		{
			// io.EOF is not in the registry (classifies Unknown), but
			// handleGenericErrors recognises it and logs via LavaFormatProduction.
			// The handler routes the metric through EmitErrorMetric only for
			// this path so the structured log fires exactly once while the
			// Prometheus counter still increments.
			name:         "Unknown + handleGenericErrors logs (io.EOF)",
			err:          io.EOF,
			expectedCode: common.LavaErrorUnknown.Code,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := &metricCounter{}
			common.SetErrorMetricsCallback(mc.callback())
			defer common.SetErrorMetricsCallback(nil)

			handleAndClassify(context.Background(), tt.err, common.TransportJsonRPC, common.ChainFamilyEVM, "ETH1", &genericErrorHandler{})

			assert.Equal(t, 1, mc.snapshot(), "exactly one metric must fire per handleAndClassify call")
			assert.Equal(t, tt.expectedCode, mc.last.code, "metric must reflect the final classification")
		})
	}
}
