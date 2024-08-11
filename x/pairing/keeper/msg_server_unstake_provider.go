package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/v2/x/pairing/types"
)

func (k msgServer) UnstakeProvider(goCtx context.Context, msg *types.MsgUnstakeProvider) (*types.MsgUnstakeProviderResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	err := k.Keeper.UnstakeEntry(ctx, msg.Validator, msg.ChainID, msg.Creator, types.UnstakeDescriptionProviderUnstake)
	return &types.MsgUnstakeProviderResponse{}, err
}
