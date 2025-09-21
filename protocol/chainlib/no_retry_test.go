package chainlib

import (
	"errors"
	"testing"

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
