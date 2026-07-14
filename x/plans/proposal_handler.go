package plans

import (
	"log"
	"strings"

	sdkerrors "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	legacyerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"
	"github.com/lavanet/lava/v5/utils"
	"github.com/lavanet/lava/v5/x/plans/keeper"
	"github.com/lavanet/lava/v5/x/plans/types"
)

// NewPlanProposalsHandler creates a new governance Handler for a Plan
func NewPlansProposalsHandler(k keeper.Keeper) v1beta1.Handler {
	return func(ctx sdk.Context, content v1beta1.Content) error {
		switch c := content.(type) {
		case *types.PlansAddProposal:
			return handlePlansAddProposal(ctx, k, c)
		case *types.PlansDelProposal:
			return handlePlansDelProposal(ctx, k, c)
		default:
			log.Println("unrecognized plans proposal content")
			return sdkerrors.Wrapf(legacyerrors.ErrUnknownRequest, "unrecognized plans proposal content type: %T", c)
		}
	}
}

func handlePlansAddProposal(ctx sdk.Context, k keeper.Keeper, p *types.PlansAddProposal) error {
	// add the plans to the plan storage
	for _, planElem := range p.Plans {
		logger := k.Logger(ctx)

		// grab the plan version this proposal replaces (if any), to reflect
		// in the event what the update actually changed
		replacedBlock := uint64(ctx.BlockHeight())
		if p.Modify {
			replacedBlock = planElem.Block
		}
		replacedPlan, isUpdate := k.FindPlan(ctx, planElem.Index, replacedBlock)

		err := k.AddPlan(ctx, planElem, p.Modify)
		if err != nil {
			return utils.LavaFormatError("could not add new plan", err,
				utils.Attribute{Key: "planIndex", Value: planElem.Index},
			)
		}

		details := map[string]string{"planDetails": planElem.String()}
		if isUpdate {
			changes := types.DiffPlans(replacedPlan, planElem)
			if len(changes) == 0 {
				details["changes"] = "none (identical to the replaced version)"
			} else {
				details["changes"] = strings.Join(changes, "; ")
			}
		}
		utils.LogLavaEvent(ctx, logger, types.PlanAddEventName, details, "Gov Proposal Accepted Plans")
	}
	return nil
}

func handlePlansDelProposal(ctx sdk.Context, k keeper.Keeper, p *types.PlansDelProposal) error {
	// add the plans to the plan storage
	for _, index := range p.Plans {
		logger := k.Logger(ctx)
		err := k.DelPlan(ctx, index)
		if err != nil {
			return utils.LavaFormatError("could not del existing plan", err,
				utils.Attribute{Key: "planIndex", Value: index},
			)
		}

		details := map[string]string{"index": index}
		utils.LogLavaEvent(ctx, logger, types.PlanDelEventName, details, "Gov Proposal Accepted Plans")
	}
	return nil
}
