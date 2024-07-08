package keeper

import (
	"context"
	"strconv"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/utils"
	"github.com/lavanet/lava/x/rewards/types"
)

func (k msgServer) CoverIbcIprpcFundCost(goCtx context.Context, msg *types.MsgCoverIbcIprpcFundCost) (*types.MsgCoverIbcIprpcFundCostResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	err := msg.ValidateBasic()
	if err != nil {
		return &types.MsgCoverIbcIprpcFundCostResponse{}, err
	}

	cost, err := k.Keeper.CoverIbcIprpcFundCost(ctx, msg.Creator, msg.Index)
	if err == nil {
		logger := k.Keeper.Logger(ctx)
		details := map[string]string{
			"creator":      msg.Creator,
			"index":        strconv.FormatUint(msg.Index, 10),
			"cost_covered": cost.String(),
		}
		utils.LogLavaEvent(ctx, logger, types.CoverIbcIprpcFundCostEventName, details, "Covered costs of pending IBC IPRPC fund successfully")
	}

	return &types.MsgCoverIbcIprpcFundCostResponse{}, err
}
