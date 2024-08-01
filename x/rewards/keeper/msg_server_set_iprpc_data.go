package keeper

import (
	"context"
	"strings"

	sdkerrors "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/lavanet/lava/v2/utils"
	"github.com/lavanet/lava/v2/x/rewards/types"
)

func (k msgServer) SetIprpcData(goCtx context.Context, msg *types.MsgSetIprpcData) (*types.MsgSetIprpcDataResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	err := msg.ValidateBasic()
	if err != nil {
		return &types.MsgSetIprpcDataResponse{}, err
	}

	if msg.Authority != k.authority {
		return &types.MsgSetIprpcDataResponse{}, sdkerrors.Wrapf(govtypes.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.authority, msg.Authority)
	}

	err = k.Keeper.SetIprpcData(ctx, msg.MinIprpcCost, msg.IprpcSubscriptions)
	if err == nil {
		logger := k.Keeper.Logger(ctx)
		details := map[string]string{
			"min_cost":      msg.MinIprpcCost.String(),
			"subscriptions": strings.Join(msg.IprpcSubscriptions, ","),
		}
		utils.LogLavaEvent(ctx, logger, types.SetIprpcDataEventName, details, "Set IPRPC Data")
	}

	return &types.MsgSetIprpcDataResponse{}, err
}
