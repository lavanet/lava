package types

import fmt "fmt"

// this line is used by starport scaffolding # genesis/types/import

// DefaultIndex is the default capability global index
const DefaultIndex uint64 = 1

// DefaultGenesis returns the default Capability genesis state
func DefaultGenesis() *GenesisState {
	return &GenesisState{
		// this line is used by starport scaffolding # genesis/types/default
		Params: DefaultParams(),
	}
}

// Validate performs basic genesis state validation returning an error upon any
// failure.
func (gs GenesisState) Validate() error {
	// this line is used by starport scaffolding # genesis/types/validate
	timeEntries := gs.RefillRewardsTS.GetTimeEntries()
	if len(timeEntries) > 1 {
		return fmt.Errorf(`there should be up to one timer in RefillRewardsTS
			at all times. amount of timers found: %v`, len(timeEntries))
	}

	return gs.Params.Validate()
}
