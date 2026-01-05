package common

import (
	"testing"
	
	"github.com/stretchr/testify/require"
)

func TestIsUnsupportedMethodMessage_AllPatterns(t *testing.T) {
	// Test all 14 patterns are detected
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

