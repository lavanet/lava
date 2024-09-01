package keeper_test

import (
	"testing"

	testkeeper "github.com/lavanet/lava/v3/testutil/keeper"
	"github.com/lavanet/lava/v3/x/pairing/types"
	"github.com/stretchr/testify/require"
)

func TestGetParams(t *testing.T) {
	k, ctx := testkeeper.PairingKeeper(t)
	params := types.DefaultParams()

	k.SetParams(ctx, params)

	require.EqualValues(t, params, k.GetParams(ctx))
	require.EqualValues(t, params.EpochBlocksOverlap, k.EpochBlocksOverlap(ctx))
	require.EqualValues(t, params.RecommendedEpochNumToCollectPayment, k.RecommendedEpochNumToCollectPayment(ctx))
}
