package v1

// DefaultGenesisState returns a default genesis state.
func DefaultGenesisState() *GenesisState {
	return &GenesisState{
		Params:                     DefaultParams(),
		Downtimes:                  nil,
		DowntimesGarbageCollection: nil,
	}
}

func (m *GenesisState) Validate() error { return nil }
