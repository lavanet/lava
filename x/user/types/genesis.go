package types

import (
	"fmt"
)

// DefaultIndex is the default capability global index
const DefaultIndex uint64 = 1

// DefaultGenesis returns the default Capability genesis state
func DefaultGenesis() *GenesisState {
	return &GenesisState{
		UserStakeList:              []UserStake{},
		SpecStakeStorageList:       []SpecStakeStorage{},
		BlockDeadlineForCallback:   BlockDeadlineForCallback{Deadline: BlockNum{Num: 0}},
		UnstakingUsersAllSpecsList: []UnstakingUsersAllSpecs{},
		// this line is used by starport scaffolding # genesis/types/default
		Params: DefaultParams(),
	}
}

// Validate performs basic genesis state validation returning an error upon any
// failure.
func (gs GenesisState) Validate() error {
	// Check for duplicated index in userStake
	userStakeIndexMap := make(map[string]struct{})

	for _, elem := range gs.UserStakeList {
		index := string(UserStakeKey(elem.Index))
		if _, ok := userStakeIndexMap[index]; ok {
			return fmt.Errorf("duplicated index for userStake")
		}
		userStakeIndexMap[index] = struct{}{}
	}
	// Check for duplicated index in specStakeStorage
	specStakeStorageIndexMap := make(map[string]struct{})

	for _, elem := range gs.SpecStakeStorageList {
		index := string(SpecStakeStorageKey(elem.Index))
		if _, ok := specStakeStorageIndexMap[index]; ok {
			return fmt.Errorf("duplicated index for specStakeStorage")
		}
		specStakeStorageIndexMap[index] = struct{}{}
	}
	// Check for duplicated ID in unstakingUsersAllSpecs
	unstakingUsersAllSpecsIdMap := make(map[uint64]bool)
	unstakingUsersAllSpecsCount := gs.GetUnstakingUsersAllSpecsCount()
	for _, elem := range gs.UnstakingUsersAllSpecsList {
		if _, ok := unstakingUsersAllSpecsIdMap[elem.Id]; ok {
			return fmt.Errorf("duplicated id for unstakingUsersAllSpecs")
		}
		if elem.Id >= unstakingUsersAllSpecsCount {
			return fmt.Errorf("unstakingUsersAllSpecs id should be lower or equal than the last id")
		}
		unstakingUsersAllSpecsIdMap[elem.Id] = true
	}
	// this line is used by starport scaffolding # genesis/types/validate

	return gs.Params.Validate()
}
