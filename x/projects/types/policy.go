package types

import (
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

func (policy Policy) ValidateBasicPolicy() error {
	if policy.EpochCuLimit > policy.TotalCuLimit {
		return sdkerrors.Wrapf(ErrInvalidPolicyCuFields, "invalid policy's CU fields (EpochCuLimit = %v, TotalCuLimit = %v)", policy.EpochCuLimit, policy.TotalCuLimit)
	}

	if policy.MaxProvidersToPair <= 1 {
		return sdkerrors.Wrapf(ErrInvalidPolicyMaxProvidersToPair, "invalid policy's MaxProvidersToPair fields (MaxProvidersToPair = %v)", policy.MaxProvidersToPair)
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

func VerifyCuUsage(policies []*Policy, cuUsage uint64) bool {
	for _, policy := range policies {
		if policy != nil {
			if cuUsage >= policy.GetTotalCuLimit() {
				return true
			}
		}
	}

	return false
}
