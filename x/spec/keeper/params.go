package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/v2/x/spec/types"
)

// GetParams get all parameters as types.Params
func (k Keeper) GetParams(ctx sdk.Context) types.Params {
	return types.NewParams(
		k.MaxCU(ctx),
		k.AllowlistedExpeditedMsgs(ctx),
	)
}

// SetParams set the params
func (k Keeper) SetParams(ctx sdk.Context, params types.Params) {
	k.paramstore.SetParamSet(ctx, &params)
}

// MaxCU returns the MaxCU param
func (k Keeper) MaxCU(ctx sdk.Context) (res uint64) {
	k.paramstore.Get(ctx, types.KeyMaxCU, &res)
	return
}

// AllowlistedExpeditedMsgs returns the AllowlistedExpeditedMsgs param
func (k Keeper) AllowlistedExpeditedMsgs(ctx sdk.Context) (res []string) {
	k.paramstore.Get(ctx, types.KeyallowlistExpeditedMsgs, &res)
	return
}

func (k Keeper) AllowlistReset(ctx sdk.Context) {
	k.paramstore.Set(ctx, types.KeyallowlistExpeditedMsgs, []string{})
}
