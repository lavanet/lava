package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/x/pairing/types"
)

// GetParams get all parameters as types.Params
func (k Keeper) GetParams(ctx sdk.Context) types.Params {
	return types.NewParams(
		k.MintCoinsPerCU(ctx),
		k.FraudStakeSlashingFactor(ctx),
		k.FraudSlashingAmount(ctx),
		k.EpochBlocksOverlap(ctx),
		k.UnpayLimit(ctx),
		k.SlashLimit(ctx),
		k.DataReliabilityReward(ctx),
		k.QoSWeight(ctx),
		k.RecommendedEpochNumToCollectPayment(ctx),
	)
}

// SetParams set the params
func (k Keeper) SetParams(ctx sdk.Context, params types.Params) {
	k.paramstore.SetParamSet(ctx, &params)
}

// MintCoinsPerCU returns the MintCoinsPerCU param
func (k Keeper) MintCoinsPerCU(ctx sdk.Context) (res sdk.Dec) {
	k.paramstore.Get(ctx, types.KeyMintCoinsPerCU, &res)
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

func (k Keeper) EpochBlocksOverlap(ctx sdk.Context) (res uint64) {
	k.paramstore.Get(ctx, types.KeyEpochBlocksOverlap, &res)
	return
}

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

// RecommendedEpochNumToCollectPayment returns the RecommendedEpochNumToCollectPayment param
func (k Keeper) RecommendedEpochNumToCollectPayment(ctx sdk.Context) (res uint64) {
	k.paramstore.Get(ctx, types.KeyRecommendedEpochNumToCollectPayment, &res)
	return
}

func (k Keeper) SetRecommendedEpochNumToCollectPayment(ctx sdk.Context, val uint64) {
	k.paramstore.Set(ctx, types.KeyRecommendedEpochNumToCollectPayment, val)
}
