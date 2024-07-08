package keeper

import (
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/x/rewards/types"
)

// GetParams get all parameters as types.Params
func (k Keeper) GetParams(ctx sdk.Context) types.Params {
	return types.NewParams(
		k.MinBondedTarget(ctx),
		k.MaxBondedTarget(ctx),
		k.LowFactor(ctx),
		k.LeftoverBurnRate(ctx),
		k.MaxRewardBoost(ctx),
		k.ValidatorsSubscriptionParticipation(ctx),
		k.IbcIprpcExpiration(ctx),
	)
}

// SetParams set the params
func (k Keeper) SetParams(ctx sdk.Context, params types.Params) {
	k.paramstore.SetParamSet(ctx, &params)
}

// MinBondedTarget returns the MinBondedTarget param
func (k Keeper) MinBondedTarget(ctx sdk.Context) (res sdk.Dec) {
	k.paramstore.Get(ctx, types.KeyMinBondedTarget, &res)
	return
}

// MaxBondedTarget returns the MaxBondedTarget param
func (k Keeper) MaxBondedTarget(ctx sdk.Context) (res sdk.Dec) {
	k.paramstore.Get(ctx, types.KeyMaxBondedTarget, &res)
	return
}

// LowFactor returns the LowFactor param
func (k Keeper) LowFactor(ctx sdk.Context) (res sdk.Dec) {
	k.paramstore.Get(ctx, types.KeyLowFactor, &res)
	return
}

// LeftoverBurnRate returns the LeftoverBurnRate param
func (k Keeper) LeftoverBurnRate(ctx sdk.Context) (res sdk.Dec) {
	k.paramstore.Get(ctx, types.KeyLeftoverBurnRate, &res)
	return
}

// MaxRewardBoost returns the MaxRewardBoost param
func (k Keeper) MaxRewardBoost(ctx sdk.Context) (res uint64) {
	k.paramstore.Get(ctx, types.KeyMaxRewardBoost, &res)
	return
}

// ValidatorsSubscriptionParticipation returns the ValidatorsSubscriptionParticipation param
func (k Keeper) ValidatorsSubscriptionParticipation(ctx sdk.Context) (res sdk.Dec) {
	k.paramstore.Get(ctx, types.KeyValidatorsSubscriptionParticipation, &res)
	return
}

// IbcIprpcExpiration returns the IbcIprpcExpiration param
func (k Keeper) IbcIprpcExpiration(ctx sdk.Context) (res time.Duration) {
	k.paramstore.Get(ctx, types.KeyIbcIprpcExpiration, &res)
	return
}
