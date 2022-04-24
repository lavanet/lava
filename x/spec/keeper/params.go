package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/x/spec/types"
)

// GetParams get all parameters as types.Params
func (k Keeper) GetParams(ctx sdk.Context) types.Params {
	return types.NewParams(
		k.GeolocationCount(ctx),
	)
}

// SetParams set the params
func (k Keeper) SetParams(ctx sdk.Context, params types.Params) {
	k.paramstore.SetParamSet(ctx, &params)
}

// GeolocationCount returns the GeolocationCount param
func (k Keeper) GeolocationCount(ctx sdk.Context) (res uint64) {
	k.paramstore.Get(ctx, types.KeyGeolocationCount, &res)
	return
}
