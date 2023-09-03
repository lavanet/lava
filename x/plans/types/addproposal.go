package types

import (
	fmt "fmt"
	"strconv"
	"strings"

	"github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"

	sdkerrors "cosmossdk.io/errors"
)

const (
	ProposalPlansAdd = "PlansAdd"
)

func init() {
	v1beta1.RegisterProposalType(ProposalPlansAdd)
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
	err := v1beta1.ValidateAbstract(pcp)
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

// the unset fields in the proposal are in plans[0].plan_policy... format, so they need special handling
func (pcp *PlansAddProposal) HandleUnsetPlanProposalFields(unset []string) error {
	for i := 0; i < len(pcp.Plans); i++ {
		var planUnset []string
		currentPlanPrefix := "plans[" + strconv.FormatInt(int64(i), 10) + "]."

		for _, unsetStr := range unset {
			if strings.HasPrefix(unsetStr, currentPlanPrefix) {
				planUnsetField := strings.TrimPrefix(unsetStr, currentPlanPrefix)
				planUnset = append(planUnset, planUnsetField)
			}
		}

		if !CheckPlanMandatoryFields(unset) {
			return fmt.Errorf("plan has no specified price or index. plan's unset fields: %v", unset)
		}

		planPolicyPrefix := "plan_policy."
		var planPolicyUnset []string
		for _, planUnsetField := range planUnset {
			if strings.HasPrefix(planUnsetField, planPolicyPrefix) {
				planPolicyUnsetField := strings.TrimPrefix(planUnsetField, planPolicyPrefix)
				planPolicyUnset = append(planPolicyUnset, planPolicyUnsetField)
			}
		}

		policy := pcp.Plans[i].PlanPolicy
		err := policy.HandleUnsetPolicyFields(planPolicyUnset)
		if err != nil {
			return err
		}
		pcp.Plans[i].PlanPolicy = policy
	}

	return nil
}
