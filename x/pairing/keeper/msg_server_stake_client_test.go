package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/testutil/common"
	testkeeper "github.com/lavanet/lava/testutil/keeper"
	"github.com/lavanet/lava/x/pairing/types"
	"github.com/stretchr/testify/require"
)

func TestStakeClientPairingimmediately(t *testing.T) {
	servers, keepers, ctx := testkeeper.InitAllKeepers(t)

	// init keepers state
	var balance int64 = 10000
	consumer := common.CreateNewAccount(ctx, *keepers, balance)
	provider1 := common.CreateNewAccount(ctx, *keepers, balance)
	provider2 := common.CreateNewAccount(ctx, *keepers, balance)

	spec := common.CreateMockSpec()
	keepers.Spec.SetSpec(sdk.UnwrapSDKContext(ctx), spec)

	plan := common.CreateMockPlan()
	keepers.Plans.AddPlan(sdk.UnwrapSDKContext(ctx), plan)

	stake := balance / 10
	ctx = testkeeper.AdvanceEpoch(ctx, keepers)
	common.StakeAccount(t, ctx, *keepers, *servers, provider1, spec, stake)
	common.StakeAccount(t, ctx, *keepers, *servers, provider2, spec, stake)

	ctx = testkeeper.AdvanceEpoch(ctx, keepers)
	common.BuySubscription(t, ctx, *keepers, *servers, consumer, plan.Index)

	ctx = testkeeper.AdvanceBlock(ctx, keepers)
	ctx = testkeeper.AdvanceBlock(ctx, keepers)
	ctx = testkeeper.AdvanceBlock(ctx, keepers)

	epoch := keepers.Epochstorage.GetEpochStart(sdk.UnwrapSDKContext(ctx))

	// check pairing in the same epoch
	_, err := keepers.Pairing.VerifyPairingData(sdk.UnwrapSDKContext(ctx), spec.Index, consumer.Addr, epoch)
	require.Nil(t, err)

	pairing, err := keepers.Pairing.GetPairingForClient(sdk.UnwrapSDKContext(ctx), spec.Index, consumer.Addr)
	require.Nil(t, err)

	_, err = keepers.Pairing.VerifyPairing(ctx, &types.QueryVerifyPairingRequest{ChainID: spec.Index, Client: consumer.Addr.String(), Provider: pairing[0].Address, Block: epoch})
	require.Nil(t, err)
}
