package spec

import (
	"fmt"
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
	spectypes "github.com/lavanet/lava/x/spec/types"
)

const minCU = 1

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
		// set param change callback
		if c.Subspace == epochstoragetypes.ModuleName {
			details[epochstoragetypes.ModuleName] = strconv.FormatInt(ctx.BlockHeight(), 10)
			ss.Set(ctx, epochstoragetypes.KeyLatestParamChange, uint64(ctx.BlockHeight())) //set the LatestParamChange
		}

		utils.LogLavaEvent(ctx, logger, "param_change", details, "Gov Proposal Accepted Param Changed")
	}

	return nil
}

// NewSpecProposalsHandler creates a new governance Handler for a Spec
func NewSpecProposalsHandler(k keeper.Keeper) govtypes.Handler {
	log.Println(k)
	return func(ctx sdk.Context, content govtypes.Content) error {
		switch c := content.(type) {
		case *types.SpecAddProposal:
			return handleSpecAddProposal(ctx, k, c)

		case *types.SpecModifyProposal:
			return handleSpecModifyProposal(ctx, k, c)

		default:
			log.Println("unrecognized spec proposal content")
			return sdkerrors.Wrapf(sdkerrors.ErrUnknownRequest, "unrecognized spec proposal content type: %T", c)
		}
	}
}

func handleSpecAddProposal(ctx sdk.Context, k keeper.Keeper, p *types.SpecAddProposal) error {
	logger := k.Logger(ctx)
	for _, spec := range p.Specs {
		details := map[string]string{"spec": spec.Name, "status": strconv.FormatBool(spec.Enabled), "chainID": spec.Index}
		//
		// Verify 'name' is unique
		_, found := k.GetSpec(ctx, spec.Index)

		if found {
			return utils.LavaError(ctx, logger, "spec_add_dup", details, "found duplicate spec name")
		}

		functionTags := map[string]bool{}

		for _, api := range spec.Apis {
			if api.ComputeUnits < minCU || api.ComputeUnits > k.MaxCU(ctx) {
				details["api"] = api.Name
				return utils.LavaError(ctx, logger, "spec_add_cu_oor", details, "Compute units out or range")
			}

			if api.Parsing.FunctionTag != "" {
				// Validate tag name
				result := false
				for _, tag := range spectypes.SupportedTags {
					if tag == api.Parsing.FunctionTag {
						result = true
						functionTags[api.Parsing.FunctionTag] = true
					}
				}

				if !result {
					details["api"] = api.Name
					return utils.LavaError(ctx, logger, "spec_add_ft_inv", details, "Unsupported function tag")
				}
			}
		}

		if spec.ComparesHashes {
			for _, tag := range []string{spectypes.GET_BLOCKNUM, spectypes.GET_BLOCK_BY_NUM} {
				if found := functionTags[tag]; !found {
					return utils.LavaError(ctx, logger, "spec_add_ch_mis", details, fmt.Sprintf("missing tagged functions for hash comparison: %s", tag))
				}
			}
		}

		k.SetSpec(ctx, spec)
		//TODO: add api types once its implemented to the event

		utils.LogLavaEvent(ctx, logger, "spec_add", details, "Gov Proposal Accepted Spec Added")
	}

	return nil
}

func handleSpecModifyProposal(ctx sdk.Context, k keeper.Keeper, p *types.SpecModifyProposal) error {
	logger := k.Logger(ctx)
	for _, spec := range p.Specs {

		details := map[string]string{"spec": spec.Name, "status": strconv.FormatBool(spec.Enabled), "chainID": spec.Index}
		//
		// Find by name
		_, found := k.GetSpec(ctx, spec.Index)

		if !found {
			return utils.LavaError(ctx, logger, "spec_modify_missing", details, "spec to modify not found")
		}

		for _, api := range spec.Apis {
			if api.ComputeUnits < minCU || api.ComputeUnits > k.MaxCU(ctx) {
				details["api"] = api.Name
				return utils.LavaError(ctx, logger, "spec_add_cu_oor", details, "Compute units out or range")
			}
		}

		k.SetSpec(ctx, spec)
		utils.LogLavaEvent(ctx, logger, "spec_modify", details, "Gov Proposal Accepted Spec Modified")
	}

	return nil
}
