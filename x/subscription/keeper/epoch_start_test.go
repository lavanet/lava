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

	block := uint64(ts.ctx.BlockHeight())
	timestamp := ts.ctx.BlockTime()

	// fill memory
	for uint64(ts.ctx.BlockHeight()) < blocksToSave {
		ts.advanceBlock()
	}

	_, found := keeper.GetSubscription(ts.ctx, account.String())
	require.True(t, found)

	err = keeper.ChargeComputeUnitsToSubscription(ts.ctx, account.String(), block, 10)
	require.Nil(t, err)

	// fast-forward one month
	ts.advanceMonths(timestamp, 1)
	_, found = keeper.GetSubscription(ts.ctx, creator)
	require.False(t, found)

	// subscription no longer searchable, but can still charge for previous usage
	_, found = keeper.GetSubscription(ts.ctx, account.String())
	require.False(t, found)

	err = keeper.ChargeComputeUnitsToSubscription(ts.ctx, account.String(), block, 10)
	require.Nil(t, err)

	ts.advanceUntilStale()

	// subscription no longer charge-able for previous usage
	err = keeper.ChargeComputeUnitsToSubscription(ts.ctx, account.String(), block, 10)
	require.NotNil(t, err)
}
