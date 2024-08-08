package types

import (
	"fmt"
)

// DefaultIndex is the default capability global index
const DefaultIndex uint64 = 1

// DefaultGenesis returns the default Capability genesis state
func DefaultGenesis() *GenesisState {
	return &GenesisState{
		StakeStorageList:  []StakeStorage{},
		EpochDetails:      &EpochDetails{StartBlock: 0, EarliestStart: 0, DeletedEpochs: []uint64{}},
		FixatedParamsList: []FixatedParams{},
		// this line is used by starport scaffolding # genesis/types/default
		Params: DefaultParams(),
	}
}

// Validate performs basic genesis state validation returning an error upon any
// failure.
func (gs GenesisState) Validate() error {
	// Check for duplicated index in stakeStorage
	stakeStorageIndexMap := make(map[string]struct{})

	for _, elem := range gs.StakeStorageList {
		if _, ok := stakeStorageIndexMap[elem.Index]; ok {
			return fmt.Errorf("duplicated index for stakeStorage")
		}
		stakeStorageIndexMap[elem.Index] = struct{}{}
	}
	// Check for duplicated index in fixatedParams
	fixatedParamsIndexMap := make(map[string]struct{})

	for _, elem := range gs.FixatedParamsList {
		index := string(FixatedParamsKey(elem.Index))
		if _, ok := fixatedParamsIndexMap[index]; ok {
			return fmt.Errorf("duplicated index for fixatedParams")
		}
		fixatedParamsIndexMap[index] = struct{}{}
	}
	// this line is used by starport scaffolding # genesis/types/validate

	return gs.Params.Validate()
}
