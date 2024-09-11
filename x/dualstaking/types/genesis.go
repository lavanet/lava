package types

import (
	fmt "fmt"

	"cosmossdk.io/collections"
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
	delegatorRewardIndexMap := make(map[collections.Pair[string, string]]struct{})

	for _, elem := range gs.DelegatorRewardList {
		index := DelegationKey(elem.Provider, elem.Delegator)
		if _, ok := delegatorRewardIndexMap[index]; ok {
			return fmt.Errorf("duplicated index for delegatorReward")
		}
		delegatorRewardIndexMap[index] = struct{}{}
	}

	delegationIndexMap := make(map[collections.Pair[string, string]]struct{})
	for _, elem := range gs.Delegations {
		index := DelegationKey(elem.Provider, elem.Delegator)
		if _, ok := delegationIndexMap[index]; ok {
			return fmt.Errorf("duplicated index for delegations")
		}
		delegationIndexMap[index] = struct{}{}
	}
	// this line is used by starport scaffolding # genesis/types/validate

	return gs.Params.Validate()
}
