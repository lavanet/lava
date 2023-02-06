package types

import (
	fmt "fmt"
	"strings"

	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
)

const (
	ProposalPackagesAdd = "PackagesAdd"
)

var _ govtypes.Content = &PackagesAddProposal{}

func init() {
	govtypes.RegisterProposalType(ProposalPackagesAdd)
}

func NewPackagesAddProposal(title, description string, packages []Package) *PackagesAddProposal {
	return &PackagesAddProposal{title, description, packages}
}

// GetTitle returns the title of a proposal.
func (pcp *PackagesAddProposal) GetTitle() string { return pcp.Title }

// GetDescription returns the description of a proposal.
func (pcp *PackagesAddProposal) GetDescription() string { return pcp.Description }

// ProposalRoute returns the routing key of a proposal.
func (pcp *PackagesAddProposal) ProposalRoute() string { return ProposalsRouterKey }

// ProposalType returns the type of a proposal.
func (pcp *PackagesAddProposal) ProposalType() string { return ProposalPackagesAdd }

// ValidateBasic validates the proposal
func (pcp *PackagesAddProposal) ValidateBasic() error {
	err := govtypes.ValidateAbstract(pcp)
	if err != nil {
		return err
	}

	if len(pcp.Packages) == 0 {
		return sdkerrors.Wrap(ErrEmptyPackages, "proposal packages cannot be empty")
	}
	for _, packageElem := range pcp.Packages {
		err := checkPackagesProposal(packageElem)
		if err != nil {
			return err
		}
	}

	return nil
}

// String implements the Stringer interface.
func (pcp PackagesAddProposal) String() string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf(`Packages Add Proposal:
	  Title:       %s
	  Description: %s
	  Changes:
	`, pcp.Title, pcp.Description))

	for _, packageElem := range pcp.Packages {
		b = stringPackage(&packageElem, b)
	}

	return b.String()
}
