package keeper

import (
	"context"
	"fmt"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/utils"
	"github.com/lavanet/lava/x/subscription/types"
)

func (k msgServer) AutoRenewal(goCtx context.Context, msg *types.MsgAutoRenewal) (*types.MsgAutoRenewalResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// Find consumer's subscription
	sub, found := k.GetSubscription(ctx, msg.Consumer)
	if !found {
		return nil, utils.LavaFormatWarning("could not change auto-renewal of subscription", fmt.Errorf("subscription not found"),
			utils.Attribute{Key: "consumer", Value: msg.Consumer},
		)
	}

	// Verify creator either sub.Creator or sub.Consumer
	if msg.Creator != sub.Consumer && msg.Creator != sub.Creator {
		return nil, utils.LavaFormatWarning("could not change auto-renewal of subscription", fmt.Errorf("creator is not authorized to change auto-renewal for this subscription"),
			utils.Attribute{Key: "creator", Value: msg.Creator},
			utils.Attribute{Key: "consumer", Value: msg.Consumer},
		)
	}

	// If msg.Enable == false, verify not already disabled
	if !msg.Enable && sub.AutoRenewalNextPlan == types.AUTO_RENEWAL_PLAN_NONE {
		return nil, utils.LavaFormatWarning("could not change auto-renewal of subscription", fmt.Errorf("auto-renewal is already disabled"),
			utils.Attribute{Key: "creator", Value: msg.Creator},
			utils.Attribute{Key: "consumer", Value: msg.Consumer},
		)
	}

	// If msg.Enable == true, verify plan index
	if msg.Enable {
		if strings.TrimSpace(msg.Index) == "" {
			msg.Index = sub.PlanIndex
		}

		if _, found := k.plansKeeper.FindPlan(ctx, msg.Index, uint64(ctx.BlockHeight())); !found {
			return nil, utils.LavaFormatWarning("could not change auto-renewal of subscription", fmt.Errorf("could not find plan (%s)", msg.Index),
				utils.Attribute{Key: "creator", Value: msg.Creator},
				utils.Attribute{Key: "consumer", Value: msg.Consumer},
				utils.Attribute{Key: "index", Value: msg.Index},
			)
		}
	}

	newPlanIndex := msg.Index
	if !msg.Enable {
		newPlanIndex = types.AUTO_RENEWAL_PLAN_NONE
	}

	// For the event log
	prevCreator := sub.Creator
	prevAutoRenewalNextPlan := sub.AutoRenewalNextPlan

	sub.Creator = msg.Creator
	sub.AutoRenewalNextPlan = newPlanIndex
	err := k.subsFS.AppendEntry(ctx, msg.Consumer, sub.Block, &sub)
	if err != nil {
		return nil, utils.LavaFormatError("could not change auto-renewal of subscription", err,
			utils.Attribute{Key: "sub_consumer", Value: msg.Creator},
			utils.Attribute{Key: "original_auto_renewal", Value: sub.AutoRenewalNextPlan},
			utils.Attribute{Key: "new_auto_renewal", Value: msg.Enable},
		)
	}

	details := map[string]string{
		"prevCreator":       prevCreator,
		"creator":           msg.Creator,
		"consumer":          msg.Consumer,
		"prevAutoRenewPlan": prevAutoRenewalNextPlan,
		"newAutoRenewPlan":  sub.AutoRenewalNextPlan,
	}
	utils.LogLavaEvent(ctx, k.Logger(ctx), types.SubscriptionAutoRenewChangeEventName, details, "subscription auto-renew changed")

	return &types.MsgAutoRenewalResponse{}, nil
}
