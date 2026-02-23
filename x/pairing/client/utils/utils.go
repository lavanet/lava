package utils

import (
	"os"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/lavanet/lava/v5/x/pairing/types"
)

type (
	PairingUnstakeProposalJSON struct {
		Proposal types.UnstakeProposal `json:"proposal"`
		Deposit  string                `json:"deposit" yaml:"deposit"`
	}

	PairingJailProposalJSON struct {
		Proposal types.JailProposal `json:"proposal"`
		Deposit  string             `json:"deposit" yaml:"deposit"`
	}

	PairingUnjailProposalJSON struct {
		Proposal types.UnjailProposal `json:"proposal"`
		Deposit  string               `json:"deposit" yaml:"deposit"`
	}
)

// ParseUnstakeProposalJSON parses an unstake proposal from a JSON file.
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

// ParseJailProposalJSON parses a jail proposal from a JSON file.
func ParseJailProposalJSON(cdc *codec.LegacyAmino, proposalFile string) (ret PairingJailProposalJSON, err error) {
	var proposal PairingJailProposalJSON
	contents, err := os.ReadFile(proposalFile)
	if err != nil {
		return proposal, err
	}
	if err := cdc.UnmarshalJSON(contents, &proposal); err != nil {
		return proposal, err
	}
	return proposal, nil
}

// ParseUnjailProposalJSON parses an unjail proposal from a JSON file.
func ParseUnjailProposalJSON(cdc *codec.LegacyAmino, proposalFile string) (ret PairingUnjailProposalJSON, err error) {
	var proposal PairingUnjailProposalJSON
	contents, err := os.ReadFile(proposalFile)
	if err != nil {
		return proposal, err
	}
	if err := cdc.UnmarshalJSON(contents, &proposal); err != nil {
		return proposal, err
	}
	return proposal, nil
}
