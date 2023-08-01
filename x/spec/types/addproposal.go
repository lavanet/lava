package types

import (
	fmt "fmt"
	"strings"

	"github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"

	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const (
	ProposalSpecAdd = "SpecAdd"
)

var _ v1beta1.Content = &SpecAddProposal{}

func init() {
	v1beta1.RegisterProposalType(ProposalSpecAdd)
}

func NewSpecAddProposal(title, description string, specs []Spec) *SpecAddProposal {
	return &SpecAddProposal{title, description, specs}
}

// GetTitle returns the title of a proposal.
func (pcp *SpecAddProposal) GetTitle() string { return pcp.Title }

// GetDescription returns the description of a proposal.
func (pcp *SpecAddProposal) GetDescription() string { return pcp.Description }

// ProposalRoute returns the routing key of a proposal.
func (pcp *SpecAddProposal) ProposalRoute() string { return ProposalsRouterKey }

// ProposalType returns the type of a proposal.
func (pcp *SpecAddProposal) ProposalType() string { return ProposalSpecAdd }

// ValidateBasic validates the proposal
func (pcp *SpecAddProposal) ValidateBasic() error {
	err := v1beta1.ValidateAbstract(pcp)
	if err != nil {
		return err
	}

	if len(pcp.Specs) == 0 {
		return sdkerrors.Wrap(ErrEmptySpecs, "proposal specs cannot be empty")
	}
	for _, spec := range pcp.Specs {
		err := checkSpecProposal(spec)
		if err != nil {
			return err
		}
	}

	return nil
}

// String implements the Stringer interface.
func (pcp SpecAddProposal) String() string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf(`Spec Add Proposal:
	  Title:       %s
	  Description: %s
	  Changes:
	`, pcp.Title, pcp.Description))

	for _, spec := range pcp.Specs {
		b = stringSpec(spec, b)
	}

	return b.String()
}
