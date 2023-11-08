package keeper

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/utils"
	"github.com/lavanet/lava/x/subscription/types"
)

func (k msgServer) AutoRenewal(goCtx context.Context, msg *types.MsgAutoRenewal) (*types.MsgAutoRenewalResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	sub, found := k.GetSubscription(ctx, msg.Creator)
	if !found {
		return nil, utils.LavaFormatWarning("could not change auto-renewal of subscription", fmt.Errorf("subscription not found"),
			utils.Attribute{Key: "sub_consumer", Value: msg.Creator},
		)
	}

	sub.AutoRenewal = msg.Enable
	err := k.subsFS.AppendEntry(ctx, msg.Creator, uint64(ctx.BlockHeight()), &sub)
	if err != nil {
		return nil, utils.LavaFormatError("could not change auto-renewal of subscription", err,
			utils.Attribute{Key: "sub_consumer", Value: msg.Creator},
			utils.Attribute{Key: "original_auto_renewal", Value: sub.AutoRenewal},
			utils.Attribute{Key: "new_auto_renewal", Value: msg.Enable},
		)
	}

	return &types.MsgAutoRenewalResponse{}, nil
}
