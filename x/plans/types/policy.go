package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

func (policy *Policy) ContainsChainID(chainID string) bool {
	if len(policy.ChainPolicies) == 0 {
		// empty chainPolicies -> support all chains
		return true
	}

	for _, chain := range policy.ChainPolicies {
		if chain.ChainId == chainID {
			return true
		}
	}
	return false
}

func (policy Policy) ValidateBasicPolicy(isPlanPolicy bool) error {
	// plan policy checks
	if isPlanPolicy {
		if policy.EpochCuLimit == 0 || policy.TotalCuLimit == 0 {
			return sdkerrors.Wrapf(ErrInvalidPolicyCuFields, `plan's compute units fields can't be zero 
				(EpochCuLimit = %v, TotalCuLimit = %v)`, policy.EpochCuLimit, policy.TotalCuLimit)
		}

		if policy.SelectedProvidersMode == 3 && len(policy.SelectedProviders) != 0 {
			return sdkerrors.Wrap(ErrPolicyInvalidSelectedProvidersConfig, `cannot configure mode = 3 (selected 
				providers feature is disabled) and non-empty list of selected providers`)
		}

		// non-plan policy checks
	} else if policy.SelectedProvidersMode == 3 {
		return sdkerrors.Wrap(ErrPolicyInvalidSelectedProvidersConfig, `cannot configure mode = 3 (selected 
				providers feature is disabled) for a policy that is not plan policy`)
	}

	// general policy checks
	if policy.EpochCuLimit > policy.TotalCuLimit {
		return sdkerrors.Wrapf(ErrInvalidPolicyCuFields, "EpochCuLimit can't be larger than TotalCuLimit (EpochCuLimit = %v, TotalCuLimit = %v)", policy.EpochCuLimit, policy.TotalCuLimit)
	}

	if policy.MaxProvidersToPair <= 1 {
		return sdkerrors.Wrapf(ErrInvalidPolicyMaxProvidersToPair, "invalid policy's MaxProvidersToPair fields (MaxProvidersToPair = %v)", policy.MaxProvidersToPair)
	}

	if policy.SelectedProvidersMode == 0 && len(policy.SelectedProviders) != 0 {
		return sdkerrors.Wrap(ErrPolicyInvalidSelectedProvidersConfig, `cannot configure mode = 0 (no 
			providers restrictions) and non-empty list of selected providers`)
	}

	if policy.GeolocationProfile == uint64(Geolocation_GLS) && !isPlanPolicy {
		return sdkerrors.Wrap(ErrPolicyGeolocation, `cannot configure geolocation = GLS (0)`)
	}

	seen := map[string]bool{}
	for _, addr := range policy.SelectedProviders {
		_, err := sdk.AccAddressFromBech32(addr)
		if err != nil {
			return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid selected provider address (%s)", err)
		}

		if seen[addr] {
			return sdkerrors.Wrapf(ErrPolicyInvalidSelectedProvidersConfig, "found duplicate provider address %s", addr)
		}
		seen[addr] = true
	}

	return nil
}

func CheckChainIdExistsInPolicies(chainID string, policies []*Policy) bool {
	for _, policy := range policies {
		if policy != nil {
			if policy.ContainsChainID(chainID) {
				return true
			}
		}
	}

	return false
}

func VerifyTotalCuUsage(policies []*Policy, cuUsage uint64) bool {
	for _, policy := range policies {
		if policy != nil {
			if cuUsage >= policy.GetTotalCuLimit() {
				return false
			}
		}
	}

	return true
}
