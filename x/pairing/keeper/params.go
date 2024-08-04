package keeper

import (
	"strconv"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/v2/utils"
	"github.com/lavanet/lava/v2/x/pairing/types"
)

// GetParams get all parameters as types.Params
func (k Keeper) GetParams(ctx sdk.Context) types.Params {
	return types.NewParams(
		k.EpochBlocksOverlap(ctx),
		k.QoSWeight(ctx),
		k.RecommendedEpochNumToCollectPayment(ctx),
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

func (k Keeper) SetRecommendedEpochNumToCollectPayment(ctx sdk.Context, val uint64) {
	k.paramstore.Set(ctx, types.KeyRecommendedEpochNumToCollectPayment, val)
}
