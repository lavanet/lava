package chainlib

import (
	"testing"

	"github.com/lavanet/lava/v5/protocol/common"
	"github.com/stretchr/testify/require"
)

func TestIsBlockNotAvailableError(t *testing.T) {
	tests := []struct {
		name     string
		response []byte
		expected bool
	}{
		{
			name:     "Solana -32004 block not available",
			response: []byte(`{"jsonrpc":"2.0","id":1,"error":{"code":-32004,"message":"Block not available for slot 12345"}}`),
			expected: true,
		},
		{
			name:     "Solana -32004 minimal error",
			response: []byte(`{"error":{"code":-32004}}`),
			expected: true,
		},
		{
			name:     "different error code -32001",
			response: []byte(`{"jsonrpc":"2.0","id":1,"error":{"code":-32001,"message":"Block cleaned up"}}`),
			expected: false,
		},
		{
			name:     "different error code -32007",
			response: []byte(`{"jsonrpc":"2.0","id":1,"error":{"code":-32007,"message":"Slot skipped"}}`),
			expected: false,
		},
		{
			name:     "success response with result",
			response: []byte(`{"jsonrpc":"2.0","id":1,"result":{"blockhash":"abc123","slot":100}}`),
			expected: false,
		},
		{
			name:     "empty response",
			response: []byte{},
			expected: false,
		},
		{
			name:     "nil response",
			response: nil,
			expected: false,
		},
		{
			name:     "invalid JSON",
			response: []byte(`not json`),
			expected: false,
		},
		{
			name:     "no error field",
			response: []byte(`{"jsonrpc":"2.0","id":1}`),
			expected: false,
		},
		{
			name:     "EVM -32004 method not supported (same code different chain)",
			response: []byte(`{"jsonrpc":"2.0","id":1,"error":{"code":-32004,"message":"Method not supported"}}`),
			expected: true, // The function only checks the code, chain filtering is done by the caller
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsBlockNotAvailableError(tt.response)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestIsSolanaFamily(t *testing.T) {
	tests := []struct {
		chainID  string
		expected bool
	}{
		{"SOLANA", true},
		{"SOLANAT", true},
		{"KOII", true},
		{"KOIIT", true},
		{"ETH1", false},
		{"COSMOSHUB", false},
		{"BTC", false},
		{"NEAR", false},
		{"solana", false}, // case-sensitive
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.chainID, func(t *testing.T) {
			result := common.IsSolanaFamily(tt.chainID)
			require.Equal(t, tt.expected, result)
		})
	}
}
