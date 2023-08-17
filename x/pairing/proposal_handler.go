package pairing

import (
	"log"

	sdkerrors "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	legacyerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"
	"github.com/lavanet/lava/x/pairing/keeper"
	"github.com/lavanet/lava/x/pairing/types"
)

// NewPlanProposalsHandler creates a new governance Handler for a Plan
func NewPairingProposalsHandler(k keeper.Keeper) v1beta1.Handler {
	return func(ctx sdk.Context, content v1beta1.Content) error {
		switch c := content.(type) {
		case *types.UnstakeProposal:
			return handleUnstakeProposal(ctx, k, c)
		default:
			log.Println("unrecognized plans proposal content")
			return sdkerrors.Wrapf(legacyerrors.ErrUnknownRequest, "unrecognized plans proposal content type: %T", c)
		}
	}
}

func handleUnstakeProposal(ctx sdk.Context, k keeper.Keeper, p *types.UnstakeProposal) error {
	for _, providerInfo := range p.ProvidersInfo {
		stakeEntry, err := k.GetStakeEntry(ctx, providerInfo.ChainId, providerInfo.Provider)
		if err != nil {
			return err
		}

		err = k.UnstakeEntry(ctx, stakeEntry.Chain, stakeEntry.Address, "unstaked via gov proposal")
		if err != nil {
			return err
		}
	}

	return nil
}
