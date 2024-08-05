package types

import (
	fmt "fmt"
	"strings"

	"github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"

	sdkerrors "cosmossdk.io/errors"
	commontypes "github.com/lavanet/lava/v2/utils/common/types"
)

const (
	ProposalPlansDel = "PlansDel"
)

func init() {
	v1beta1.RegisterProposalType(ProposalPlansDel)
}

func NewPlansDelProposal(title, description string, indices []string) *PlansDelProposal {
	return &PlansDelProposal{title, description, indices}
}

// GetTitle returns the title of a proposal.
func (pcp *PlansDelProposal) GetTitle() string { return pcp.Title }

// GetDescription returns the description of a proposal.
func (pcp *PlansDelProposal) GetDescription() string { return pcp.Description }

// ProposalRoute returns the routing key of a proposal.
func (pcp *PlansDelProposal) ProposalRoute() string { return ProposalsRouterKey }

// ProposalType returns the type of a proposal.
func (pcp *PlansDelProposal) ProposalType() string { return ProposalPlansDel }

// ValidateBasic validates the proposal
func (pcp *PlansDelProposal) ValidateBasic() error {
	err := v1beta1.ValidateAbstract(pcp)
	if err != nil {
		return err
	}

	if len(pcp.Plans) == 0 {
		return sdkerrors.Wrap(ErrEmptyPlans, "proposal plans delete cannot be empty")
	}
	for _, index := range pcp.Plans {
		if !commontypes.ValidateString(index, commontypes.NAME_RESTRICTIONS, nil) {
			return fmt.Errorf("invalid plan name: %s", index)
		}
	}

	return nil
}

// String implements the Stringer interface.
func (pcp PlansDelProposal) String() string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf(`Plan Del Proposal:
	  Title:       %s
	  Description: %s
	  Deleted:     %s
	`, pcp.Title, pcp.Description, strings.Join(pcp.Plans, ", ")))

	return b.String()
}
