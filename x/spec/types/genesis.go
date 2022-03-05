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
	specIdMap := make(map[uint64]bool)
	specCount := gs.GetSpecCount()
	for _, elem := range gs.SpecList {
		if _, ok := specIdMap[elem.Id]; ok {
			return fmt.Errorf("duplicated id for spec")
		}
		if elem.Id >= specCount {
			return fmt.Errorf("spec id should be lower or equal than the last id")
		}
		specIdMap[elem.Id] = true
	}
	// this line is used by starport scaffolding # genesis/types/validate

	return gs.Params.Validate()
}
