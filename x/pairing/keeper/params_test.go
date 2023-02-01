package keeper_test

import (
	"testing"

	testkeeper "github.com/lavanet/lava/testutil/keeper"
	"github.com/lavanet/lava/x/pairing/types"
	"github.com/stretchr/testify/require"
)

func TestGetParams(t *testing.T) {
	k, ctx := testkeeper.PairingKeeper(t)
	params := types.DefaultParams()

	k.SetParams(ctx, params)

	require.EqualValues(t, params, k.GetParams(ctx))
	require.EqualValues(t, params.MintCoinsPerCU, k.MintCoinsPerCU(ctx))
	require.EqualValues(t, params.BurnCoinsPerCU, k.BurnCoinsPerCU(ctx))
	require.EqualValues(t, params.FraudStakeSlashingFactor, k.FraudStakeSlashingFactor(ctx))
	require.EqualValues(t, params.FraudSlashingAmount, k.FraudSlashingAmount(ctx))
	require.EqualValues(t, params.ServicersToPairCount, k.ServicersToPairCountRaw(ctx))
	require.EqualValues(t, params.EpochBlocksOverlap, k.EpochBlocksOverlap(ctx))
	require.EqualValues(t, params.RecommendedEpochNumToCollectPayment, k.RecommendedEpochNumToCollectPayment(ctx))
}
