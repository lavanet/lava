package rpcsmartrouter

import (
	"testing"

	"github.com/lavanet/lava/v5/protocol/common"
	"github.com/stretchr/testify/require"
)

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
