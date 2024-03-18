package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"
)

const (
	ProposalUnstake = "PairingUnstake"
)

func init() {
	v1beta1.RegisterProposalType(ProposalUnstake)
}

func NewUnstakeProposal(title, description string, providers []ProviderUnstakeInfo, delegatorsSlashing []DelegatorSlashing) *UnstakeProposal {
	return &UnstakeProposal{title, description, providers, delegatorsSlashing}
}

// GetTitle returns the title of a proposal.
func (up *UnstakeProposal) GetTitle() string { return up.Title }

// GetDescription returns the description of a proposal.
func (up *UnstakeProposal) GetDescription() string { return up.Description }

// ProposalRoute returns the routing key of a proposal.
func (up *UnstakeProposal) ProposalRoute() string { return ProposalsRouterKey }

// ProposalType returns the type of a proposal.
func (up *UnstakeProposal) ProposalType() string { return ProposalUnstake }

// ValidateBasic validates the proposal
func (up *UnstakeProposal) ValidateBasic() error {
	err := v1beta1.ValidateAbstract(up)
	if err != nil {
		return err
	}

	for _, providerInfo := range up.ProvidersInfo {
		_, err = sdk.AccAddressFromBech32(providerInfo.Provider)
		if err != nil {
			return err
		}
	}

	return nil
}
