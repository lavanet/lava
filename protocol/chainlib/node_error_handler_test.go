package chainlib

import (
	"errors"
	"fmt"
	"testing"

	"github.com/lavanet/lava/v5/protocol/chainlib/chainproxy/rpcclient"
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
			result := IsUnsupportedMethodErrorMessage(tt.message)
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

func TestGetUnsupportedMethodPatterns(t *testing.T) {
	patterns := GetUnsupportedMethodPatterns()

	// Verify all pattern categories exist
	require.Contains(t, patterns, "json-rpc")
	require.Contains(t, patterns, "rest")
	require.Contains(t, patterns, "grpc")

	// Verify JSON-RPC patterns
	jsonRPCPatterns := patterns["json-rpc"]
	require.Contains(t, jsonRPCPatterns, JSONRPCMethodNotFound)
	require.Contains(t, jsonRPCPatterns, JSONRPCMethodNotSupported)
	require.Contains(t, jsonRPCPatterns, JSONRPCUnknownMethod)
	require.Contains(t, jsonRPCPatterns, JSONRPCMethodDoesNotExist)
	require.Contains(t, jsonRPCPatterns, JSONRPCInvalidMethod)
	require.Contains(t, jsonRPCPatterns, JSONRPCErrorCode)

	// Verify REST patterns
	restPatterns := patterns["rest"]
	require.Contains(t, restPatterns, RESTEndpointNotFound)
	require.Contains(t, restPatterns, RESTRouteNotFound)
	require.Contains(t, restPatterns, RESTPathNotFound)
	require.Contains(t, restPatterns, RESTMethodNotAllowed)

	// Verify gRPC patterns
	grpcPatterns := patterns["grpc"]
	require.Contains(t, grpcPatterns, GRPCMethodNotImplemented)
	require.Contains(t, grpcPatterns, GRPCUnimplemented)
	require.Contains(t, grpcPatterns, GRPCServiceNotFound)
}

func TestErrorConstants(t *testing.T) {
	// Verify constant values haven't changed
	require.Equal(t, "method not found", JSONRPCMethodNotFound)
	require.Equal(t, "method not supported", JSONRPCMethodNotSupported)
	require.Equal(t, "unknown method", JSONRPCUnknownMethod)
	require.Equal(t, "method does not exist", JSONRPCMethodDoesNotExist)
	require.Equal(t, "invalid method", JSONRPCInvalidMethod)
	require.Equal(t, "-32601", JSONRPCErrorCode)

	require.Equal(t, "endpoint not found", RESTEndpointNotFound)
	require.Equal(t, "route not found", RESTRouteNotFound)
	require.Equal(t, "path not found", RESTPathNotFound)
	require.Equal(t, "method not allowed", RESTMethodNotAllowed)

	require.Equal(t, "method not implemented", GRPCMethodNotImplemented)
	require.Equal(t, "unimplemented", GRPCUnimplemented)
	require.Equal(t, "service not found", GRPCServiceNotFound)

	require.Equal(t, 404, HTTPStatusNotFound)
	require.Equal(t, 405, HTTPStatusMethodNotAllowed)
	require.Equal(t, -32601, JSONRPCMethodNotFoundCode)
}

// Benchmark tests to ensure performance
func BenchmarkIsUnsupportedMethodErrorMessage(b *testing.B) {
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
			_ = IsUnsupportedMethodErrorMessage(msg)
		}
	}
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
