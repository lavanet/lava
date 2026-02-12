package chainlib

import (
	"errors"
	"fmt"
	"testing"

	"github.com/lavanet/lava/v5/protocol/chainlib/chainproxy/rpcclient"
	"github.com/lavanet/lava/v5/protocol/common"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestUnsupportedMethodError(t *testing.T) {
	t.Run("Error formatting with method name", func(t *testing.T) {
		originalErr := errors.New("original error")
		err := NewUnsupportedMethodError(originalErr, "eth_someMethod")

		require.Equal(t, "unsupported method 'eth_someMethod': original error", err.Error())
	})

	t.Run("Error formatting without method name", func(t *testing.T) {
		originalErr := errors.New("original error")
		err := NewUnsupportedMethodError(originalErr, "")

		require.Equal(t, "unsupported method: original error", err.Error())
	})

	t.Run("WithMethod sets method name", func(t *testing.T) {
		originalErr := errors.New("original error")
		err := &UnsupportedMethodError{originalError: originalErr}
		err = err.WithMethod("eth_call")

		require.Equal(t, "eth_call", err.methodName)
		require.Equal(t, "unsupported method 'eth_call': original error", err.Error())
	})

	t.Run("Unwrap returns original error", func(t *testing.T) {
		originalErr := errors.New("original error")
		err := NewUnsupportedMethodError(originalErr, "method")

		require.Equal(t, originalErr, err.Unwrap())
	})

	t.Run("GetMethodName returns method name", func(t *testing.T) {
		originalErr := errors.New("original error")
		err := NewUnsupportedMethodError(originalErr, "test-method")
		require.Equal(t, "test-method", err.GetMethodName())
	})

	t.Run("GetMethodName returns empty string when no method name", func(t *testing.T) {
		originalErr := errors.New("original error")
		err := NewUnsupportedMethodError(originalErr, "")
		require.Equal(t, "", err.GetMethodName())
	})
}

func TestIsUnsupportedMethodErrorMessageBytes(t *testing.T) {
	tests := []struct {
		name     string
		message  []byte
		expected bool
	}{
		{
			name:     "JSON-RPC method not found",
			message:  []byte("Method not found"),
			expected: true,
		},
		{
			name:     "Empty message",
			message:  []byte(""),
			expected: false,
		},
		{
			name:     "Generic error",
			message:  []byte("Internal server error"),
			expected: false,
		},
		{
			name:     "gRPC unimplemented",
			message:  []byte("UNIMPLEMENTED: unknown service"),
			expected: true,
		},
		{
			name:     "JSON-RPC error code",
			message:  []byte(`{"error":{"code":-32601,"message":"Method not found"}}`),
			expected: true,
		},
		{
			name:     "Nil message",
			message:  nil,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := common.IsUnsupportedMethodErrorMessageBytes(tt.message)
			require.Equal(t, tt.expected, result, "Message: %s", string(tt.message))
		})
	}
}

func TestIsUnsupportedMethodErrorMessage(t *testing.T) {
	tests := []struct {
		name     string
		message  string
		expected bool
	}{
		// JSON-RPC patterns
		{
			name:     "JSON-RPC method not found",
			message:  "Method not found",
			expected: true,
		},
		{
			name:     "JSON-RPC method not found lowercase",
			message:  "method not found",
			expected: true,
		},
		{
			name:     "JSON-RPC method not supported",
			message:  "Method not supported",
			expected: true,
		},
		{
			name:     "JSON-RPC unknown method",
			message:  "Unknown method: eth_someNewMethod",
			expected: true,
		},
		{
			name:     "JSON-RPC method does not exist",
			message:  "The method does not exist/is not available",
			expected: true,
		},
		{
			name:     "JSON-RPC invalid method",
			message:  "Invalid method parameter(s)",
			expected: true,
		},
		{
			name:     "JSON-RPC error code -32601",
			message:  "Error: -32601: Method not found",
			expected: true,
		},
		// REST patterns
		{
			name:     "REST endpoint not found",
			message:  "Endpoint not found",
			expected: true,
		},
		{
			name:     "REST route not found",
			message:  "Route not found for /api/v1/unknown",
			expected: true,
		},
		{
			name:     "REST path not found",
			message:  "Path not found",
			expected: true,
		},
		{
			name:     "REST method not allowed",
			message:  "405 Method Not Allowed",
			expected: true,
		},
		// gRPC patterns
		{
			name:     "gRPC method not implemented",
			message:  "Method not implemented",
			expected: true,
		},
		{
			name:     "gRPC unimplemented",
			message:  "UNIMPLEMENTED: unknown service",
			expected: true,
		},
		{
			name:     "gRPC not implemented capitalized",
			message:  "Not Implemented",
			expected: true,
		},
		{
			name:     "gRPC service not found",
			message:  "Service not found",
			expected: true,
		},
		// Negative cases
		{
			name:     "Empty message",
			message:  "",
			expected: false,
		},
		{
			name:     "Generic error",
			message:  "Internal server error",
			expected: false,
		},
		{
			name:     "Connection error",
			message:  "Connection refused",
			expected: false,
		},
		{
			name:     "Timeout error",
			message:  "Request timeout",
			expected: false,
		},
		// Mixed case sensitivity
		{
			name:     "Mixed case method not found",
			message:  "METHOD NOT FOUND",
			expected: true,
		},
		{
			name:     "Sentence with method not found",
			message:  "The requested method not found in the current API version",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := common.IsUnsupportedMethodMessage(tt.message)
			require.Equal(t, tt.expected, result, "Message: %s", tt.message)
		})
	}
}

