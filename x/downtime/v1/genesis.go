package v1

// DefaultGenesisState returns a default genesis state.
func DefaultGenesisState() *GenesisState {
	return &GenesisState{}
}

func (m *GenesisState) Validate() error { return nil }
