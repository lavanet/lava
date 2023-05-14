package keeper

import (
	"context"
	"strconv"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/utils"
	"github.com/lavanet/lava/x/subscription/types"
)

func (k msgServer) Buy(goCtx context.Context, msg *types.MsgBuy) (*types.MsgBuyResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	err := k.Keeper.CreateSubscription(ctx, msg.Creator, msg.Consumer, msg.Index, msg.Duration)
	if err == nil {
		logger := k.Keeper.Logger(ctx)
		details := map[string]string{"consumer": msg.Consumer, "duration": strconv.FormatUint(msg.Duration, 10), "plan": msg.Index}
		utils.LogLavaEvent(ctx, logger, types.BuySubscriptionEventName, details, "consumer bought subscription")
	}
	return &types.MsgBuyResponse{}, err
}
