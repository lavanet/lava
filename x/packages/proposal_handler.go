package packages

import (
	"log"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/lavanet/lava/utils"
	"github.com/lavanet/lava/x/packages/keeper"
	"github.com/lavanet/lava/x/packages/types"
)

// NewPackagesProposalsHandler creates a new governance Handler for a Packages
func NewPackagesProposalsHandler(k keeper.Keeper) govtypes.Handler {
	return func(ctx sdk.Context, content govtypes.Content) error {
		switch c := content.(type) {
		case *types.PackagesAddProposal:
			return handlePackagesProposal(ctx, k, c)

		default:
			log.Println("unrecognized packages proposal content")
			return sdkerrors.Wrapf(sdkerrors.ErrUnknownRequest, "unrecognized spec proposal content type: %T", c)
		}
	}
}

func handlePackagesProposal(ctx sdk.Context, k keeper.Keeper, p *types.PackagesAddProposal) error {
	for _, packageElem := range p.Packages {
		logger := k.Logger(ctx)
		err := k.AddNewPackageToStorage(ctx, &packageElem)
		if err != nil {
			return utils.LavaError(ctx, logger, "add_new_package_to_storage", map[string]string{"err": err.Error(), "packageIndex": packageElem.GetIndex()}, "could not add new package")
		}

		utils.LogLavaEvent(ctx, logger, "add_new_package_to_storage", map[string]string{"packageDetails": packageElem.String()}, "Gov Proposal Accepted Package")
	}
	return nil
}
