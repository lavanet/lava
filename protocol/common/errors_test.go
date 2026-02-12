package common

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIsUnsupportedMethodMessage_AllPatterns(t *testing.T) {
	// Test all 15 patterns are detected
	tests := []struct {
		name    string
		message string
		want    bool
	}{
		// JSON-RPC patterns (6)
		{"method not found", "method not found", true},
		{"method not supported", "method not supported", true},
		{"unknown method", "unknown method", true},
		{"method does not exist", "method does not exist", true},
		{"invalid method", "invalid method", true},
		{"-32601 code", "error: -32601", true},

		// REST patterns (4)
		{"endpoint not found", "endpoint not found", true},
		{"route not found", "route not found", true},
		{"path not found", "path not found", true},
		{"method not allowed", "method not allowed", true},

		// gRPC patterns (4)
		{"method not implemented", "method not implemented", true},
		{"unimplemented", "unimplemented", true},
		{"not implemented", "not implemented", true},
		{"service not found", "service not found", true},

		// Generic catch-all pattern (1) - catches "method X not supported" format
		{"not supported generic", "not supported", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsUnsupportedMethodMessage(tt.message)
			require.Equal(t, tt.want, got, "Pattern '%s' should be detected", tt.message)
		})
	}
}

func TestIsUnsupportedMethodMessage_CaseInsensitive(t *testing.T) {
	tests := []string{
		"METHOD NOT FOUND",
		"Method Not Found",
		"MeThOd NoT fOuNd",
		"UNIMPLEMENTED",
	}

	for _, msg := range tests {
		t.Run(msg, func(t *testing.T) {
			require.True(t, IsUnsupportedMethodMessage(msg), "Should be case insensitive")
		})
	}
}

func TestIsUnsupportedMethodMessage_PartialMatch(t *testing.T) {
	// Should match patterns within longer messages
	tests := []string{
		"Error: method not found in API",
		"RPC endpoint not found on server",
		"The service is unimplemented",
		"JSON-RPC error -32601: method not found",
		// Specific format: "method X not supported" (catches GenericNotSupported pattern)
		"method eth_call not supported",
		"method someMethod not supported",
		"feature not supported",
	}

	for _, msg := range tests {
		t.Run(msg, func(t *testing.T) {
			require.True(t, IsUnsupportedMethodMessage(msg), "Should match pattern in longer message")
		})
	}
}

func TestIsUnsupportedMethodMessage_SmartContractErrors(t *testing.T) {
	// CRITICAL: Test Issue #1 fix - smart contract errors should NOT match
	tests := []struct {
		name    string
		message string
	}{
		{"NFT not found", "execution reverted: NFT not found"},
		{"User not found", "execution reverted: User not found"},
		{"Token not found", "execution reverted: Token not found"},
		{"Identity not found", "execution reverted: identity not found"},
		{"IdentityRegistry specific", "execution reverted: IdentityRegistry: identity not found"},
		{"Record not found", "execution reverted: Record not found in database"},
		{"Generic not found", "user not found"}, // Without "execution reverted"
		{"Item not found", "item not found"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsUnsupportedMethodMessage(tt.message)
			require.False(t, got, "Smart contract error '%s' should NOT be classified as unsupported method", tt.message)
		})
	}
}

func TestIsUnsupportedMethodMessage_NegativeCases(t *testing.T) {
	tests := []string{
		"",
		"normal error",
		"internal server error",
		"timeout",
		"connection refused",
		"execution reverted",
		"execution reverted: some other error",
	}

	for _, msg := range tests {
		t.Run(msg, func(t *testing.T) {
			require.False(t, IsUnsupportedMethodMessage(msg))
		})
	}
}

// TestIsUnsupportedMethodErrorMessageBytes_AllPatterns tests the bytes version with all patterns
func TestIsUnsupportedMethodErrorMessageBytes_AllPatterns(t *testing.T) {
	// Test all 14 patterns are detected
	tests := []struct {
		name    string
		message []byte
		want    bool
	}{
		// JSON-RPC patterns (6)
		{"method not found", []byte("method not found"), true},
		{"method not supported", []byte("method not supported"), true},
		{"unknown method", []byte("unknown method"), true},
		{"method does not exist", []byte("method does not exist"), true},
		{"invalid method", []byte("invalid method"), true},
		{"-32601 code", []byte("error: -32601"), true},

		// REST patterns (4)
		{"endpoint not found", []byte("endpoint not found"), true},
		{"route not found", []byte("route not found"), true},
		{"path not found", []byte("path not found"), true},
		{"method not allowed", []byte("method not allowed"), true},

		// gRPC patterns (4)
		{"method not implemented", []byte("method not implemented"), true},
		{"unimplemented", []byte("unimplemented"), true},
		{"not implemented", []byte("not implemented"), true},
		{"service not found", []byte("service not found"), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsUnsupportedMethodErrorMessageBytes(tt.message)
			require.Equal(t, tt.want, got, "Pattern '%s' should be detected", string(tt.message))
		})
	}
}

