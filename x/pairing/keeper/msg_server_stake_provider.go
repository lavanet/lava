package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/x/pairing/types"
)

func (k msgServer) StakeProvider(goCtx context.Context, msg *types.MsgStakeProvider) (*types.MsgStakeProviderResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if err := msg.ValidateBasic(); err != nil {
		return &types.MsgStakeProviderResponse{}, err
	}
	// stakes a new provider entry
	err := k.Keeper.StakeNewEntry(ctx, msg.Creator, msg.Validator, msg.ChainID, msg.Amount, msg.Endpoints, msg.Geolocation, msg.Moniker, msg.DelegateLimit, msg.DelegateCommission)

	return &types.MsgStakeProviderResponse{}, err
}
