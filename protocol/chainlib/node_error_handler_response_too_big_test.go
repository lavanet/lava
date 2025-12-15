package chainlib

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIsNoRetryNodeErrorByMessage(t *testing.T) {
	tests := []struct {
		name          string
		errorMessage  string
		shouldNoRetry bool
	}{
		{
			name:          "Exact match - response is too big",
			errorMessage:  "response is too big",
			shouldNoRetry: true,
		},
		{
			name:          "Capitalized - Response is too big",
			errorMessage:  "Response is too big",
			shouldNoRetry: true,
		},
		{
			name:          "All caps - RESPONSE IS TOO BIG",
			errorMessage:  "RESPONSE IS TOO BIG",
			shouldNoRetry: true,
		},
		{
			name:          "Mixed case - ReSpOnSe Is ToO BiG",
			errorMessage:  "ReSpOnSe Is ToO BiG",
			shouldNoRetry: true,
		},
		{
			name:          "Within longer message",
			errorMessage:  "Error: response is too big for processing",
			shouldNoRetry: true,
		},
		{
			name:          "With data - from real log",
			errorMessage:  `{"code":-32008,"message":"Response is too big","data":"Exceeded max limit of 167772160"}`,
			shouldNoRetry: true,
		},
		{
			name:          "Empty message",
			errorMessage:  "",
			shouldNoRetry: false,
		},
		{
			name:          "Generic server error - should retry",
			errorMessage:  "Internal server error",
			shouldNoRetry: false,
		},
		{
			name:          "Timeout error - should retry",
			errorMessage:  "Request timeout",
			shouldNoRetry: false,
		},
		{
			name:          "Connection error - should retry",
			errorMessage:  "Connection refused",
			shouldNoRetry: false,
		},
		{
			name:          "Method not found - different category",
			errorMessage:  "method not found",
			shouldNoRetry: false,
		},
		{
			name:          "Invalid params - not handled yet",
			errorMessage:  "Invalid params",
			shouldNoRetry: false,
		},
		{
			name:          "Similar but different - response size",
			errorMessage:  "response size exceeded",
			shouldNoRetry: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsNoRetryNodeErrorByMessage(tt.errorMessage)
			require.Equal(t, tt.shouldNoRetry, result,
				"Message '%s' should be no-retry=%v", tt.errorMessage, tt.shouldNoRetry)
		})
	}
}

func TestShouldRetryErrorWithNoRetryNodeError(t *testing.T) {
	tests := []struct {
		name        string
		error       error
		shouldRetry bool
	}{
		{
			name:        "Nil error should not retry",
			error:       nil,
			shouldRetry: false,
		},
		{
			name:        "Error with 'response is too big' message should not retry",
			error:       errors.New("response is too big"),
			shouldRetry: false,
		},
		{
			name:        "Error with 'Response is too big' (capitalized) should not retry",
			error:       errors.New("Response is too big"),
			shouldRetry: false,
		},
		{
			name:        "UnsupportedMethodError should not retry",
			error:       NewUnsupportedMethodError(errors.New("method not found"), "eth_testMethod"),
			shouldRetry: false,
		},
		{
			name:        "Error with unsupported pattern should not retry",
			error:       errors.New("method not found"),
			shouldRetry: false,
		},
		{
			name:        "Generic error should retry",
			error:       errors.New("Internal server error"),
			shouldRetry: true,
		},
		{
			name:        "Timeout error should retry",
			error:       errors.New("Request timeout"),
			shouldRetry: true,
		},
		{
			name:        "Connection error should retry",
			error:       errors.New("Connection refused"),
			shouldRetry: true,
		},
		{
			name:        "Unknown error should retry",
			error:       errors.New("some random error"),
			shouldRetry: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ShouldRetryError(tt.error)
			require.Equal(t, tt.shouldRetry, result,
				"Error '%v' should retry=%v", tt.error, tt.shouldRetry)
		})
	}
}

func TestResponseTooBigPatternConstant(t *testing.T) {
	t.Run("Pattern constant value", func(t *testing.T) {
		require.Equal(t, "response is too big", ResponseTooBigPattern,
			"ResponseTooBigPattern should match the expected error message")
	})

	t.Run("Pattern constant is used correctly", func(t *testing.T) {
		// Test that the constant works in the function
		result := IsNoRetryNodeErrorByMessage(ResponseTooBigPattern)
		require.True(t, result, "ResponseTooBigPattern should be detected as no-retry error")
	})
}

func TestIsNoRetryNodeErrorByMessageEdgeCases(t *testing.T) {
	t.Run("Message with only whitespace", func(t *testing.T) {
		result := IsNoRetryNodeErrorByMessage("   ")
		require.False(t, result, "Whitespace-only message should not be no-retry")
	})

	t.Run("Message with newlines", func(t *testing.T) {
		result := IsNoRetryNodeErrorByMessage("error\nresponse is too big\n")
		require.True(t, result, "Message with newlines should still match")
	})

	t.Run("Message with tabs", func(t *testing.T) {
		result := IsNoRetryNodeErrorByMessage("error\tresponse is too big\t")
		require.True(t, result, "Message with tabs should still match")
	})

	t.Run("Partial match should not work", func(t *testing.T) {
		result := IsNoRetryNodeErrorByMessage("response is too")
		require.False(t, result, "Incomplete pattern should not match")
	})

	t.Run("Extra spacing in pattern", func(t *testing.T) {
		result := IsNoRetryNodeErrorByMessage("response  is  too  big")
		require.False(t, result, "Extra spacing should not match (exact pattern check)")
	})
}
