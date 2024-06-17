package spec

import (
	"fmt"

	legacyerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"

	sdkerrors "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	paramkeeper "github.com/cosmos/cosmos-sdk/x/params/keeper"
	paramproposal "github.com/cosmos/cosmos-sdk/x/params/types/proposal"
	"github.com/lavanet/lava/utils"
	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
	"github.com/lavanet/lava/x/spec/types"
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
