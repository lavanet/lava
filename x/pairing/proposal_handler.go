package pairing

import (
	"log"
	"strings"

	sdkerrors "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	legacyerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"
	"github.com/lavanet/lava/utils"
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
	var providersNotStaked []string
	var providersFailedUnstaking []string
	var providersUnstaked []string
	details := map[string]string{}

	for _, providerUnstakeInfo := range p.ProvidersInfo {
		var chainIDs []string
		if providerUnstakeInfo.ChainId == "*" {
			chainIDs = k.GetAllChainIDs(ctx)
		} else {
			chainIDs = []string{providerUnstakeInfo.ChainId}
		}

		for _, chainID := range chainIDs {
			stakeEntry, err := k.GetStakeEntry(ctx, chainID, providerUnstakeInfo.Provider)
			if err != nil {
				providersNotStaked = append(providersNotStaked, strings.Join([]string{
					providerUnstakeInfo.Provider,
					chainID,
					err.Error(),
				}, ","))
			}

			err = k.UnstakeEntry(ctx, stakeEntry.Chain, stakeEntry.Address, "unstaked via gov proposal")
			if err != nil {
				providersFailedUnstaking = append(providersFailedUnstaking, strings.Join([]string{
					stakeEntry.Address,
					stakeEntry.Chain,
					err.Error(),
				}, ","))
			}

			providersUnstaked = append(providersUnstaked, strings.Join([]string{
				providerUnstakeInfo.Provider,
				chainID,
			}, ","))
		}
	}

	details["providers_unstaked"] = strings.Join(providersUnstaked, ";")
	details["providers_not_staked_from_before"] = strings.Join(providersNotStaked, ";")
	details["providers_failed_unstaking"] = strings.Join(providersFailedUnstaking, ";")

	utils.LogLavaEvent(ctx, k.Logger(ctx), "unstake_gov_proposal", details, "Unstake gov proposal performed")

	return nil
}
