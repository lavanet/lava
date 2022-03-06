package spec

import (
	"fmt"
	"log"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/lavanet/lava/x/spec/keeper"
	"github.com/lavanet/lava/x/spec/types/addproposal"
)

// NewSpecAddProposalHandler creates a new governance Handler for a SpecAddProposal
func NewSpecAddProposalHandler(k keeper.Keeper) govtypes.Handler {
	log.Println(k)
	return func(ctx sdk.Context, content govtypes.Content) error {
		switch c := content.(type) {
		case *addproposal.SpecAddProposal:
			log.Println("addproposal.SpecAddProposal", k)
			return handleSpecAddProposal(ctx, k, c)

		default:
			log.Println("unrecognized spec proposal content")
			return sdkerrors.Wrapf(sdkerrors.ErrUnknownRequest, "unrecognized spec proposal content type: %T", c)
		}
	}
}

func handleSpecAddProposal(ctx sdk.Context, k keeper.Keeper, p *addproposal.SpecAddProposal) error {
	log.Println("handleSpecAddProposal", k)

	for _, c := range p.Specs {
		log.Println("c", c)

		//log.Println("k.StoreKey", )
		specCount := k.GetSpecCount(ctx)
		log.Println("specCount", specCount)

		existingSpecs := k.GetAllSpec(ctx)
		log.Println("existingSpecs", existingSpecs)

		for _, existingSpec := range existingSpecs {
			if existingSpec.Name == c.Name {

				log.Println("existingSpec.Name == c.Name", existingSpec.Name, c.Name)
				return sdkerrors.Wrapf(addproposal.ErrEmptyChanges, "name: %s", c.Name)
			}
		}

		log.Println("attempt to add new spec")

		k.Logger(ctx).Info(
			fmt.Sprintf("attempt to add new spec; name: %s", c.Name),
		)

		k.AppendSpec(ctx, c)
	}

	log.Println("handleSpecAddProposal done")

	return nil
}
