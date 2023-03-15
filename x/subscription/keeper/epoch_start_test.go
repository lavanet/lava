package keeper_test

import (
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/relayer/sigs"
	"github.com/lavanet/lava/testutil/common"
	testkeeper "github.com/lavanet/lava/testutil/keeper"
	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
	"github.com/stretchr/testify/require"
)

func TestSubscriptionExpire(t *testing.T) {
	_, keepers, _ctx := testkeeper.InitAllKeepers(t)
	ctx := sdk.UnwrapSDKContext(_ctx)

	keeper := keepers.Subscription
	bank := keepers.BankKeeper
	plansKeeper := keepers.Plans

	plan := common.CreateMockPlan()
	plansKeeper.AddPlan(ctx, plan)

	_, account := sigs.GenerateFloatingKey()
	coins := sdk.NewCoins(sdk.NewCoin(epochstoragetypes.TokenDenom, sdk.NewInt(10000)))
	bank.SetBalance(ctx, account, coins)

	creator := account.String()
	consumer := account.String()

	// advance block to reach time > 0
	_ctx = testkeeper.AdvanceBlock(_ctx, keepers, time.Minute)
	ctx = sdk.UnwrapSDKContext(_ctx)

	err := keeper.CreateSubscription(ctx, creator, consumer, "mockPlan", 1)
	require.Nil(t, err)

	sub, found := keeper.GetSubscription(ctx, account.String())
	require.True(t, found)

	// change expiration time to 1 second ago
	sub.ExpiryTime = uint64(ctx.BlockTime().Add(-time.Second).UTC().Unix())
	keeper.SetSubscription(ctx, sub)

	// trigger EpochStart() processing	
	keeper.EpochStart(ctx)

	sub, found = keeper.GetSubscription(ctx, account.String())
	require.False(t, found)
}
