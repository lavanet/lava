package keeper_test

import (
	"testing"

	"github.com/lavanet/lava/testutil/common"
	spectypes "github.com/lavanet/lava/x/spec/types"
	"github.com/stretchr/testify/require"
)

func TestUnstakeStaticProvider(t *testing.T) {
	ts := newTester(t)

	// will overwrite the default "mock" spec
	ts.spec.ProvidersTypes = spectypes.Spec_static
	ts.AddSpec("mock", ts.spec)

	balance := 5 * ts.spec.MinStakeProvider.Amount.Int64()
	providerAcct, operator := ts.AddAccount(common.PROVIDER, 0, balance)

	err := ts.StakeProvider(operator, providerAcct.Vault.Addr.String(), ts.spec, balance/2)
	require.NoError(t, err)

	ts.AdvanceEpoch()

	unstakeHoldBlocks := ts.Keepers.Epochstorage.UnstakeHoldBlocks(ts.Ctx, ts.BlockHeight())
	unstakeHoldBlocksStatic := ts.Keepers.Epochstorage.UnstakeHoldBlocksStatic(ts.Ctx, ts.BlockHeight())

	_, err = ts.TxPairingUnstakeProvider(providerAcct.Vault.Addr.String(), ts.spec.Index)
	require.NoError(t, err)

	ts.AdvanceBlocks(unstakeHoldBlocks)

	_, found := ts.Keepers.Epochstorage.UnstakeEntryByAddress(ts.Ctx, operator)
	require.True(t, found)

	ts.AdvanceBlocks(unstakeHoldBlocksStatic - unstakeHoldBlocks)

	_, found = ts.Keepers.Epochstorage.UnstakeEntryByAddress(ts.Ctx, operator)
	require.False(t, found)
}

// TestVaultOperatorUnstake tests that only the vault address can unstake.
// Scenarios:
// 1. unstake with vault -> should work
// 2. try with operator -> should fail
func TestVaultOperatorUnstake(t *testing.T) {
	ts := newTester(t)
	ts.setupForPayments(1, 0, 0)

	acc, _ := ts.GetAccount(common.PROVIDER, 0)
	operator := acc.Addr.String()
	vault := acc.Vault.Addr.String()

	tests := []struct {
		name    string
		creator string
		valid   bool
	}{
		{"operator unstakes", operator, false},
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
