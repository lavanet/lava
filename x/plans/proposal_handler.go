package plans

import (
	"log"

	sdkerrors "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	legacyerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"
	"github.com/lavanet/lava/v2/utils"
	"github.com/lavanet/lava/v2/x/plans/keeper"
	"github.com/lavanet/lava/v2/x/plans/types"
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
		err := k.AddPlan(ctx, planElem, p.Modify)
		if err != nil {
			return utils.LavaFormatError("could not add new plan", err,
				utils.Attribute{Key: "planIndex", Value: planElem.Index},
			)
		}

		details := map[string]string{"planDetails": planElem.String()}
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
