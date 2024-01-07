package keeper

import (
	"context"

	sdkerrors "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/x/pairing/types"
)

func (k msgServer) StakeProvider(goCtx context.Context, msg *types.MsgStakeProvider) (*types.MsgStakeProviderResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if msg.DelegateLimit.Denom != k.stakingKeeper.BondDenom(ctx) {
		_, err := sdk.AccAddressFromBech32(msg.Creator)
		return &types.MsgStakeProviderResponse{}, sdkerrors.Wrapf(types.DelegateLimitError, "Invalid coin (%s)", err.Error())
	}

	// stakes a new provider entry
	err := k.Keeper.StakeNewEntry(ctx, msg.Validator, msg.Creator, msg.ChainID, msg.Amount, msg.Endpoints, msg.Geolocation, msg.Moniker, msg.DelegateLimit, msg.DelegateCommission)

	return &types.MsgStakeProviderResponse{}, err
}
