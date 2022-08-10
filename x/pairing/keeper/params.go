package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/x/pairing/types"
)

// GetParams get all parameters as types.Params
func (k Keeper) GetParams(ctx sdk.Context) types.Params {
	return types.NewParams(
		k.MinStakeProvider(ctx),
		k.MinStakeClient(ctx),
		k.MintCoinsPerCU(ctx),
		k.BurnCoinsPerCU(ctx),
		k.FraudStakeSlashingFactor(ctx),
		k.FraudSlashingAmount(ctx),
		k.ServicersToPairCount(ctx),
		k.EpochBlocksOverlap(ctx),
		k.StakeToMaxCUList(ctx),
		k.UnpayLimit(ctx),
		k.SlashLimit(ctx),
		k.DataReliabilityReward(ctx),
		k.QoSWeight(ctx),
	)
}

// SetParams set the params
func (k Keeper) SetParams(ctx sdk.Context, params types.Params) {
	k.paramstore.SetParamSet(ctx, &params)
}

// MinStakeProvider returns the MinStakeProvider param
func (k Keeper) MinStakeProvider(ctx sdk.Context) (res sdk.Coin) {
	k.paramstore.Get(ctx, types.KeyMinStakeProvider, &res)
	return
}

// MinStakeClient returns the MinStakeClient param
func (k Keeper) MinStakeClient(ctx sdk.Context) (res sdk.Coin) {
	k.paramstore.Get(ctx, types.KeyMinStakeClient, &res)
	return
}

// MintCoinsPerCU returns the MintCoinsPerCU param
func (k Keeper) MintCoinsPerCU(ctx sdk.Context) (res sdk.Dec) {
	k.paramstore.Get(ctx, types.KeyMintCoinsPerCU, &res)
	return
}

// BurnCoinsPerCU returns the BurnCoinsPerCU param
func (k Keeper) BurnCoinsPerCU(ctx sdk.Context) (res sdk.Dec) {
	k.paramstore.Get(ctx, types.KeyBurnCoinsPerCU, &res)
	return
}

// FraudStakeSlashingFactor returns the FraudStakeSlashingFactor param
func (k Keeper) FraudStakeSlashingFactor(ctx sdk.Context) (res sdk.Dec) {
	k.paramstore.Get(ctx, types.KeyFraudStakeSlashingFactor, &res)
	return
}

// FraudSlashingAmount returns the FraudSlashingAmount param
func (k Keeper) FraudSlashingAmount(ctx sdk.Context) (res uint64) {
	k.paramstore.Get(ctx, types.KeyFraudSlashingAmount, &res)
	return
}

// ServicersToPairCount returns the ServicersToPairCount param
func (k Keeper) ServicersToPairCount(ctx sdk.Context) (res uint64) {
	k.paramstore.Get(ctx, types.KeyServicersToPairCount, &res)
	return
}

// EpochBlocksOverlap returns the EpochBlocksOverlap param
func (k Keeper) EpochBlocksOverlap(ctx sdk.Context) (res uint64) {
	k.paramstore.Get(ctx, types.KeyEpochBlocksOverlap, &res)
	return
}

// EpochBlocksOverlap returns the EpochBlocksOverlap param
func (k Keeper) StakeToMaxCUList(ctx sdk.Context) (res types.StakeToMaxCUList) {
	k.paramstore.Get(ctx, types.KeyStakeToMaxCUList, &res)
	return
}

// EpochBlocksOverlap returns the EpochBlocksOverlap param
func (k Keeper) UnpayLimit(ctx sdk.Context) (res sdk.Dec) {
	k.paramstore.Get(ctx, types.KeyUnpayLimit, &res)
	return
}
func (k Keeper) SlashLimit(ctx sdk.Context) (res sdk.Dec) {
	k.paramstore.Get(ctx, types.KeySlashLimit, &res)
	return
}

func (k Keeper) DataReliabilityReward(ctx sdk.Context) (res sdk.Dec) {
	k.paramstore.Get(ctx, types.KeyDataReliabilityReward, &res)
	return
}

func (k Keeper) QoSWeight(ctx sdk.Context) (res sdk.Dec) {
	k.paramstore.Get(ctx, types.KeyQoSWeight, &res)
	return
}

// func (k Keeper) FixateServicersToPair(ctx sdk.Context, block uint64) {
// 	latestParamChange := k.lates(ctx)
// 	if latestParamChange == 0 { // no change
// 		return
// 	}
// 	if latestParamChange > block {
// 		utils.LavaError(ctx, k.Logger(ctx), "invalid_latest_param_change", map[string]string{"error": "latestParamChange > block", "latestParamChange": strconv.FormatUint(latestParamChange, 10)}, "latest param change cant be in the future")
// 		return
// 	}
// 	earliestEpochStart := k.GetEarliestEpochStart(ctx) //this is the previous epoch start, before we update it to the current block
// 	if latestParamChange < earliestEpochStart {
// 		//latest param change is older than memory, so remove it
// 		k.paramstore.Set(ctx, types.KeyLatestParamChange, uint64(0))
// 		//clean up older fixated params, they no longer matter
// 		k.CleanOlderFixatedParams(ctx, 1) //everything after 0 is too old since there wasn't a param change in a while
// 		return
// 	}
// 	// we have a param change, is it in the last epoch?
// 	prevEpochStart, err := k.GetPreviousEpochStartForBlock(ctx, block)
// 	if err != nil {
// 		utils.LavaError(ctx, k.Logger(ctx), "GetPreviousEpochStartForBlock_pushFixation", map[string]string{"error": err.Error(), "block": strconv.FormatUint(block, 10)}, "can't get block in epoch")
// 	} else if latestParamChange > prevEpochStart {
// 		// this is a recent change so we need to move the current fixation backwards
// 		k.PushFixatedParams(ctx, block, earliestEpochStart)
// 	}
// }
