package types

import (
	fmt "fmt"
	"strings"

	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
)

const (
	ProposalSpecModify = "SpecModify"
)

var _ govtypes.Content = &SpecModifyProposal{}

func init() {
	govtypes.RegisterProposalType(ProposalSpecModify)
}

func NewSpecModifyProposal(title, description string, specs []Spec) *SpecModifyProposal {
	return &SpecModifyProposal{title, description, specs}
}

// GetTitle returns the title of a proposal.
func (pcp *SpecModifyProposal) GetTitle() string { return pcp.Title }

// GetDescription returns the description of a proposal.
func (pcp *SpecModifyProposal) GetDescription() string { return pcp.Description }

// ProposalRoute returns the routing key of a proposal.
func (pcp *SpecModifyProposal) ProposalRoute() string { return ProposalsRouterKey }

// ProposalType returns the type of a proposal.
func (pcp *SpecModifyProposal) ProposalType() string { return ProposalSpecModify }

// ValidateBasic validates the proposal
func (pcp *SpecModifyProposal) ValidateBasic() error {
	err := govtypes.ValidateAbstract(pcp)
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
func (pcp SpecModifyProposal) String() string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf(`Spec Modify Proposal:
	  Title:       %s
	  Description: %s
	  Changes:
	`, pcp.Title, pcp.Description))

	for _, spec := range pcp.Specs {
		b = stringSpec(spec, b)
	}

	return b.String()
}
