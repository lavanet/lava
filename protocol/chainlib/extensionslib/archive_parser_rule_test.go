package extensionslib

import (
	"testing"

	spectypes "github.com/lavanet/lava/v5/x/spec/types"
	"github.com/stretchr/testify/require"
)

// mockExtensionsChainMessage implements ExtensionsChainMessage for testing
type mockExtensionsChainMessage struct {
	latestBlock   int64
	earliestBlock int64
}

func (m *mockExtensionsChainMessage) RequestedBlock() (latest int64, earliest int64) {
	return m.latestBlock, m.earliestBlock
}

func (m *mockExtensionsChainMessage) SetExtension(extension *spectypes.Extension) {
	// Mock implementation - not needed for these tests
}

func TestArchiveParserRule_isPassingRule(t *testing.T) {
	tests := []struct {
		name             string
		extension        *spectypes.Extension
		latestBlock      uint64
		requestedBlock   int64
		expectedResult   bool
		expectedLogParts []string // Parts that should appear in logs
	}{
		{
			name: "EARLIEST_BLOCK always triggers archive",
			extension: &spectypes.Extension{
				Name: "archive",
				Rule: &spectypes.Rule{
					Block: 1000,
				},
			},
			latestBlock:      2000,
			requestedBlock:   -3, // EARLIEST_BLOCK
			expectedResult:   true,
			expectedLogParts: []string{"Negative block check", "isEarliestBlock", "true"},
		},
		{
			name: "Latest block is 0 forces archive",
			extension: &spectypes.Extension{
				Name: "archive",
				Rule: &spectypes.Rule{
					Block: 1000,
				},
			},
			latestBlock:      0,
			requestedBlock:   500,
			expectedResult:   true,
			expectedLogParts: []string{"Latest block is 0", "forcing archive"},
		},
		{
			name: "Requested block >= latest block returns false",
			extension: &spectypes.Extension{
				Name: "archive",
				Rule: &spectypes.Rule{
					Block: 1000,
				},
			},
			latestBlock:      1000,
			requestedBlock:   1500, // Future block
			expectedResult:   false,
			expectedLogParts: []string{"Requested block >= latest block", "no archive needed"},
		},
		{
			name: "Block old enough triggers archive (normal case)",
			extension: &spectypes.Extension{
				Name: "archive",
				Rule: &spectypes.Rule{
					Block: 1000,
				},
			},
			latestBlock:      3000,
			requestedBlock:   1500, // 3000 - 1000 = 2000, 1500 < 2000, so archive
			expectedResult:   true,
			expectedLogParts: []string{"Checking configured block threshold", "isOldEnough", "true"},
		},
		{
			name: "Block not old enough returns false",
			extension: &spectypes.Extension{
				Name: "archive",
				Rule: &spectypes.Rule{
					Block: 1000,
				},
			},
			latestBlock:      3000,
			requestedBlock:   2100, // 3000 - 1000 = 2000, 2100 > 2000, no archive
			expectedResult:   false,
			expectedLogParts: []string{"Checking configured block threshold", "isOldEnough", "false"},
		},
		{
			name: "Latest block <= rule block prevents underflow",
			extension: &spectypes.Extension{
				Name: "archive",
				Rule: &spectypes.Rule{
					Block: 3000,
				},
			},
			latestBlock:      2000, // Less than rule block
			requestedBlock:   1000,
			expectedResult:   false,
			expectedLogParts: []string{"Latest block <= rule block", "no archive needed"},
		},
		{
			name: "No rule configured returns false",
			extension: &spectypes.Extension{
				Name: "archive",
				Rule: nil,
			},
			latestBlock:      3000,
			requestedBlock:   1000,
			expectedResult:   false,
			expectedLogParts: []string{"No rule configured", "no archive"},
		},
		{
			name: "Rule block is 0 returns false",
			extension: &spectypes.Extension{
				Name: "archive",
				Rule: &spectypes.Rule{
					Block: 0,
				},
			},
			latestBlock:      3000,
			requestedBlock:   1000,
			expectedResult:   false,
			expectedLogParts: []string{"rule block is 0", "no archive"},
		},
		{
			name: "Normal block request within threshold",
			extension: &spectypes.Extension{
				Name: "archive",
				Rule: &spectypes.Rule{
					Block: 1000,
				},
			},
			latestBlock:      3000,
			requestedBlock:   2500, // Within recent blocks
			expectedResult:   false,
			expectedLogParts: []string{"Checking configured block threshold", "isOldEnough", "false"},
		},
		{
			name: "Edge case: block exactly at threshold",
			extension: &spectypes.Extension{
				Name: "archive",
				Rule: &spectypes.Rule{
					Block: 1000,
				},
			},
			latestBlock:      3000,
			requestedBlock:   2000,  // Exactly at threshold (3000 - 1000 = 2000)
			expectedResult:   false, // Should be false because 2000 is not < 2000
			expectedLogParts: []string{"Checking configured block threshold", "isOldEnough", "false"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockMessage := &mockExtensionsChainMessage{
				latestBlock:   tt.requestedBlock,
				earliestBlock: tt.requestedBlock,
			}

			rule := ArchiveParserRule{extension: tt.extension}
			result := rule.isPassingRule(mockMessage, tt.latestBlock)

			require.Equal(t, tt.expectedResult, result, "ArchiveParserRule result mismatch")

			// Note: We can't easily test the log output in unit tests without more complex setup
			// The logs would be captured by the logging system in actual usage
		})
	}
}

// TestArchiveParserRule_UnderflowProtection specifically tests the underflow fix
func TestArchiveParserRule_UnderflowProtection(t *testing.T) {
	// This test specifically ensures the uint64 underflow bug is fixed
	extension := &spectypes.Extension{
		Name: "archive",
		Rule: &spectypes.Rule{
			Block: 5000, // Rule block is larger than latest block
		},
	}

	mockMessage := &mockExtensionsChainMessage{
		latestBlock:   1000,
		earliestBlock: 1000,
	}

	rule := ArchiveParserRule{extension: extension}

	// Before the fix, this would incorrectly return true due to uint64 underflow
	// latestBlock (1000) - ruleBlock (5000) = underflow → large uint64
	// Now it should correctly return false
	result := rule.isPassingRule(mockMessage, 1000)
	require.False(t, result, "Underflow protection should prevent incorrect archive triggering")
}
