package v1

import "fmt"

// DefaultGenesisState returns a default genesis state.
func DefaultGenesisState() *GenesisState {
	return &GenesisState{
		Params:                     DefaultParams(),
		Downtimes:                  nil,
		DowntimesGarbageCollection: nil,
		LastBlockTime:              nil,
	}
}

func (m *GenesisState) Validate() error {
	if err := m.Params.Validate(); err != nil {
		return fmt.Errorf("invalid params: %w", err)
	}
	for i, d := range m.Downtimes {
		if err := d.Validate(); err != nil {
			return fmt.Errorf("invalid downtime %d: %w", i, err)
		}
	}

	for i, gc := range m.DowntimesGarbageCollection {
		if err := gc.Validate(); err != nil {
			return fmt.Errorf("invalid downtime garbage collection %d: %w", i, err)
		}
	}

	return nil
}

func (m *Downtime) Validate() error {
	if m.Duration <= 0 {
		return fmt.Errorf("invalid downtime duration: %s", m.Duration)
	}
	if m.Block == 0 {
		return fmt.Errorf("invalid downtime block: %d", m.Block)
	}
	return nil
}

func (m *DowntimeGarbageCollection) Validate() error {
	if m.Block == 0 {
		return fmt.Errorf("invalid downtime garbage collection block: %d", m.Block)
	}
	return nil
}
