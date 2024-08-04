package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/v2/x/protocol/types"
)

// GetParams get all parameters as types.Params
func (k Keeper) GetParams(ctx sdk.Context) types.Params {
	return types.NewParams(
		k.Version(ctx),
	)
}

// SetParams set the params
func (k Keeper) SetParams(ctx sdk.Context, params types.Params) {
	k.paramstore.SetParamSet(ctx, &params)
}

// Version returns the Version param
func (k Keeper) Version(ctx sdk.Context) (res types.Version) {
	if !k.paramstore.Has(ctx, types.KeyVersion) {
		params := types.DefaultParams()
		k.paramstore.SetParamSet(ctx, &params)
	}
	k.paramstore.Get(ctx, types.KeyVersion, &res)
	return
}
