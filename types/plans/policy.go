package plans

import (
	epochstoragetypes "github.com/lavanet/lava/v5/types/epoch"
)

// ChainRequirement describes the addon and extension requirements for a single chain.
type ChainRequirement struct {
	ChainID    string   `json:"chain_id"`
	Addons     []string `json:"addons"`
	Extensions []string `json:"extensions"`
}

// Policy describes the consumer's subscription policy for a set of chains.
// It is the standalone replacement for the protobuf-generated type of the
// same name in the lavanet.lava.plans module.
type Policy struct {
	// ChainPolicies maps chain IDs to their per-chain requirements.
	ChainPolicies []ChainPolicy `json:"chain_policies"`
}

// ChainPolicy contains the addon and extension policy for one chain.
type ChainPolicy struct {
	ChainID      string             `json:"chain_id"`
	Requirements []ChainRequirement `json:"requirements"`
}

// GetSupportedAddons returns the addons supported by this policy for the
// given specID.  Satisfies the chainlib.PolicyInf interface.
func (p *Policy) GetSupportedAddons(specID string) ([]string, error) {
	if p == nil {
		return nil, nil
	}
	for _, cp := range p.ChainPolicies {
		if cp.ChainID == specID {
			var addons []string
			for _, req := range cp.Requirements {
				addons = append(addons, req.Addons...)
			}
			return addons, nil
		}
	}
	return nil, nil
}

// GetSupportedExtensions returns the extensions supported by this policy for
// the given specID.  Satisfies the chainlib.PolicyInf interface.
func (p *Policy) GetSupportedExtensions(specID string) ([]epochstoragetypes.EndpointService, error) {
	if p == nil {
		return nil, nil
	}
	for _, cp := range p.ChainPolicies {
		if cp.ChainID == specID {
			var services []epochstoragetypes.EndpointService
			for _, req := range cp.Requirements {
				for _, ext := range req.Extensions {
					services = append(services, epochstoragetypes.EndpointService{
						Extension: ext,
					})
				}
			}
			return services, nil
		}
	}
	return nil, nil
}
