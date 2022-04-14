package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/x/pairing/types"
)

func (k msgServer) UnstakeProvider(goCtx context.Context, msg *types.MsgUnstakeProvider) (*types.MsgUnstakeProviderResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// TODO: Handling the message
	_ = ctx

	return &types.MsgUnstakeProviderResponse{}, nil
}
