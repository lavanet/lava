package chainlib

import (
	"errors"
	"testing"

	"github.com/lavanet/lava/v5/protocol/common"
	"github.com/stretchr/testify/require"
)

func TestShouldRetryError(t *testing.T) {
	tests := []struct {
		name        string
		err         error
		shouldRetry bool
	}{
		{
			name:        "Nil error should not retry",
			err:         nil,
			shouldRetry: false,
		},
		{
			name:        "Unsupported method error should not retry",
			err:         NewUnsupportedMethodError(errors.New("method not found"), "eth_test"),
			shouldRetry: false,
		},
		{
			name:        "Method not found message should not retry",
			err:         errors.New("method not found"),
			shouldRetry: false,
		},
		{
			name:        "Generic error should allow retry",
			err:         errors.New("connection timeout"),
			shouldRetry: true,
		},
		{
			name:        "Network error should allow retry",
			err:         errors.New("network unreachable"),
			shouldRetry: true,
		},
		// Solana non-retryable error tests
		{
			name:        "Solana non-retryable error type should not retry",
			err:         NewSolanaNonRetryableError(errors.New("missing in long-term storage")),
			shouldRetry: false,
		},
		{
			name:        "Solana -32009 missing in long-term storage should not retry",
			err:         errors.New("Slot 397535724 was skipped, or missing in long-term storage"),
			shouldRetry: false,
		},
		// Solana retryable errors - these SHOULD retry
		{
			name:        "Solana -32007 ledger jump SHOULD retry (another provider may have data)",
			err:         errors.New("Slot 397535724 was skipped, or missing due to ledger jump to recent snapshot"),
			shouldRetry: true,
		},
		{
			name:        "Solana slot was skipped without storage detail SHOULD retry",
			err:         errors.New("Slot was skipped"),
			shouldRetry: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ShouldRetryError(tt.err)
			require.Equal(t, tt.shouldRetry, result, "ShouldRetryError result mismatch")
		})
	}
}

func TestIsUnsupportedMethodErrorType(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "Nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "UnsupportedMethodError type",
			err:      NewUnsupportedMethodError(errors.New("method not found"), "eth_test"),
			expected: true,
		},
		{
			name:     "Generic error",
			err:      errors.New("some other error"),
			expected: false,
		},
		{
			name:     "Wrapped UnsupportedMethodError",
			err:      NewUnsupportedMethodError(errors.New("method not found"), ""),
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsUnsupportedMethodErrorType(tt.err)
			require.Equal(t, tt.expected, result, "IsUnsupportedMethodErrorType result mismatch")
		})
	}
}

func TestIsSolanaNonRetryableErrorType(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "Nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "SolanaNonRetryableError type",
			err:      NewSolanaNonRetryableError(errors.New("missing in long-term storage")),
			expected: true,
		},
		{
			name:     "Generic error",
			err:      errors.New("some other error"),
			expected: false,
		},
		{
			name:     "Wrapped SolanaNonRetryableError",
			err:      NewSolanaNonRetryableError(errors.New("Slot 397535724 was skipped, or missing in long-term storage")),
			expected: true,
		},
		{
			name:     "UnsupportedMethodError is not SolanaNonRetryableError",
			err:      NewUnsupportedMethodError(errors.New("method not found"), "eth_test"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsSolanaNonRetryableErrorType(tt.err)
			require.Equal(t, tt.expected, result, "IsSolanaNonRetryableErrorType result mismatch")
		})
	}
}

func TestIsSolanaNonRetryableError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "Nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "missing in long-term storage message (-32009)",
			err:      errors.New("Slot 397535724 was skipped, or missing in long-term storage"),
			expected: true,
		},
		{
			name:     "Generic error",
			err:      errors.New("connection timeout"),
			expected: false,
		},
		{
			name:     "Method not found should not match",
			err:      errors.New("method not found"),
			expected: false,
		},
		// These should NOT match (retryable errors)
		{
			name:     "ledger jump message (-32007) should NOT match - is retryable",
			err:      errors.New("Slot 397535724 was skipped, or missing due to ledger jump to recent snapshot"),
			expected: false,
		},
		{
			name:     "slot was skipped without storage detail should NOT match - is retryable",
			err:      errors.New("Slot was skipped"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsSolanaNonRetryableError(tt.err)
			require.Equal(t, tt.expected, result, "IsSolanaNonRetryableError result mismatch")
		})
	}
}

func TestSolanaNonRetryableError_ErrorMessage(t *testing.T) {
	originalErr := errors.New("Slot 397535724 was skipped, or missing in long-term storage")
	wrappedErr := NewSolanaNonRetryableError(originalErr)

	require.Contains(t, wrappedErr.Error(), "solana non-retryable error")
	require.Contains(t, wrappedErr.Error(), "missing in long-term storage")

	// Test Unwrap
	require.Equal(t, originalErr, wrappedErr.Unwrap())
}

func TestSolanaNonRetryableErrorCodes(t *testing.T) {
	// Verify the error codes are correctly defined
	// These are the JSON-RPC error codes that should NOT trigger retries
	t.Run("InvalidParamsCode is -32602", func(t *testing.T) {
		require.Equal(t, -32602, int(common.JSONRPCInvalidParamsCode))
	})

	t.Run("MissingInLongTermStorageCode is -32009", func(t *testing.T) {
		require.Equal(t, -32009, int(common.SolanaMissingInLongTermStorageCode))
	})
}
