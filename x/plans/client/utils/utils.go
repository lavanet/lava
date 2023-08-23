package utils

import (
	"fmt"
	"math/big"
	"os"
	"reflect"
	"strings"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/utils/decoder"
	"github.com/lavanet/lava/x/plans/types"
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
func ParsePlansAddProposalJSON(cdc *codec.LegacyAmino, proposalFile string) (PlansAddProposalJSON, error) {
	var ret PlansAddProposalJSON
	decoderHooks := []mapstructure.DecodeHookFunc{
		priceDecodeHookFunc,
		types.PolicyEnumDecodeHookFunc,
	}

	files := strings.Split(proposalFile, ",")
	for _, fileName := range files {
		var (
			plansAddProposal PlansAddProposalJSON
			unused           []string
			unset            []string
			err              error
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

		if len(plansAddProposal.Proposal.Plans) > 0 {
			ret.Proposal.Plans = append(ret.Proposal.Plans, plansAddProposal.Proposal.Plans...)
			ret.Proposal.Description = ret.Proposal.Description + " " + plansAddProposal.Proposal.Description
			ret.Proposal.Title = ret.Proposal.Title + " " + plansAddProposal.Proposal.Title

			proposalDeposit, err := sdk.ParseCoinNormalized(plansAddProposal.Deposit)
			if err != nil {
				return PlansAddProposalJSON{}, err
			}
			if ret.Deposit != "" {
				retDeposit, err := sdk.ParseCoinNormalized(ret.Deposit)
				if err != nil {
					return PlansAddProposalJSON{}, err
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

		contents, err := os.ReadFile(fileName)
		if err != nil {
			return proposal, err
		}

		if err := cdc.UnmarshalJSON(contents, &proposal); err != nil {
			return proposal, err
		}
		if len(ret.Proposal.Plans) > 0 {
			ret.Proposal.Plans = append(ret.Proposal.Plans, proposal.Proposal.Plans...)
			ret.Proposal.Description = proposal.Proposal.Description + " " + ret.Proposal.Description
			ret.Proposal.Title = proposal.Proposal.Title + " " + ret.Proposal.Title
			retDeposit, err := sdk.ParseCoinNormalized(ret.Deposit)
			if err != nil {
				return proposal, err
			}
			proposalDeposit, err := sdk.ParseCoinNormalized(proposal.Deposit)
			if err != nil {
				return proposal, err
			}
			ret.Deposit = retDeposit.Add(proposalDeposit).String()
		} else {
			ret = proposal
		}
	}
	return ret, nil
}

// PriceDecodeHookFunc helps the decoder to correctly unmarshal the price field's amount (type sdk.Int)
func priceDecodeHookFunc(f reflect.Type, t reflect.Type, data interface{}) (interface{}, error) {
	if t == reflect.TypeOf(sdk.NewInt(0)) {
		amountStr, ok := data.(string)
		if !ok {
			return nil, fmt.Errorf("unexpected data type for amount field")
		}

		// Convert the string amount to an sdk.Int
		amount, ok := new(big.Int).SetString(amountStr, 10)
		if !ok {
			return nil, fmt.Errorf("failed to convert amount to sdk.Int")
		}
		return sdk.NewIntFromBigInt(amount), nil
	}

	return data, nil
}
