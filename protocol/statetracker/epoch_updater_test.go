package statetracker

import (
	"context"
	"testing"

	"github.com/lavanet/lava/protocol/statetracker/mock"
	"github.com/stretchr/testify/assert"
)

func TestEpochUpdater_UpdaterKey(t *testing.T) {
	// Initialize the EpochUpdater object
	eu := NewEpochUpdater(nil)

	// Ensure the UpdaterKey() function returns the correct key
	assert.Equal(t, eu.UpdaterKey(), CallbackKeyForEpochUpdate)
}

func TestEpochUpdater_RegisterEpochUpdatable(t *testing.T) {
	// Initialize the EpochUpdater object
	eu := NewEpochUpdater(nil)

	// Register new epoch updater
	eu.RegisterEpochUpdatable(context.Background(), mock.MockEpochUpdatable{})

	// Ensure the new updater was added
	assert.Equal(t, len(eu.epochUpdatables), 1)
}

func TestEpochUpdater_Update(t *testing.T) {
	// Initialize the EpochUpdater object
	eu := NewEpochUpdater(mock.MockProviderStateQuery{})

	// Register new epoch updater
	eu.RegisterEpochUpdatable(context.Background(), mock.MockEpochUpdatable{})

	// Update
	eu.Update(10)

	// Update same epoch
	eu.Update(10)

	// Initialize the EpochUpdater object with error
	eu = NewEpochUpdater(mock.MockErrorProviderStateQuery{})

	// Update
	eu.Update(10)
}
