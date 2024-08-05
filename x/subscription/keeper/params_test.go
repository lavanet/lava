package keeper_test

import (
	"testing"

	testkeeper "github.com/lavanet/lava/v2/testutil/keeper"
	"github.com/lavanet/lava/v2/x/subscription/types"
	"github.com/stretchr/testify/require"
)

func TestGetParams(t *testing.T) {
	k, ctx := testkeeper.SubscriptionKeeper(t)
	params := types.DefaultParams()

	k.SetParams(ctx, params)

	require.EqualValues(t, params, k.GetParams(ctx))
}
