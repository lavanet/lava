package addproposal

import (
	"log"

	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	types "github.com/lavanet/lava/x/spec/types"
)

const (
	// ProposalTypeChange defines the type for a ParameterChangeProposal
	ProposalSpecAdd = "SpecAdd"
)

// Assert ParameterChangeProposal implements govtypes.Content at compile-time
var _ govtypes.Content = &SpecAddProposal{}

func init() {
	govtypes.RegisterProposalType(ProposalSpecAdd)
}

func NewSpecAddProposal(title, description string, specs []types.Spec) *SpecAddProposal {
	return &SpecAddProposal{title, description, specs}
}

// GetTitle returns the title of a parameter change proposal.
func (pcp *SpecAddProposal) GetTitle() string { return pcp.Title }

// GetDescription returns the description of a parameter change proposal.
func (pcp *SpecAddProposal) GetDescription() string { return pcp.Description }

// ProposalRoute returns the routing key of a parameter change proposal.
func (pcp *SpecAddProposal) ProposalRoute() string { return RouterKey }

// ProposalType returns the type of a parameter change proposal.
func (pcp *SpecAddProposal) ProposalType() string { return ProposalSpecAdd }

// ValidateBasic validates the parameter change proposal
func (pcp *SpecAddProposal) ValidateBasic() error {
	err := govtypes.ValidateAbstract(pcp)
	if err != nil {
		log.Println("ValidateBasic: err", err)
		return err
	}
	log.Println("ValidateBasic: done")

	//return ValidateChanges(pcp.Changes)
	return nil
}

// String implements the Stringer interface.
func (pcp SpecAddProposal) String() string {
	// 	var b strings.Builder

	// 	b.WriteString(fmt.Sprintf(`Spec Add Proposal:
	//   Title:       %s
	//   Description: %s
	//   Changes:
	// `, pcp.Title, pcp.Description))

	// 	for _, pc := range pcp.Specs {
	// 		b.WriteString(fmt.Sprintf(`    Spec Add:
	// 	      Name: %s
	// 	`, pc.Name))
	// 	}

	// 	return b.String()
	return ""
}

// func NewParamChange(subspace, key, value string) ParamChange {
// 	return ParamChange{subspace, key, value}
// }

// String implements the Stringer interface.
// func (pc ParamChange) String() string {
// 	out, _ := yaml.Marshal(pc)
// 	return string(out)
// }

// ValidateChanges performs basic validation checks over a set of ParamChange. It
// returns an error if any ParamChange is invalid.
// func ValidateChanges(changes []ParamChange) error {
// 	if len(changes) == 0 {
// 		return ErrEmptyChanges
// 	}

// 	for _, pc := range changes {
// 		if len(pc.Subspace) == 0 {
// 			return ErrEmptySubspace
// 		}
// 		if len(pc.Key) == 0 {
// 			return ErrEmptyKey
// 		}
// 		if len(pc.Value) == 0 {
// 			return ErrEmptyValue
// 		}
// 	}

// 	return nil
// }
