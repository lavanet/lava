package types

import (
	"fmt"
)

// DefaultIndex is the default capability global index
const DefaultIndex uint64 = 1

// DefaultGenesis returns the default Capability genesis state
func DefaultGenesis() *GenesisState {
	return &GenesisState{
		SpecList: []Spec{},
		// this line is used by starport scaffolding # genesis/types/default
		Params: DefaultParams(),
	}
}

// Validate performs basic genesis state validation returning an error upon any
// failure.
func (gs GenesisState) Validate() error {
	// Check for duplicated ID in spec
	SpecIndexMap := make(map[string]struct{})

	for _, elem := range gs.SpecList {
		index := string(SpecKey(elem.Index))
		if _, ok := SpecIndexMap[index]; ok {
			return fmt.Errorf("duplicated index for Spec")
		}
		SpecIndexMap[index] = struct{}{}
	}

	if gs.SpecCount != uint64(len(gs.SpecList)) {
		return fmt.Errorf("Spec count mismatch spec list")
	}
	// this line is used by starport scaffolding # genesis/types/validate

	return gs.Params.Validate()
}
