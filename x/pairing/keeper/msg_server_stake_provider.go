package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/v2/utils"
	"github.com/lavanet/lava/v2/x/pairing/types"
)

func (k msgServer) StakeProvider(goCtx context.Context, msg *types.MsgStakeProvider) (*types.MsgStakeProviderResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if err := utils.ValidateCoins(ctx, k.stakingKeeper.BondDenom(ctx), msg.DelegateLimit, true); err != nil {
		return &types.MsgStakeProviderResponse{}, err
	}

	if err := utils.ValidateCoins(ctx, k.stakingKeeper.BondDenom(ctx), msg.Amount, false); err != nil {
		return &types.MsgStakeProviderResponse{}, err
	}

	if err := msg.ValidateBasic(); err != nil {
		return &types.MsgStakeProviderResponse{}, err
	}

	description, err := msg.Description.EnsureLength()
	if err != nil {
		return &types.MsgStakeProviderResponse{}, err
	}

	// stakes a new provider entry
	err = k.Keeper.StakeNewEntry(ctx, msg.Validator, msg.Creator, msg.ChainID, msg.Amount, msg.Endpoints, msg.Geolocation, msg.DelegateLimit, msg.DelegateCommission, msg.Address, description)

	return &types.MsgStakeProviderResponse{}, err
}
