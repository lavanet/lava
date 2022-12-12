package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/x/pairing/types"
)

func (k msgServer) StakeProvider(goCtx context.Context, msg *types.MsgStakeProvider) (*types.MsgStakeProviderResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// stakes a new provider entry
	err := k.Keeper.StakeNewEntry(ctx, true, msg.Creator, msg.ChainID, msg.Amount, msg.Endpoints, msg.Geolocation, "")

	return &types.MsgStakeProviderResponse{}, err
}
