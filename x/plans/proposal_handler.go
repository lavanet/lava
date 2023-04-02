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
			return handlePlansProposal(ctx, k, c)

		default:
			log.Println("unrecognized plans proposal content")
			return sdkerrors.Wrapf(sdkerrors.ErrUnknownRequest, "unrecognized plans proposal content type: %T", c)
		}
	}
}

func handlePlansProposal(ctx sdk.Context, k keeper.Keeper, p *types.PlansAddProposal) error {
	// add the plans to the plan storage
	for _, planElem := range p.Plans {
		logger := k.Logger(ctx)
		if len(planElem.PlanPolicy.GetChainPolicies()) > 0 {
			err := k.ValidateChainPolicies(ctx, planElem.GetPlanPolicy())
			if err != nil {
				details := map[string]string{
					"planIndex": planElem.GetIndex(),
					"err":       err.Error(),
				}
				return utils.LavaError(ctx, logger, "validate_chain_policies", details, "invalid chain policies in plan policy")
			}
		}

		err := k.AddPlan(ctx, planElem)
		if err != nil {
			details := map[string]string{
				"planIndex": planElem.GetIndex(),
				"err":       err.Error(),
			}
			return utils.LavaError(ctx, logger, "add_new_plan_to_storage", details, "could not add new plan")
		}

		details := map[string]string{"planDetails": planElem.String()}
		utils.LogLavaEvent(ctx, logger, "add_new_plan_to_storage", details, "Gov Proposal Accepted Package")
	}
	return nil
}
