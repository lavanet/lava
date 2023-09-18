package keeper_test

import (
	"testing"

	testkeeper "github.com/lavanet/lava/testutil/keeper"
	"github.com/lavanet/lava/x/dualstaking/types"
	"github.com/stretchr/testify/require"
)

func TestGetParams(t *testing.T) {
	k, ctx := testkeeper.DualstakingKeeper(t)
	params := types.DefaultParams()

	k.SetParams(ctx, params)

	require.EqualValues(t, params, k.GetParams(ctx))
}
