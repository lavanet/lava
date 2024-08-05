package spec

import (
	"fmt"
	"log"
	"strings"

	legacyerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"

	sdkerrors "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	paramkeeper "github.com/cosmos/cosmos-sdk/x/params/keeper"
	paramproposal "github.com/cosmos/cosmos-sdk/x/params/types/proposal"
	"github.com/lavanet/lava/v2/utils"
	epochstoragetypes "github.com/lavanet/lava/v2/x/epochstorage/types"
	"github.com/lavanet/lava/v2/x/spec/keeper"
	"github.com/lavanet/lava/v2/x/spec/types"
)

// overwriting the params handler so we can add events and callbacks on specific params
// NewParamChangeProposalHandler creates a new governance Handler for a ParamChangeProposal
func NewParamChangeProposalHandler(k paramkeeper.Keeper) v1beta1.Handler {
	return func(ctx sdk.Context, content v1beta1.Content) error {
		switch c := content.(type) {
		case *paramproposal.ParameterChangeProposal:
			return HandleParameterChangeProposal(ctx, k, c)

		default:
			return sdkerrors.Wrapf(legacyerrors.ErrUnknownRequest, "unrecognized param proposal content type: %T", c)
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
		details := []utils.Attribute{
			{Key: "param", Value: c.Key},
			{Key: "value", Value: c.Value},
		}
		if c.Key == string(epochstoragetypes.KeyLatestParamChange) {
			return utils.LavaFormatWarning("Gov Proposal Param Change Error", fmt.Errorf("tried to modify "+string(epochstoragetypes.KeyLatestParamChange)),
				details...,
			)
		}
		if err := ss.Update(ctx, []byte(c.Key), []byte(c.Value)); err != nil {
			return utils.LavaFormatWarning("Gov Proposal Param Change Error", fmt.Errorf("tried to modify %s: %w", c.Key, err),
				details...,
			)
		}

		details = append(details, utils.Attribute{Key: epochstoragetypes.ModuleName, Value: ctx.BlockHeight()})

		detailsMap := map[string]string{}
		for _, atr := range details {
			detailsMap[atr.Key] = fmt.Sprint(atr.Value)
		}
		utils.LogLavaEvent(ctx, logger, types.ParamChangeEventName, detailsMap, "Gov Proposal Accepted Param Changed")
	}

	ss, ok := k.GetSubspace(epochstoragetypes.ModuleName)
	if !ok {
		return sdkerrors.Wrap(paramproposal.ErrUnknownSubspace, epochstoragetypes.ModuleName)
	}
	ss.Set(ctx, epochstoragetypes.KeyLatestParamChange, uint64(ctx.BlockHeight())) // set the LatestParamChange

	return nil
}

// NewSpecProposalsHandler creates a new governance Handler for a Spec
func NewSpecProposalsHandler(k keeper.Keeper) v1beta1.Handler {
	return func(ctx sdk.Context, content v1beta1.Content) error {
		switch c := content.(type) {
		case *types.SpecAddProposal:
			return handleSpecProposal(ctx, k, c)

		default:
			log.Println("unrecognized spec proposal content")
			return sdkerrors.Wrapf(legacyerrors.ErrUnknownRequest, "unrecognized spec proposal content type: %T", c)
		}
	}
}

func handleSpecProposal(ctx sdk.Context, k keeper.Keeper, p *types.SpecAddProposal) error {
	logger := k.Logger(ctx)

	type event struct {
		name    string
		event   string
		details map[string]string
	}

	var events []event

	for _, spec := range p.Specs {
		_, found := k.GetSpec(ctx, spec.Index)

		spec.BlockLastUpdated = uint64(ctx.BlockHeight())
		k.SetSpec(ctx, spec)

		details, err := k.ValidateSpec(ctx, spec)
		if err != nil {
			attrs := utils.StringMapToAttributes(details)
			return utils.LavaFormatWarning("invalid spec", err, attrs...)
		}

		name := types.SpecAddEventName

		if found {
			name = types.SpecModifyEventName
		}

		// collect the events first, and only log them after everything succeeded
		events = append(events, event{
			name:    name,
			event:   "Gov Proposal Accepted Spec",
			details: details,
		})

		// TODO: add api types once its implemented to the event
	}

	// re-validate all the specs, in case the modified spec is imported by
	// other specs and the new version creates a conflict; also update the
	// BlockLastUpdated of all specs that inherit from the modified spec.
	for _, spec := range k.GetAllSpec(ctx) {
		inherits, err := k.RefreshSpec(ctx, spec, p.Specs)
		if err != nil {
			return utils.LavaFormatWarning("invalidated spec", err)
		}
		if len(inherits) > 0 {
			details := map[string]string{
				"name":   spec.Index,
				"import": strings.Join(inherits, ","),
			}
			name := types.SpecRefreshEventName
			events = append(events, event{
				name:    name,
				event:   "Gov Proposal Refreshsed Spec",
				details: details,
			})
		}
	}

	for _, e := range events {
		utils.LogLavaEvent(ctx, logger, e.name, e.details, e.event)
	}

	return nil
}
