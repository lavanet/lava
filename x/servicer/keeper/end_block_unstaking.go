package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k Keeper) checkUnstakingForCommit(goCtx context.Context) error {
	ctx := sdk.UnwrapSDKContext(goCtx)

	_ = ctx
	return nil
}
