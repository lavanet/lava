package keeper_test

import (
	"testing"

	testkeeper "github.com/lavanet/lava/testutil/keeper"
	"github.com/lavanet/lava/x/servicer/types"
	"github.com/stretchr/testify/require"
)

func TestGetParams(t *testing.T) {
	k, ctx := testkeeper.ServicerKeeper(t)
	params := types.DefaultParams()

	k.SetParams(ctx, params)

	require.EqualValues(t, params, k.GetParams(ctx))
	require.EqualValues(t, params.MinStake, k.MinStake(ctx))
	require.EqualValues(t, params.CoinsPerCU, k.CoinsPerCU(ctx))
	require.EqualValues(t, params.UnstakeHoldBlocks, k.UnstakeHoldBlocks(ctx))
	require.EqualValues(t, params.FraudStakeSlashingFactor, k.FraudStakeSlashingFactor(ctx))
	require.EqualValues(t, params.FraudSlashingAmount, k.FraudSlashingAmount(ctx))
	require.EqualValues(t, params.ServicersToPairCount, k.ServicersToPairCount(ctx))
	require.EqualValues(t, params.SessionBlocksOverlap, k.SessionBlocksOverlap(ctx))
}
