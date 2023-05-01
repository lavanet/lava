package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/testutil/common"
	testkeeper "github.com/lavanet/lava/testutil/keeper"
	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
	"github.com/lavanet/lava/x/pairing/types"
	spectypes "github.com/lavanet/lava/x/spec/types"
	"github.com/stretchr/testify/require"
)

func TestUnstakeStaticProvider(t *testing.T) {
	servers, keepers, ctx := testkeeper.InitAllKeepers(t)

	spec := common.CreateMockSpec()
	spec.ProvidersTypes = spectypes.Spec_static
	keepers.Spec.SetSpec(sdk.UnwrapSDKContext(ctx), spec)

	balance := 5 * spec.MinStakeProvider.Amount.Int64()
	provider := common.CreateNewAccount(ctx, *keepers, balance)

	common.StakeAccount(t, ctx, *keepers, *servers, provider, spec, balance/2, true)
	ctx = testkeeper.AdvanceEpoch(ctx, keepers)

	unstakeHoldBlocks := keepers.Epochstorage.UnstakeHoldBlocks(sdk.UnwrapSDKContext(ctx), uint64(sdk.UnwrapSDKContext(ctx).BlockHeight()))
	unstakeHoldBlocksStatic := keepers.Epochstorage.UnstakeHoldBlocksStatic(sdk.UnwrapSDKContext(ctx), uint64(sdk.UnwrapSDKContext(ctx).BlockHeight()))

	_, err := servers.PairingServer.UnstakeProvider(ctx, &types.MsgUnstakeProvider{Creator: provider.Addr.String(), ChainID: spec.Index})
	require.Nil(t, err)

	ctx = testkeeper.AdvanceBlocks(ctx, keepers, int(unstakeHoldBlocks))

	_, found, _ := keepers.Epochstorage.UnstakeEntryByAddress(sdk.UnwrapSDKContext(ctx), epochstoragetypes.ProviderKey, provider.Addr)
	require.True(t, found)

	ctx = testkeeper.AdvanceBlocks(ctx, keepers, int(unstakeHoldBlocksStatic-unstakeHoldBlocks))

	_, found, _ = keepers.Epochstorage.UnstakeEntryByAddress(sdk.UnwrapSDKContext(ctx), epochstoragetypes.ProviderKey, provider.Addr)
	require.False(t, found)
}