func TestIsUnsupportedMethodError(t *testing.T) {
	t.Run("Nil error returns false", func(t *testing.T) {
		require.False(t, IsUnsupportedMethodError(nil))
	})

	t.Run("Error message patterns", func(t *testing.T) {
		tests := []struct {
			name     string
			err      error
			expected bool
		}{
			{
				name:     "Method not found error",
				err:      errors.New("method not found"),
				expected: true,
			},
			{
				name:     "Generic error",
				err:      errors.New("internal server error"),
				expected: false,
			},
			{
				name:     "Wrapped method not found",
				err:      fmt.Errorf("request failed: %w", errors.New("method not found")),
				expected: true,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result := IsUnsupportedMethodError(tt.err)
				require.Equal(t, tt.expected, result)
			})
		}
	})

	t.Run("HTTP status codes", func(t *testing.T) {
		tests := []struct {
			name     string
			err      rpcclient.HTTPError
			expected bool
		}{
			{
				name:     "404 Not Found",
				err:      rpcclient.HTTPError{StatusCode: 404, Status: "404 Not Found"},
				expected: true,
			},
			{
				name:     "405 Method Not Allowed",
				err:      rpcclient.HTTPError{StatusCode: 405, Status: "405 Method Not Allowed"},
				expected: true,
			},
			{
				name:     "500 Internal Server Error",
				err:      rpcclient.HTTPError{StatusCode: 500, Status: "500 Internal Server Error"},
				expected: false,
			},
			{
				name:     "200 OK",
				err:      rpcclient.HTTPError{StatusCode: 200, Status: "200 OK"},
				expected: false,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result := IsUnsupportedMethodError(tt.err)
				require.Equal(t, tt.expected, result)
			})
		}
	})

	t.Run("gRPC status codes", func(t *testing.T) {
		tests := []struct {
			name     string
			err      error
			expected bool
		}{
			{
				name:     "Unimplemented status",
				err:      status.Error(codes.Unimplemented, "method not implemented"),
				expected: true,
			},
			{
				name:     "NotFound status",
				err:      status.Error(codes.NotFound, "resource not available"),
				expected: false,
			},
			{
				name:     "Internal status",
				err:      status.Error(codes.Internal, "internal error"),
				expected: false,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result := IsUnsupportedMethodError(tt.err)
				require.Equal(t, tt.expected, result)
			})
		}
	})

	t.Run("JSON-RPC error recovery", func(t *testing.T) {
		// This tests the TryRecoverNodeErrorFromClientError path
		// Creating a mock HTTP error with JSON-RPC error body
		jsonRPCError := `{"jsonrpc":"2.0","error":{"code":-32601,"message":"Method not found"},"id":1}`
		httpErr := rpcclient.HTTPError{
			StatusCode: 200,
			Status:     "200 OK",
			Body:       []byte(jsonRPCError),
		}

		result := IsUnsupportedMethodError(httpErr)
		require.True(t, result, "Should detect JSON-RPC method not found error code")
	})

	t.Run("Combined error scenarios", func(t *testing.T) {
		// Test a gRPC error with method not found message
		err := status.Error(codes.Internal, "method not found")
		require.True(t, IsUnsupportedMethodError(err), "Should detect by message even with different status code")

		// Test HTTP error with method not found only in body (not in error message)
		httpErr := rpcclient.HTTPError{
			StatusCode: 200,
			Status:     "200 OK",
			Body:       []byte("Method not found: eth_newMethod"),
		}
		// This returns true because the HTTPError.Error() includes the body content
		// and the body contains "method not found" as a substring
		require.True(t, IsUnsupportedMethodError(httpErr))
	})
}

func TestErrorConstants(t *testing.T) {
	// Verify constant values haven't changed
	require.Equal(t, "method not found", common.JSONRPCMethodNotFound)
	require.Equal(t, "method not supported", common.JSONRPCMethodNotSupported)
	require.Equal(t, "unknown method", common.JSONRPCUnknownMethod)
	require.Equal(t, "method does not exist", common.JSONRPCMethodDoesNotExist)
	require.Equal(t, "invalid method", common.JSONRPCInvalidMethod)
	require.Equal(t, "-32601", common.JSONRPCErrorCode)

	require.Equal(t, "endpoint not found", common.RESTEndpointNotFound)
	require.Equal(t, "route not found", common.RESTRouteNotFound)
	require.Equal(t, "path not found", common.RESTPathNotFound)
	require.Equal(t, "method not allowed", common.RESTMethodNotAllowed)

	require.Equal(t, "method not implemented", common.GRPCMethodNotImplemented)
	require.Equal(t, "unimplemented", common.GRPCUnimplemented)
	require.Equal(t, "service not found", common.GRPCServiceNotFound)

	require.Equal(t, 404, common.HTTPStatusNotFound)
	require.Equal(t, 405, common.HTTPStatusMethodNotAllowed)
	require.Equal(t, -32601, common.JSONRPCMethodNotFoundCode)
}

