package keeper

import (
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
		k.Providers(ctx),
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

// Providers returns the Providers param
func (k Keeper) Providers(ctx sdk.Context) (res types.Providers) {
	k.paramstore.Get(ctx, types.KeyProviders, &res)
	return
}
