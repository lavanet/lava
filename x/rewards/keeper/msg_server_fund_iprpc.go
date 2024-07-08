package keeper

import (
	"context"
	"strconv"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/utils"
	"github.com/lavanet/lava/x/rewards/types"
)

func (k msgServer) FundIprpc(goCtx context.Context, msg *types.MsgFundIprpc) (*types.MsgFundIprpcResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	err := msg.ValidateBasic()
	if err != nil {
		return &types.MsgFundIprpcResponse{}, err
	}

	err = k.Keeper.FundIprpc(ctx, msg.Creator, msg.Duration, msg.Amounts, msg.Spec)
	if err == nil {
		logger := k.Keeper.Logger(ctx)
		details := map[string]string{
			"creator":  msg.Creator,
			"spec":     msg.Spec,
			"duration": strconv.FormatUint(msg.Duration, 10),
			"amounts":  msg.Amounts.String(),
		}
		utils.LogLavaEvent(ctx, logger, types.FundIprpcEventName, details, "Funded IPRPC pool successfully")
	}

	return &types.MsgFundIprpcResponse{}, err
}
