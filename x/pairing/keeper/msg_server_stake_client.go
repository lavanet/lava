package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/x/pairing/types"
)

func (k msgServer) StakeClient(goCtx context.Context, msg *types.MsgStakeClient) (*types.MsgStakeClientResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// stakes a new client entry
	err := k.Keeper.StakeNewEntry(ctx, false, msg.Creator, msg.ChainID, msg.Amount, nil, msg.Geolocation, msg.Vrfpk)

	return &types.MsgStakeClientResponse{}, err
}