// Benchmark tests to ensure performance
func BenchmarkIsUnsupportedMethodMessage(b *testing.B) {
	testMessages := []string{
		"method not found",
		"internal server error",
		"The requested method 'eth_someMethod' does not exist",
		"404 Not Found",
		"Connection timeout",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, msg := range testMessages {
			_ = common.IsUnsupportedMethodMessage(msg)
		}
	}
}

func BenchmarkIsUnsupportedMethodErrorMessageBytes(b *testing.B) {
	testMessages := [][]byte{
		[]byte("method not found"),
		[]byte("internal server error"),
		[]byte("The requested method 'eth_someMethod' does not exist"),
		[]byte("404 Not Found"),
		[]byte("Connection timeout"),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, msg := range testMessages {
			_ = common.IsUnsupportedMethodErrorMessageBytes(msg)
		}
	}
}

// BenchmarkIsUnsupportedMethodErrorMessageLongString tests performance with realistic response sizes
func BenchmarkIsUnsupportedMethodErrorMessageLongString(b *testing.B) {
	// Simulate a realistic JSON-RPC error response
	longError := `{"jsonrpc":"2.0","id":1,"error":{"code":-32601,"message":"The method eth_someUnsupportedMethod does not exist/is not available. See available methods at https://docs.example.com/api","data":null}}`
	longErrorBytes := []byte(longError)

	b.Run("String", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = common.IsUnsupportedMethodMessage(longError)
		}
	})

	b.Run("Bytes", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = common.IsUnsupportedMethodErrorMessageBytes(longErrorBytes)
		}
	})

	// Test with non-matching long string (worst case - checks all patterns)
	longNonMatch := `{"jsonrpc":"2.0","id":1,"result":"0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef"}`
	longNonMatchBytes := []byte(longNonMatch)

	b.Run("String_NoMatch", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = common.IsUnsupportedMethodMessage(longNonMatch)
		}
	})

	b.Run("Bytes_NoMatch", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = common.IsUnsupportedMethodErrorMessageBytes(longNonMatchBytes)
		}
	})
}

func BenchmarkIsUnsupportedMethodError(b *testing.B) {
	testErrors := []error{
		errors.New("method not found"),
		rpcclient.HTTPError{StatusCode: 404},
		status.Error(codes.Unimplemented, "not implemented"),
		errors.New("generic error"),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, err := range testErrors {
			_ = IsUnsupportedMethodError(err)
		}
	}
}

// NEW TEST: Verifies Issue #1 fix - smart contract errors should NOT be classified as unsupported
// This is a critical test for preventing false positives on smart contract reverts
func TestIsUnsupportedMethodErrorMessage_SmartContractErrors(t *testing.T) {
	tests := []struct {
		name     string
		message  string
		expected bool
	}{
		// CRITICAL: Smart contract errors should NOT match (Issue #1 fix)
		{
			name:     "Smart contract NFT not found",
			message:  "execution reverted: NFT not found",
			expected: false,
		},
		{
			name:     "Smart contract User not found",
			message:  "execution reverted: User not found",
			expected: false,
		},
		{
			name:     "Smart contract Token not found",
			message:  "execution reverted: Token not found",
			expected: false,
		},
		{
			name:     "Smart contract identity not found",
			message:  "execution reverted: identity not found",
			expected: false,
		},
		{
			name:     "Smart contract IdentityRegistry specific",
			message:  "execution reverted: IdentityRegistry: identity not found",
			expected: false,
		},
		{
			name:     "Smart contract Record not found",
			message:  "execution reverted: Record not found in database",
			expected: false,
		},
		{
			name:     "Generic not found without execution reverted",
			message:  "user not found",
			expected: false,
		},
		{
			name:     "Item not found",
			message:  "item not found",
			expected: false,
		},
		// Verify actual unsupported methods still work correctly
		{
			name:     "Actual method not found",
			message:  "method not found",
			expected: true,
		},
		{
			name:     "Actual endpoint not found",
			message:  "endpoint not found",
			expected: true,
		},
		{
			name:     "Actual route not found",
			message:  "route not found",
			expected: true,
		},
		{
			name:     "JSON-RPC method not supported",
			message:  "method not supported",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := common.IsUnsupportedMethodMessage(tt.message)
			require.Equal(t, tt.expected, got, "Message: %s", tt.message)
		})
	}
}
