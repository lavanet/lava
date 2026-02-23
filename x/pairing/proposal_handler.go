package pairing

import (
	"log"
	"math"
	"strings"

	sdkerrors "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	legacyerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"
	"github.com/lavanet/lava/v5/utils"
	"github.com/lavanet/lava/v5/x/pairing/keeper"
	"github.com/lavanet/lava/v5/x/pairing/types"
)

// NewPlanProposalsHandler creates a new governance Handler for a Plan
func NewPairingProposalsHandler(k keeper.Keeper) v1beta1.Handler {
	return func(ctx sdk.Context, content v1beta1.Content) error {
		switch c := content.(type) {
		case *types.UnstakeProposal:
			return handleUnstakeProposal(ctx, k, c)
		case *types.JailProposal:
			return handleJailProposal(ctx, k, c)
		case *types.UnjailProposal:
			return handleUnjailProposal(ctx, k, c)
		default:
			log.Println("unrecognized pairing proposal content")
			return sdkerrors.Wrapf(legacyerrors.ErrUnknownRequest, "unrecognized pairing proposal content type: %T", c)
		}
	}
}

func handleUnstakeProposal(ctx sdk.Context, k keeper.Keeper, p *types.UnstakeProposal) error {
	var providersNotStaked []string
	var providersFailedUnstaking []string
	var delegatorsFailedSlashing []string
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
				continue
			}

			err = k.UnstakeEntryForce(ctx, stakeEntry.Chain, stakeEntry.Address, "unstaked via gov proposal")
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

	for _, delegatorSlashing := range p.DelegatorsSlashing {
		err := k.SlashDelegator(ctx, delegatorSlashing)
		if err != nil {
			delegatorsFailedSlashing = append(delegatorsFailedSlashing, strings.Join([]string{delegatorSlashing.Delegator, err.Error()}, ","))
		}
	}
	details["providers_unstaked"] = strings.Join(providersUnstaked, ";")
	details["providers_not_staked_from_before"] = strings.Join(providersNotStaked, ";")
	details["providers_failed_unstaking"] = strings.Join(providersFailedUnstaking, ";")
	details["delegators_failed_slaghing"] = strings.Join(delegatorsFailedSlashing, ";")

	utils.LogLavaEvent(ctx, k.Logger(ctx), types.UnstakeProposalEventName, details, "Unstake gov proposal performed")

	return nil
}

func handleJailProposal(ctx sdk.Context, k keeper.Keeper, p *types.JailProposal) error {
	const permanentJailEndTime = int64(math.MaxInt64)

	details := map[string]string{}
	var jailed, notFound []string

	for _, info := range p.ProvidersInfo {
		chainIDs := []string{info.ChainId}
		if info.ChainId == "*" {
			chainIDs = k.GetAllChainIDs(ctx)
		}

		jailEndTime := info.JailEndTime
		if jailEndTime == 0 {
			jailEndTime = permanentJailEndTime
		}

		for _, chainID := range chainIDs {
			if err := k.JailProviderForProposal(ctx, info.Provider, chainID, jailEndTime); err != nil {
				notFound = append(notFound, strings.Join([]string{info.Provider, chainID, err.Error()}, ","))
				continue
			}
			jailed = append(jailed, strings.Join([]string{info.Provider, chainID}, ","))
		}
	}

	details["jailed"]    = strings.Join(jailed, ";")
	details["not_found"] = strings.Join(notFound, ";")
	details["reason"]    = func() string {
		reasons := make([]string, 0, len(p.ProvidersInfo))
		for _, info := range p.ProvidersInfo {
			reasons = append(reasons, info.Reason)
		}
		return strings.Join(reasons, ";")
	}()

	utils.LogLavaEvent(ctx, k.Logger(ctx), types.JailProposalEventName, details, "Jail provider gov proposal performed")

	return nil
}

func handleUnjailProposal(ctx sdk.Context, k keeper.Keeper, p *types.UnjailProposal) error {
	details := map[string]string{}
	var unjailed, notFound []string

	for _, info := range p.ProvidersInfo {
		chainIDs := []string{info.ChainId}
		if info.ChainId == "*" {
			chainIDs = k.GetAllChainIDs(ctx)
		}

		for _, chainID := range chainIDs {
			if err := k.UnjailProviderForProposal(ctx, info.Provider, chainID); err != nil {
				notFound = append(notFound, strings.Join([]string{info.Provider, chainID, err.Error()}, ","))
				continue
			}
			unjailed = append(unjailed, strings.Join([]string{info.Provider, chainID}, ","))
		}
	}

	details["unjailed"]  = strings.Join(unjailed, ";")
	details["not_found"] = strings.Join(notFound, ";")

	utils.LogLavaEvent(ctx, k.Logger(ctx), types.UnjailProposalEventName, details, "Unjail provider gov proposal performed")

	return nil
}
