package spec

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/lavanet/lava/x/spec/keeper"
	"github.com/lavanet/lava/x/spec/types/addproposal"
)

// NewSpecAddProposalHandler creates a new governance Handler for a SpecAddProposal
func NewSpecAddProposalHandler(k keeper.Keeper) govtypes.Handler {
	return func(ctx sdk.Context, content govtypes.Content) error {
		switch c := content.(type) {
		case *addproposal.SpecAddProposal:
			return handleSpecAddProposal(ctx, k, c)

		default:
			return sdkerrors.Wrapf(sdkerrors.ErrUnknownRequest, "unrecognized spec proposal content type: %T", c)
		}
	}
}

func handleSpecAddProposal(ctx sdk.Context, k keeper.Keeper, p *addproposal.SpecAddProposal) error {
	for _, c := range p.Specs {
		existingSpecs := k.GetAllSpec(ctx)
		for _, existingSpec := range existingSpecs {
			if existingSpec.Name == c.Name {
				return sdkerrors.Wrapf(addproposal.ErrEmptyChanges, "name: %s", c.Name)
			}
		}
		k.Logger(ctx).Info(
			fmt.Sprintf("attempt to add new spec; name: %s", c.Name),
		)

		k.AppendSpec(ctx, c)
	}

	return nil
}
