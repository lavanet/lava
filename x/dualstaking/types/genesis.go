package types

import (
	fmt "fmt"
)

// DefaultIndex is the default global index
const DefaultIndex uint64 = 1

// DefaultGenesis returns the default genesis state
func DefaultGenesis() *GenesisState {
	return &GenesisState{
		// this line is used by starport scaffolding # genesis/types/default
		Params:              DefaultParams(),
		DelegatorRewardList: []DelegatorReward{},
		Delegations:         []Delegation{},
	}
}

// Validate performs basic genesis state validation returning an error upon any
// failure.
func (gs GenesisState) Validate() error {
	// Check for duplicated index in delegatorReward
	delegatorRewardIndexMap := make(map[string]struct{})

	for _, elem := range gs.DelegatorRewardList {
		key := DelegationKey(elem.Provider, elem.Delegator)
		index := key.K1() + key.K2()
		if _, ok := delegatorRewardIndexMap[index]; ok {
			return fmt.Errorf("duplicated index for delegatorReward")
		}
		delegatorRewardIndexMap[index] = struct{}{}
	}

	delegationIndexMap := make(map[string]struct{})
	for _, elem := range gs.Delegations {
		key := DelegationKey(elem.Provider, elem.Delegator)
		index := key.K1() + key.K2()
		if _, ok := delegationIndexMap[index]; ok {
			return fmt.Errorf("duplicated index for delegations")
		}
		delegationIndexMap[index] = struct{}{}
	}
	// this line is used by starport scaffolding # genesis/types/validate

	return gs.Params.Validate()
}
