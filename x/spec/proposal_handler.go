package spec

import (
	"log"
	"strconv"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	paramkeeper "github.com/cosmos/cosmos-sdk/x/params/keeper"
	paramproposal "github.com/cosmos/cosmos-sdk/x/params/types/proposal"
	"github.com/lavanet/lava/utils"
	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
	"github.com/lavanet/lava/x/spec/keeper"
	"github.com/lavanet/lava/x/spec/types"
)

const (
	SPEC_ADD    = "add_spec"
	SPEC_MODIFY = "spec_modify"
)

// overwriting the params handler so we can add events and callbacks on specific params
// NewParamChangeProposalHandler creates a new governance Handler for a ParamChangeProposal
func NewParamChangeProposalHandler(k paramkeeper.Keeper) govtypes.Handler {
	return func(ctx sdk.Context, content govtypes.Content) error {
		switch c := content.(type) {
		case *paramproposal.ParameterChangeProposal:
			return HandleParameterChangeProposal(ctx, k, c)

		default:
			return sdkerrors.Wrapf(sdkerrors.ErrUnknownRequest, "unrecognized param proposal content type: %T", c)
		}
	}
}

func HandleParameterChangeProposal(ctx sdk.Context, k paramkeeper.Keeper, p *paramproposal.ParameterChangeProposal) error {
	for _, c := range p.Changes {
		ss, ok := k.GetSubspace(c.Subspace)
		if !ok {
			return sdkerrors.Wrap(paramproposal.ErrUnknownSubspace, c.Subspace)
		}

		logger := k.Logger(ctx)
		details := map[string]string{"param": c.Key, "value": c.Value}
		if c.Key == string(epochstoragetypes.KeyLatestParamChange) {
			details["error"] = "tried to modify " + string(epochstoragetypes.KeyLatestParamChange)
			return utils.LavaError(ctx, logger, "param_change", details, "Gov Proposal Param Change Error")
		}
		if err := ss.Update(ctx, []byte(c.Key), []byte(c.Value)); err != nil {
			details["error"] = err.Error()
			return utils.LavaError(ctx, logger, "param_change", details, "Gov Proposal Param Change Error")
		}

		details[epochstoragetypes.ModuleName] = strconv.FormatInt(ctx.BlockHeight(), 10)
		utils.LogLavaEvent(ctx, logger, "param_change", details, "Gov Proposal Accepted Param Changed")
	}

	ss, ok := k.GetSubspace(epochstoragetypes.ModuleName)
	if !ok {
		return sdkerrors.Wrap(paramproposal.ErrUnknownSubspace, epochstoragetypes.ModuleName)
	}
	ss.Set(ctx, epochstoragetypes.KeyLatestParamChange, uint64(ctx.BlockHeight())) // set the LatestParamChange

	return nil
}

// NewSpecProposalsHandler creates a new governance Handler for a Spec
func NewSpecProposalsHandler(k keeper.Keeper) govtypes.Handler {
	return func(ctx sdk.Context, content govtypes.Content) error {
		switch c := content.(type) {
		case *types.SpecAddProposal:
			return handleSpecProposal(ctx, k, c)

		default:
			log.Println("unrecognized spec proposal content")
			return sdkerrors.Wrapf(sdkerrors.ErrUnknownRequest, "unrecognized spec proposal content type: %T", c)
		}
	}
}

func handleSpecProposal(ctx sdk.Context, k keeper.Keeper, p *types.SpecAddProposal) error {
	for _, spec := range p.Specs {
		_, found := k.GetSpec(ctx, spec.Index)

		logger := k.Logger(ctx)

		details, err := spec.ValidateSpec(k.MaxCU(ctx))
		if err != nil {
			return utils.LavaError(ctx, logger, "invalid_spec", details, err.Error())
		}
		k.SetSpec(ctx, spec)
		// TODO: add api types once its implemented to the event

		var name string
		if found {
			name = SPEC_MODIFY
		} else {
			name = SPEC_ADD
		}

		utils.LogLavaEvent(ctx, logger, name, details, "Gov Proposal Accepted Spec")
	}
	return nil
}
