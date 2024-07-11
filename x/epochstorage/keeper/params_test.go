package keeper_test

import (
	"testing"

	testkeeper "github.com/lavanet/lava/testutil/keeper"
	"github.com/lavanet/lava/x/epochstorage/types"
	"github.com/stretchr/testify/require"
)

func TestGetParams(t *testing.T) {
	k, ctx := testkeeper.EpochstorageKeeper(t)
	params := types.DefaultParams()

	k.SetParams(ctx, params)

	require.EqualValues(t, params, k.GetParams(ctx))
	require.EqualValues(t, params.EpochBlocks, k.EpochBlocksRaw(ctx))
	require.EqualValues(t, params.EpochsToSave, k.EpochsToSaveRaw(ctx))
}
