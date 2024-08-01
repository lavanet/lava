package keeper

import (
	"context"
	"fmt"
	"strconv"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/v2/utils"
	"github.com/lavanet/lava/v2/x/subscription/types"
)

func (k msgServer) Buy(goCtx context.Context, msg *types.MsgBuy) (*types.MsgBuyResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	var err error

	if _, err := sdk.AccAddressFromBech32(msg.Consumer); err != nil {
		return nil, utils.LavaFormatError("Invalid consumer address", err,
			utils.LogAttr("consumer", msg.Consumer),
		)
	}

	if _, err := sdk.AccAddressFromBech32(msg.Creator); err != nil {
		return nil, utils.LavaFormatError("Invalid creator address", err,
			utils.LogAttr("creator", msg.Creator),
		)
	}

	if msg.Duration == 0 {
		return nil, utils.LavaFormatError("Invalid duration", fmt.Errorf("duration must be greater than 0"),
			utils.LogAttr("duration", msg.Duration),
		)
	}

	if msg.AdvancePurchase {
		err = k.Keeper.CreateFutureSubscription(ctx, msg.Creator, msg.Consumer, msg.Index, msg.Duration)
		if err == nil {
			logger := k.Keeper.Logger(ctx)
			details := map[string]string{
				"consumer": msg.Consumer,
				"duration": strconv.FormatUint(msg.Duration, 10),
				"plan":     msg.Index,
			}
			utils.LogLavaEvent(ctx, logger, types.AdvancedBuySubscriptionEventName, details, "advanced subscription purchased")
		}
	} else {
		err = k.Keeper.CreateSubscription(ctx, msg.Creator, msg.Consumer, msg.Index, msg.Duration, msg.AutoRenewal)
		if err == nil {
			logger := k.Keeper.Logger(ctx)
			details := map[string]string{
				"consumer": msg.Consumer,
				"duration": strconv.FormatUint(msg.Duration, 10),
				"plan":     msg.Index,
			}
			utils.LogLavaEvent(ctx, logger, types.BuySubscriptionEventName, details, "subscription purchased")
		}
	}

	return &types.MsgBuyResponse{}, err
}
