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

		require.Contains(t, err.Error(), "eth_someMethod")
		require.Contains(t, err.Error(), "unsupported method")
	})

	t.Run("Error formatting without method name", func(t *testing.T) {
		originalErr := errors.New("original error")
		err := NewUnsupportedMethodError(originalErr, "")

		require.Contains(t, err.Error(), "unsupported method")
	})

	t.Run("IsUnsupportedMethodErrorType detects the error", func(t *testing.T) {
		originalErr := errors.New("original error")
		err := NewUnsupportedMethodError(originalErr, "eth_call")

		require.True(t, IsUnsupportedMethodErrorType(err))
		require.Contains(t, err.Error(), "eth_call")
	})

	t.Run("Unwrap returns LavaError", func(t *testing.T) {
		originalErr := errors.New("original error")
		err := NewUnsupportedMethodError(originalErr, "method")

		unwrapped := errors.Unwrap(err)
		require.NotNil(t, unwrapped)
		require.True(t, errors.Is(err, common.LavaErrorNodeMethodNotFound))
	})

	t.Run("Error message contains method name when provided", func(t *testing.T) {
		originalErr := errors.New("original error")
		err := NewUnsupportedMethodError(originalErr, "test-method")
		require.Contains(t, err.Error(), "test-method")
	})

	t.Run("Error message has no method name when empty", func(t *testing.T) {
		originalErr := errors.New("original error")
		err := NewUnsupportedMethodError(originalErr, "")
		require.Contains(t, err.Error(), "unsupported method")
	})
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

// TestIsUnsupportedMethodError_SmartContractErrors verifies that smart contract errors
// are NOT classified as unsupported methods (prevents false positives on reverts).
func TestIsUnsupportedMethodError_SmartContractErrors(t *testing.T) {
	tests := []struct {
		name     string
		message  string
		expected bool
	}{
		// CRITICAL: Smart contract errors should NOT match
		{"Smart contract NFT not found", "execution reverted: NFT not found", false},
		{"Smart contract User not found", "execution reverted: User not found", false},
		{"Smart contract Token not found", "execution reverted: Token not found", false},
		{"Smart contract identity not found", "execution reverted: identity not found", false},
		{"Smart contract IdentityRegistry specific", "execution reverted: IdentityRegistry: identity not found", false},
		{"Smart contract Record not found", "execution reverted: Record not found in database", false},
		{"Generic not found without execution reverted", "user not found", false},
		{"Item not found", "item not found", false},
		// Verify actual unsupported methods still work correctly
		{"Actual method not found", "method not found", true},
		{"Actual endpoint not found", "endpoint not found", true},
		{"Actual route not found", "route not found", true},
		// Note: "method not supported" (-32004) is retryable (another provider may support it), not SubCategoryUnsupportedMethod
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsUnsupportedMethodError(errors.New(tt.message))
			require.Equal(t, tt.expected, got, "Message: %s", tt.message)
		})
	}
}
