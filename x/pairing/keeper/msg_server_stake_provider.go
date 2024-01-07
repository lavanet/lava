package keeper

import (
	"context"

	sdkerrors "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/utils"
	"github.com/lavanet/lava/x/pairing/types"
)

func (k msgServer) StakeProvider(goCtx context.Context, msg *types.MsgStakeProvider) (*types.MsgStakeProviderResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if _, err := sdk.AccAddressFromBech32(msg.Creator); err != nil {
		return &types.MsgStakeProviderResponse{}, sdkerrors.Wrapf(types.InvalidCreatorAddressError, "Invalid creator address (%s)", err.Error())
	}

	if err := utils.ValidateCoins(ctx, k.stakingKeeper.BondDenom(ctx), msg.DelegateLimit, true); err != nil {
		return &types.MsgStakeProviderResponse{}, err
	}

	if err := utils.ValidateCoins(ctx, k.stakingKeeper.BondDenom(ctx), msg.Amount, false); err != nil {
		return &types.MsgStakeProviderResponse{}, err
	}

	if err := msg.ValidateBasic(); err != nil {
		return &types.MsgStakeProviderResponse{}, err
	}

	// stakes a new provider entry
	err := k.Keeper.StakeNewEntry(ctx, msg.Validator, msg.Creator, msg.ChainID, msg.Amount, msg.Endpoints, msg.Geolocation, msg.Moniker, msg.DelegateLimit, msg.DelegateCommission)

	return &types.MsgStakeProviderResponse{}, err
}
