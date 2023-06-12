package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/utils/sigs"
	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
	"github.com/stretchr/testify/require"
)

func TestSubscriptionExpire(t *testing.T) {
	ts := setupTestStruct(t, 1)
	keeper := ts.keepers.Subscription

	_, account := sigs.GenerateFloatingKey()
	coins := sdk.NewCoins(sdk.NewCoin(epochstoragetypes.TokenDenom, sdk.NewInt(10000)))
	ts.keepers.BankKeeper.SetBalance(ts.ctx, account, coins)

	creator := account.String()
	consumer := account.String()

	blocksToSave, err := ts.keepers.Epochstorage.BlocksToSave(ts.ctx, uint64(ts.ctx.BlockHeight()))
	require.Nil(t, err)

	err = keeper.CreateSubscription(ts.ctx, creator, consumer, "mockPlan1", 1)
	require.Nil(t, err)

	// fill memory
	for uint64(ts.ctx.BlockHeight()) < blocksToSave {
		ts.advanceBlock()
	}

	sub, found := keeper.GetSubscription(ts.ctx, account.String())
	require.True(t, found)

	// fast-forward one month
	ts.expireSubscription(sub)

	// subscription remains searchable, but with zero duration and CU
	sub, found = keeper.GetSubscription(ts.ctx, account.String())
	require.True(t, found)
	require.Equal(t, uint64(0), sub.DurationLeft)
	require.Equal(t, uint64(0), sub.MonthCuLeft)

	// fill memory
	for uint64(ts.ctx.BlockHeight()) < sub.PrevExpiryBlock+blocksToSave {
		ts.advanceBlock()
	}
	ts.advanceEpoch()

	// the subscription is finally gone
	_, found = keeper.GetSubscription(ts.ctx, account.String())
	require.False(t, found)
}
