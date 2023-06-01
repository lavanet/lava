package types

import (
	fmt "fmt"
	"strings"

	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
)

const (
	ProposalPlansAdd = "PlansAdd"
)

func init() {
	govtypes.RegisterProposalType(ProposalPlansAdd)
}

func NewPlansAddProposal(title, description string, plans []Plan) *PlansAddProposal {
	return &PlansAddProposal{title, description, plans}
}

// GetTitle returns the title of a proposal.
func (pcp *PlansAddProposal) GetTitle() string { return pcp.Title }

// GetDescription returns the description of a proposal.
func (pcp *PlansAddProposal) GetDescription() string { return pcp.Description }

// ProposalRoute returns the routing key of a proposal.
func (pcp *PlansAddProposal) ProposalRoute() string { return ProposalsRouterKey }

// ProposalType returns the type of a proposal.
func (pcp *PlansAddProposal) ProposalType() string { return ProposalPlansAdd }

// ValidateBasic validates the proposal
func (pcp *PlansAddProposal) ValidateBasic() error {
	err := govtypes.ValidateAbstract(pcp)
	if err != nil {
		return err
	}

	if len(pcp.Plans) == 0 {
		return sdkerrors.Wrap(ErrEmptyPlans, "proposal plans add cannot be empty")
	}
	for _, planElem := range pcp.Plans {
		err := planElem.ValidatePlan()
		if err != nil {
			return err
		}
	}

	return nil
}

// String implements the Stringer interface.
func (pcp PlansAddProposal) String() string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf(`Plan Add Proposal:
	  Title:       %s
	  Description: %s
	  Changes:
	`, pcp.Title, pcp.Description))

	for _, planElem := range pcp.Plans {
		b.WriteString(planElem.String())
	}

	return b.String()
}
