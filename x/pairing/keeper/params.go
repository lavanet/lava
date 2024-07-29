package keeper

import (
	"strconv"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/utils"
	"github.com/lavanet/lava/x/pairing/types"
)

// GetParams get all parameters as types.Params
func (k Keeper) GetParams(ctx sdk.Context) types.Params {
	return types.NewParams(
		k.EpochBlocksOverlap(ctx),
		k.QoSWeight(ctx),
		k.RecommendedEpochNumToCollectPayment(ctx),
		k.ReputationVarianceStabilizationPeriod(ctx),
		k.ReputationLatencyOverSyncFactor(ctx),
		k.ReputationHalfLifeFactor(ctx),
		k.ReputationRelayFailureCost(ctx),
	)
}

// SetParams set the params
func (k Keeper) SetParams(ctx sdk.Context, params types.Params) {
	k.paramstore.SetParamSet(ctx, &params)
}

func (k Keeper) EpochBlocksOverlap(ctx sdk.Context) uint64 {
	var res uint64
	k.paramstore.Get(ctx, types.KeyEpochBlocksOverlap, &res)

	epochBlocks, err := k.epochStorageKeeper.EpochBlocks(ctx, uint64(ctx.BlockHeight()))
	if err != nil {
		utils.LavaFormatError("could not get epochBlocks", err,
			utils.Attribute{Key: "block", Value: strconv.FormatInt(ctx.BlockHeight(), 10)},
		)

		return res
	}

	if epochBlocks < res {
		res = epochBlocks
	}

	return res
}

func (k Keeper) QoSWeight(ctx sdk.Context) (res sdk.Dec) {
	k.paramstore.Get(ctx, types.KeyQoSWeight, &res)
	return
}

// RecommendedEpochNumToCollectPayment returns the RecommendedEpochNumToCollectPayment param
func (k Keeper) RecommendedEpochNumToCollectPayment(ctx sdk.Context) (res uint64) {
	k.paramstore.Get(ctx, types.KeyRecommendedEpochNumToCollectPayment, &res)
	return
}

// ReputationVarianceStabilizationPeriod returns the ReputationVarianceStabilizationPeriod param
func (k Keeper) ReputationVarianceStabilizationPeriod(ctx sdk.Context) (res uint64) {
	k.paramstore.Get(ctx, types.KeyReputationVarianceStabilizationPeriod, &res)
	return
}

// ReputationLatencyOverSyncFactor returns the ReputationLatencyOverSyncFactor param
func (k Keeper) ReputationLatencyOverSyncFactor(ctx sdk.Context) (res math.LegacyDec) {
	k.paramstore.Get(ctx, types.KeyReputationLatencyOverSyncFactor, &res)
	return
}

// ReputationHalfLifeFactor returns the ReputationHalfLifeFactor param
func (k Keeper) ReputationHalfLifeFactor(ctx sdk.Context) (res uint64) {
	k.paramstore.Get(ctx, types.KeyReputationHalfLifeFactor, &res)
	return
}

// ReputationRelayFailureCost returns the ReputationRelayFailureCost param
func (k Keeper) ReputationRelayFailureCost(ctx sdk.Context) (res uint64) {
	k.paramstore.Get(ctx, types.KeyReputationRelayFailureCost, &res)
	return
}
