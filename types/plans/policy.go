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

	// GeolocationProfile is the bitmask of required geolocations.
	GeolocationProfile int32 `json:"geolocation_profile"`

	// TotalCuLimit is the overall compute-unit cap for this policy.
	TotalCuLimit uint64 `json:"total_cu_limit"`

	// EpochCuLimit is the per-epoch compute-unit cap.
	EpochCuLimit uint64 `json:"epoch_cu_limit"`

	// MaxProvidersToPair is the maximum number of providers to pair.
	MaxProvidersToPair uint64 `json:"max_providers_to_pair"`
}

// ChainPolicy contains the addon and extension policy for one chain.
type ChainPolicy struct {
	ChainID    string              `json:"chain_id"`
	Apis       []ChainRequirement  `json:"apis"`
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
