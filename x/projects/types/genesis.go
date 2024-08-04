package types

import (
	fixationtypes "github.com/lavanet/lava/v2/x/fixationstore/types"
)

// this line is used by starport scaffolding # genesis/types/import

// DefaultIndex is the default capability global index
const DefaultIndex uint64 = 1

// DefaultGenesis returns the default Capability genesis state
func DefaultGenesis() *GenesisState {
	return &GenesisState{
		// this line is used by starport scaffolding # genesis/types/default
		Params:      DefaultParams(),
		ProjectsFS:  *fixationtypes.DefaultGenesis(),
		DeveloperFS: *fixationtypes.DefaultGenesis(),
	}
}

// Validate performs basic genesis state validation returning an error upon any
// failure.
func (gs GenesisState) Validate() error {
	// this line is used by starport scaffolding # genesis/types/validate

	return gs.Params.Validate()
}
