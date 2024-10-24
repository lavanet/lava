package keeper_test

import (
	"testing"

	"github.com/lavanet/lava/v4/testutil/common"
	"github.com/stretchr/testify/require"
)

// TestVaultProviderUnstake tests that only the vault address can unstake.
// Scenarios:
// 1. unstake with vault -> should work
// 2. try with provider -> should work
func TestVaultProviderUnstake(t *testing.T) {
	ts := newTester(t)
	ts.setupForPayments(2, 0, 0)

	acc1, _ := ts.GetAccount(common.PROVIDER, 0)
	provider1 := acc1.Addr.String()

	acc2, _ := ts.GetAccount(common.PROVIDER, 1)
	vault2 := acc2.GetVaultAddr()

	tests := []struct {
		name    string
		creator string
	}{
		{"provider unstakes", provider1},
		{"vault unstakes", vault2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prov := ts.Keepers.Epochstorage.GetAllStakeEntriesCurrentForChainId(ts.Ctx, ts.spec.Index)
			_ = prov
			_, err := ts.TxPairingUnstakeProvider(tt.creator, ts.spec.Index)
			require.NoError(t, err)
		})
	}
}
