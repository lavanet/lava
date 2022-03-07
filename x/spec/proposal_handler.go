package spec

import (
	"fmt"
	"log"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/lavanet/lava/x/spec/keeper"
	"github.com/lavanet/lava/x/spec/types"
)

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
	for _, c := range p.Specs {

		//
		// Verify 'name' is unique
		existingSpecs := k.GetAllSpec(ctx)
		for _, existingSpec := range existingSpecs {
			if existingSpec.Name == c.Name {
				return sdkerrors.Wrapf(types.ErrDuplicateSpecName, "found duplicate spec name; name: %s", c.Name)
			}
		}
		k.Logger(ctx).Info(
			fmt.Sprintf("attempt to add new spec; name: %s", c.Name),
		)
		k.AppendSpec(ctx, c)
	}

	return nil
}

func handleSpecModifyProposal(ctx sdk.Context, k keeper.Keeper, p *types.SpecModifyProposal) error {
	for _, c := range p.Specs {

		//
		// Find by name
		existingSpecs := k.GetAllSpec(ctx)
		foundSpecI := -1
		for i, existingSpec := range existingSpecs {
			if existingSpec.Name == c.Name {
				foundSpecI = i
				break
			}
		}
		if foundSpecI < 0 {
			return sdkerrors.Wrapf(types.ErrSpecNotFound, "spec to modify not found; name: %s", c.Name)
		}
		c.Id = uint64(foundSpecI)

		//
		// Set new spec
		k.Logger(ctx).Info(
			fmt.Sprintf("attempt to set new spec; name: %s", c.Name),
		)
		k.SetSpec(ctx, c)
	}

	return nil
}
