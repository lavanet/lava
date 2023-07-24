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
	providerAcct, providerAddr := ts.AddAccount(common.PROVIDER, 0, balance)

	err := ts.StakeProvider(providerAddr, ts.spec, balance/2)
	require.Nil(t, err)

	ts.AdvanceEpoch()

	unstakeHoldBlocks := ts.Keepers.Epochstorage.UnstakeHoldBlocks(ts.Ctx, ts.BlockHeight())
	unstakeHoldBlocksStatic := ts.Keepers.Epochstorage.UnstakeHoldBlocksStatic(ts.Ctx, ts.BlockHeight())

	_, err = ts.TxPairingUnstakeProvider(providerAddr, ts.spec.Index)
	require.Nil(t, err)

	ts.AdvanceBlocks(unstakeHoldBlocks)

	_, found, _ := ts.Keepers.Epochstorage.UnstakeEntryByAddress(ts.Ctx, providerAcct.Addr)
	require.True(t, found)

	ts.AdvanceBlocks(unstakeHoldBlocksStatic - unstakeHoldBlocks)

	_, found, _ = ts.Keepers.Epochstorage.UnstakeEntryByAddress(ts.Ctx, providerAcct.Addr)
	require.False(t, found)
}
