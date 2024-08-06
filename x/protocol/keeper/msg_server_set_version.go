package keeper

import (
	"context"

	sdkerrors "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/lavanet/lava/v2/x/protocol/types"
)

func (k msgServer) SetVersion(goCtx context.Context, msg *types.MsgSetVersion) (*types.MsgSetVersionResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if msg.Authority != k.authority {
		sdkerrors.Wrapf(govtypes.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.authority, msg.Authority)
	}

	params := k.GetParams(ctx)
	params.Version = *msg.Version
	if err := params.Validate(); err != nil {
		return &types.MsgSetVersionResponse{}, err
	}

	k.SetParams(ctx, params)

	return &types.MsgSetVersionResponse{}, nil
}
