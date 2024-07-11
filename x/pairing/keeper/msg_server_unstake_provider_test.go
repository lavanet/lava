package keeper_test

import (
	"testing"

	"github.com/lavanet/lava/testutil/common"
	"github.com/stretchr/testify/require"
)

// TestVaultProviderUnstake tests that only the vault address can unstake.
// Scenarios:
// 1. unstake with vault -> should work
// 2. try with provider -> should fail
func TestVaultProviderUnstake(t *testing.T) {
	ts := newTester(t)
	ts.setupForPayments(1, 0, 0)

	acc, _ := ts.GetAccount(common.PROVIDER, 0)
	provider := acc.Addr.String()
	vault := acc.GetVaultAddr()

	tests := []struct {
		name    string
		creator string
		valid   bool
	}{
		{"provider unstakes", provider, false},
		{"vault unstakes", vault, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ts.TxPairingUnstakeProvider(tt.creator, ts.spec.Index)
			if tt.valid {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}
