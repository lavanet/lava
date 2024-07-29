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
	require.EqualValues(t, params.EpochBlocksOverlap, k.EpochBlocksOverlap(ctx))
	require.EqualValues(t, params.RecommendedEpochNumToCollectPayment, k.RecommendedEpochNumToCollectPayment(ctx))
	require.EqualValues(t, params.ReputationVarianceStabilizationPeriod, k.ReputationVarianceStabilizationPeriod(ctx))
	require.EqualValues(t, params.ReputationLatencyOverSyncFactor, k.ReputationLatencyOverSyncFactor(ctx))
	require.EqualValues(t, params.ReputationHalfLifeFactor, k.ReputationHalfLifeFactor(ctx))
	require.EqualValues(t, params.ReputationRelayFailureCost, k.ReputationRelayFailureCost(ctx))
}
