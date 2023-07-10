package keeper

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/x/protocol/types"
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
		fmt.Println("LAVA_lava_ doesnt has")
		params := types.DefaultParams()
		k.paramstore.SetParamSet(ctx, &params)
	}
	fmt.Println("LAVA_lava_ has")
	k.paramstore.Get(ctx, types.KeyVersion, &res)
	return
}
