package statetracker

import (
	"testing"

	"github.com/lavanet/lava/protocol/lavaprotocol"
	"github.com/lavanet/lava/protocol/statetracker/mock"
	"github.com/stretchr/testify/assert"
)

func TestFinalizationConsensusUpdater_RegisterFinalizationConsensus(t *testing.T) {
	// Initialize the FinalizationConsensusUpdater object
	fcu := NewFinalizationConsensusUpdater(nil)

	// Create a new FinalizationConsensus object
	fc := &lavaprotocol.FinalizationConsensus{}

	// Register the FinalizationConsensus object
	fcu.RegisterFinalizationConsensus(fc)

	// Ensure the FinalizationConsensus object is added to the registeredFinalizationConsensuses list
	assert.Equal(t, fcu.registeredFinalizationConsensuses[0], fc)
}

func TestFinalizationConsensusUpdater_UpdaterKey(t *testing.T) {
	// Initialize the FinalizationConsensusUpdater object
	fcu := NewFinalizationConsensusUpdater(nil)

	// Ensure the UpdaterKey() function returns the correct key
	assert.Equal(t, fcu.UpdaterKey(), CallbackKeyForFinalizationConsensusUpdate)
}

func TestFinalizationConsensusUpdater_Update(t *testing.T) {
	// Initialize the FinalizationConsensusUpdater object with mock consumer state query
	fcu := NewFinalizationConsensusUpdater(mock.MockConsumerStateQuery{})

	// Create a new FinalizationConsensus object
	fc := &lavaprotocol.FinalizationConsensus{}

	// Register the FinalizationConsensus object
	fcu.RegisterFinalizationConsensus(fc)

	lastBlock := int64(100)

	// Set nextBlockForUpdate to latest block
	fcu.Update(lastBlock)

	// Ensure the nextBlockForUpdate is equal to lastBlock
	assert.Equal(t, fcu.nextBlockForUpdate, uint64(lastBlock))

	newLowerlastBlock := int64(10)

	// If new latest block lower, don't update
	fcu.Update(newLowerlastBlock)

	// Ensure the nextBlockForUpdate haven't updated
	assert.Equal(t, fcu.nextBlockForUpdate, uint64(lastBlock))

	// On error, it should increase by 1
	fcu.Update(mock.InvalidBlockNumber)

	// Ensure the nextBlockForUpdate haven't updated
	assert.Equal(t, fcu.nextBlockForUpdate, uint64(lastBlock+1))
}
