package types

import (
	"fmt"

	"github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"
)

func init() {
	v1beta1.RegisterProposalType(ProposalTypeJail)
	v1beta1.RegisterProposalType(ProposalTypeUnjail)
}

// ---- JailProposal: v1beta1.Content interface ----

func (p *JailProposal) ProposalRoute() string  { return ProposalsRouterKey }
func (p *JailProposal) ProposalType() string   { return ProposalTypeJail }
func (p *JailProposal) GetTitle() string       { return p.Title }
func (p *JailProposal) GetDescription() string { return p.Description }
func (p *JailProposal) ValidateBasic() error {
	if len(p.ProvidersInfo) == 0 {
		return fmt.Errorf("jail proposal must include at least one provider")
	}
	for _, info := range p.ProvidersInfo {
		if info.Provider == "" {
			return fmt.Errorf("jail proposal provider address cannot be empty")
		}
		if info.ChainId == "" {
			return fmt.Errorf("jail proposal chain_id cannot be empty")
		}
	}
	return nil
}

// ---- UnjailProposal: v1beta1.Content interface ----

func (p *UnjailProposal) ProposalRoute() string  { return ProposalsRouterKey }
func (p *UnjailProposal) ProposalType() string   { return ProposalTypeUnjail }
func (p *UnjailProposal) GetTitle() string       { return p.Title }
func (p *UnjailProposal) GetDescription() string { return p.Description }
func (p *UnjailProposal) ValidateBasic() error {
	if len(p.ProvidersInfo) == 0 {
		return fmt.Errorf("unjail proposal must include at least one provider")
	}
	for _, info := range p.ProvidersInfo {
		if info.Provider == "" {
			return fmt.Errorf("unjail proposal provider address cannot be empty")
		}
		if info.ChainId == "" {
			return fmt.Errorf("unjail proposal chain_id cannot be empty")
		}
	}
	return nil
}

// compile-time interface check
var _ v1beta1.Content = &JailProposal{}
var _ v1beta1.Content = &UnjailProposal{}
