package utils

import (
	"os"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/lavanet/lava/v2/x/pairing/types"
)

type (
	PairingUnstakeProposalJSON struct {
		Proposal types.UnstakeProposal `json:"proposal"`
		Deposit  string                `json:"deposit" yaml:"deposit"`
	}
)

// Parse unstake add proposal JSON from file
func ParseUnstakeProposalJSON(cdc *codec.LegacyAmino, proposalFile string) (ret PairingUnstakeProposalJSON, err error) {
	var proposal PairingUnstakeProposalJSON

	contents, err := os.ReadFile(proposalFile)
	if err != nil {
		return proposal, err
	}

	if err := cdc.UnmarshalJSON(contents, &proposal); err != nil {
		return proposal, err
	}

	return proposal, nil
}
