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

	// advance block to reach time > 0
	ts.advanceBlock()

	err := keeper.CreateSubscription(ts.ctx, creator, consumer, "mockPlan1", 1, "")
	require.Nil(t, err)

	sub, found := keeper.GetSubscription(ts.ctx, account.String())
	require.True(t, found)

	// fast-forward one month
	ts.expireSubscription(sub)

	// subscription remains searchable, but with zero duration and CU
	sub, found = keeper.GetSubscription(ts.ctx, account.String())
	require.True(t, found)
	require.Equal(t, uint64(0), sub.DurationLeft)
	require.Equal(t, uint64(0), sub.MonthCuLeft)

	// fast-forward another month
	sub = ts.expireSubscription(sub)

	// the subscription is finally gone
	_, found = keeper.GetSubscription(ts.ctx, account.String())
	require.False(t, found)
}
