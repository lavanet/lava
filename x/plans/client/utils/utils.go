package utils

import (
	"strings"

	sdkerrors "cosmossdk.io/errors"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	commontypes "github.com/lavanet/lava/v2/utils/common/types"
	"github.com/lavanet/lava/v2/utils/decoder"
	"github.com/lavanet/lava/v2/x/plans/types"
	"github.com/mitchellh/mapstructure"
)

type (
	PlansAddProposalJSON struct {
		Proposal types.PlansAddProposal `json:"proposal"`
		Deposit  string                 `json:"deposit"`
	}
)

type (
	PlansDelProposalJSON struct {
		Proposal types.PlansDelProposal `json:"proposal"`
		Deposit  string                 `json:"deposit"`
	}
)

// Parse plans add proposal JSON form file
func ParsePlansAddProposalJSON(proposalFile string) (ret PlansAddProposalJSON, err error) {
	decoderHooks := []mapstructure.DecodeHookFunc{
		types.PriceDecodeHookFunc,
		types.PolicyEnumDecodeHookFunc,
	}

	for _, fileName := range strings.Split(proposalFile, ",") {
		var (
			plansAddProposal PlansAddProposalJSON
			unused           []string
			unset            []string
		)

		err = decoder.DecodeFile(fileName, "proposal", &plansAddProposal.Proposal, decoderHooks, &unset, &unused)
		if err != nil {
			return PlansAddProposalJSON{}, err
		}

		err = decoder.DecodeFile(fileName, "deposit", &plansAddProposal.Deposit, nil, nil, nil)
		if err != nil {
			return PlansAddProposalJSON{}, err
		}

		err = plansAddProposal.Proposal.ValidateBasic()
		if err != nil {
			return PlansAddProposalJSON{}, err
		}

		if len(unset) > 0 {
			err = plansAddProposal.Proposal.HandleUnsetPlanProposalFields(unset)
			if err != nil {
				return PlansAddProposalJSON{}, err
			}
		}

		if len(plansAddProposal.Proposal.Plans) > 0 {
			ret.Proposal.Plans = append(ret.Proposal.Plans, plansAddProposal.Proposal.Plans...)
			ret.Proposal.Description = ret.Proposal.Description + " " + plansAddProposal.Proposal.Description
			ret.Proposal.Title = ret.Proposal.Title + " " + plansAddProposal.Proposal.Title
			ret.Proposal.Modify = plansAddProposal.Proposal.Modify

			proposalDeposit, err := sdk.ParseCoinNormalized(plansAddProposal.Deposit)
			if err != nil {
				return PlansAddProposalJSON{}, err
			}

			if proposalDeposit.Denom != commontypes.TokenDenom {
				return PlansAddProposalJSON{}, sdkerrors.Wrapf(types.ErrInvalidDenom, "Coin denomanator is not ulava")
			}

			if ret.Deposit != "" {
				retDeposit, err := sdk.ParseCoinNormalized(ret.Deposit)
				if err != nil {
					return PlansAddProposalJSON{}, err
				}
				if retDeposit.Denom != commontypes.TokenDenom {
					return PlansAddProposalJSON{}, sdkerrors.Wrapf(types.ErrInvalidDenom, "Coin denomanator is not ulava")
				}
				ret.Deposit = retDeposit.Add(proposalDeposit).String()
			} else {
				ret.Deposit = proposalDeposit.String()
			}
		}
	}

	return ret, nil
}

// Parse plans delete proposal JSON form file
func ParsePlansDelProposalJSON(cdc *codec.LegacyAmino, proposalFile string) (ret PlansDelProposalJSON, err error) {
	for _, fileName := range strings.Split(proposalFile, ",") {
		var proposal PlansDelProposalJSON

		err = decoder.DecodeFile(fileName, "proposal", &proposal.Proposal, nil, nil, nil)
		if err != nil {
			return PlansDelProposalJSON{}, err
		}

		err = decoder.DecodeFile(fileName, "deposit", &proposal.Deposit, nil, nil, nil)
		if err != nil {
			return PlansDelProposalJSON{}, err
		}

		err = proposal.Proposal.ValidateBasic()
		if err != nil {
			return PlansDelProposalJSON{}, err
		}

		if len(ret.Proposal.Plans) > 0 {
			ret.Proposal.Plans = append(ret.Proposal.Plans, proposal.Proposal.Plans...)
			ret.Proposal.Description = proposal.Proposal.Description + " " + ret.Proposal.Description
			ret.Proposal.Title = proposal.Proposal.Title + " " + ret.Proposal.Title
			retDeposit, err := sdk.ParseCoinNormalized(ret.Deposit)
			if err != nil {
				return proposal, err
			}

			if retDeposit.Denom != commontypes.TokenDenom {
				return proposal, sdkerrors.Wrapf(types.ErrInvalidDenom, "Coin denomanator is not ulava")
			}

			proposalDeposit, err := sdk.ParseCoinNormalized(proposal.Deposit)
			if err != nil {
				return proposal, err
			}

			if proposalDeposit.Denom != commontypes.TokenDenom {
				return proposal, sdkerrors.Wrapf(types.ErrInvalidDenom, "Coin denomanator is not ulava")
			}

			ret.Deposit = retDeposit.Add(proposalDeposit).String()
		} else {
			ret = proposal
		}
	}
	return ret, nil
}
