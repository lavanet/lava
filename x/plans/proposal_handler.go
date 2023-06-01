package plans

import (
	"log"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/lavanet/lava/utils"
	"github.com/lavanet/lava/x/plans/keeper"
	"github.com/lavanet/lava/x/plans/types"
)

// NewPlanProposalsHandler creates a new governance Handler for a Plan
func NewPlansProposalsHandler(k keeper.Keeper) govtypes.Handler {
	return func(ctx sdk.Context, content govtypes.Content) error {
		switch c := content.(type) {
		case *types.PlansAddProposal:
			return handlePlansAddProposal(ctx, k, c)
		case *types.PlansDelProposal:
			return handlePlansDelProposal(ctx, k, c)
		default:
			log.Println("unrecognized plans proposal content")
			return sdkerrors.Wrapf(sdkerrors.ErrUnknownRequest, "unrecognized plans proposal content type: %T", c)
		}
	}
}

func handlePlansAddProposal(ctx sdk.Context, k keeper.Keeper, p *types.PlansAddProposal) error {
	// add the plans to the plan storage
	for _, planElem := range p.Plans {
		logger := k.Logger(ctx)
		err := k.AddPlan(ctx, planElem)
		if err != nil {
			return utils.LavaFormatError("could not add new plan", err,
				utils.Attribute{Key: "planIndex", Value: planElem.Index},
			)
		}

		details := map[string]string{"planDetails": planElem.String()}
		utils.LogLavaEvent(ctx, logger, "add_new_plan_to_storage", details, "Gov Proposal Accepted Plans")
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
		utils.LogLavaEvent(ctx, logger, "del_plan_from_storage", details, "Gov Proposal Accepted Plans")
	}
	return nil
}
