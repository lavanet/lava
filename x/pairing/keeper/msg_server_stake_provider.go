package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/x/pairing/types"
)

func (k msgServer) StakeProvider(goCtx context.Context, msg *types.MsgStakeProvider) (*types.MsgStakeProviderResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// TODO: Handling the message
	_ = ctx

	return &types.MsgStakeProviderResponse{}, nil
}
