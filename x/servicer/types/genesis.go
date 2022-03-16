package types

import (
	"fmt"
)

// DefaultIndex is the default capability global index
const DefaultIndex uint64 = 1

// DefaultGenesis returns the default Capability genesis state
func DefaultGenesis() *GenesisState {
	return &GenesisState{
		StakeMapList:                   []StakeMap{},
		SpecStakeStorageList:           []SpecStakeStorage{},
		BlockDeadlineForCallback:       nil,
		UnstakingServicersAllSpecsList: []UnstakingServicersAllSpecs{},
		// this line is used by starport scaffolding # genesis/types/default
		Params: DefaultParams(),
	}
}

// Validate performs basic genesis state validation returning an error upon any
// failure.
func (gs GenesisState) Validate() error {
	// Check for duplicated index in stakeMap
	stakeMapIndexMap := make(map[string]struct{})

	for _, elem := range gs.StakeMapList {
		index := string(StakeMapKey(elem.Index))
		if _, ok := stakeMapIndexMap[index]; ok {
			return fmt.Errorf("duplicated index for stakeMap")
		}
		stakeMapIndexMap[index] = struct{}{}
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
	// Check for duplicated ID in unstakingServicersAllSpecs
	unstakingServicersAllSpecsIdMap := make(map[uint64]bool)
	unstakingServicersAllSpecsCount := gs.GetUnstakingServicersAllSpecsCount()
	for _, elem := range gs.UnstakingServicersAllSpecsList {
		if _, ok := unstakingServicersAllSpecsIdMap[elem.Id]; ok {
			return fmt.Errorf("duplicated id for unstakingServicersAllSpecs")
		}
		if elem.Id >= unstakingServicersAllSpecsCount {
			return fmt.Errorf("unstakingServicersAllSpecs id should be lower or equal than the last id")
		}
		unstakingServicersAllSpecsIdMap[elem.Id] = true
	}
	// this line is used by starport scaffolding # genesis/types/validate

	return gs.Params.Validate()
}
