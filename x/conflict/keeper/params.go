package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/x/conflict/types"
)

// GetParams get all parameters as types.Params
func (k Keeper) GetParams(ctx sdk.Context) types.Params {
	return types.NewParams(
		k.MajorityPercent(ctx),
	)
}

// SetParams set the params
func (k Keeper) SetParams(ctx sdk.Context, params types.Params) {
	k.paramstore.SetParamSet(ctx, &params)
}

// MajorityPercent returns the MajorityPercent param
func (k Keeper) MajorityPercent(ctx sdk.Context) (res string) {
	k.paramstore.Get(ctx, types.KeyMajorityPercent, &res)
	return
}
