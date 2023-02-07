package utils

import (
	"os"
	"strings"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/x/packages/types"
)

type (
	PackagesAddProposalJSON struct {
		Proposal types.PackagesAddProposal `json:"proposal"`
		Deposit  string                    `json:"deposit" yaml:"deposit"`
	}
)

// Parse packages add proposal JSON form file
func ParsePackagesAddProposalJSON(cdc *codec.LegacyAmino, proposalFile string) (ret PackagesAddProposalJSON, err error) {
	for _, fileName := range strings.Split(proposalFile, ",") {
		proposal := PackagesAddProposalJSON{}

		contents, err := os.ReadFile(fileName)
		if err != nil {
			return proposal, err
		}

		if err := cdc.UnmarshalJSON(contents, &proposal); err != nil {
			return proposal, err
		}
		if len(ret.Proposal.Packages) > 0 {
			ret.Proposal.Packages = append(ret.Proposal.Packages, proposal.Proposal.Packages...)
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