func TestIsUnsupportedMethodErrorMessageBytes_CaseInsensitive(t *testing.T) {
	tests := [][]byte{
		[]byte("METHOD NOT FOUND"),
		[]byte("Method Not Found"),
		[]byte("MeThOd NoT fOuNd"),
		[]byte("UNIMPLEMENTED"),
	}

	for _, msg := range tests {
		t.Run(string(msg), func(t *testing.T) {
			require.True(t, IsUnsupportedMethodErrorMessageBytes(msg), "Should be case insensitive")
		})
	}
}

func TestIsUnsupportedMethodErrorMessageBytes_PartialMatch(t *testing.T) {
	// Should match patterns within longer messages
	tests := [][]byte{
		[]byte("Error: method not found in API"),
		[]byte("RPC endpoint not found on server"),
		[]byte("The service is unimplemented"),
		[]byte("JSON-RPC error -32601: method not found"),
	}

	for _, msg := range tests {
		t.Run(string(msg), func(t *testing.T) {
			require.True(t, IsUnsupportedMethodErrorMessageBytes(msg), "Should match pattern in longer message")
		})
	}
}

func TestIsUnsupportedMethodErrorMessageBytes_SmartContractErrors(t *testing.T) {
	// CRITICAL: Test Issue #1 fix - smart contract errors should NOT match
	tests := []struct {
		name    string
		message []byte
	}{
		{"NFT not found", []byte("execution reverted: NFT not found")},
		{"User not found", []byte("execution reverted: User not found")},
		{"Token not found", []byte("execution reverted: Token not found")},
		{"Identity not found", []byte("execution reverted: identity not found")},
		{"IdentityRegistry specific", []byte("execution reverted: IdentityRegistry: identity not found")},
		{"Record not found", []byte("execution reverted: Record not found in database")},
		{"Generic not found", []byte("user not found")}, // Without "execution reverted"
		{"Item not found", []byte("item not found")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsUnsupportedMethodErrorMessageBytes(tt.message)
			require.False(t, got, "Smart contract error '%s' should NOT be classified as unsupported method", string(tt.message))
		})
	}
}

func TestIsUnsupportedMethodErrorMessageBytes_NegativeCases(t *testing.T) {
	tests := [][]byte{
		nil,
		[]byte(""),
		[]byte("normal error"),
		[]byte("internal server error"),
		[]byte("timeout"),
		[]byte("connection refused"),
		[]byte("execution reverted"),
		[]byte("execution reverted: some other error"),
	}

	for _, msg := range tests {
		testName := string(msg)
		if msg == nil {
			testName = "nil"
		}
		t.Run(testName, func(t *testing.T) {
			require.False(t, IsUnsupportedMethodErrorMessageBytes(msg))
		})
	}
}

// TestEquivalence_StringVsBytes verifies that both functions produce identical results
func TestEquivalence_StringVsBytes(t *testing.T) {
	testCases := []string{
		// Positive cases
		"method not found",
		"METHOD NOT FOUND",
		"Method Not Found",
		"unknown method",
		"endpoint not found",
		"method not implemented",
		"unimplemented",
		"Error: method not found in API",
		"JSON-RPC error -32601: method not found",

		// Negative cases
		"",
		"normal error",
		"internal server error",
		"execution reverted: NFT not found",
		"user not found",
		"item not found",
	}

	for _, msg := range testCases {
		t.Run(msg, func(t *testing.T) {
			stringResult := IsUnsupportedMethodMessage(msg)
			bytesResult := IsUnsupportedMethodErrorMessageBytes([]byte(msg))

			require.Equal(t, stringResult, bytesResult,
				"String and bytes versions should produce identical results for message: %q", msg)
		})
	}
}

// TestIsUnsupportedMethodErrorMessageBytes_EmptyAndNil tests edge cases
func TestIsUnsupportedMethodErrorMessageBytes_EmptyAndNil(t *testing.T) {
	t.Run("nil slice", func(t *testing.T) {
		require.False(t, IsUnsupportedMethodErrorMessageBytes(nil))
	})

	t.Run("empty slice", func(t *testing.T) {
		require.False(t, IsUnsupportedMethodErrorMessageBytes([]byte{}))
	})

	t.Run("single byte", func(t *testing.T) {
		require.False(t, IsUnsupportedMethodErrorMessageBytes([]byte("a")))
	})
}

// TestIsUnsupportedMethodErrorMessageBytes_LengthOptimization verifies the length check optimization
func TestIsUnsupportedMethodErrorMessageBytes_LengthOptimization(t *testing.T) {
	// Test that patterns longer than the message are skipped (optimization)
	shortMsg := []byte("hi")
	require.False(t, IsUnsupportedMethodErrorMessageBytes(shortMsg))

	// Test that patterns shorter or equal to message are checked
	longMsg := []byte("this is a very long error message that contains method not found")
	require.True(t, IsUnsupportedMethodErrorMessageBytes(longMsg))
}
