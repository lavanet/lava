package statetracker

import (
	"context"
	"testing"

	"github.com/lavanet/lava/protocol/lavasession"
	"github.com/lavanet/lava/protocol/statetracker/mock"
	"github.com/stretchr/testify/assert"
)

func TestVoteUpdater_UpdaterKey(t *testing.T) {
	// Initialize the VoteUpdater object
	vu := NewVoteUpdater(nil)

	// Ensure the UpdaterKey() function returns the correct key
	assert.Equal(t, vu.UpdaterKey(), CallbackKeyForVoteUpdate)
}

func TestVoteUpdater_RegisterEpochUpdatable(t *testing.T) {
	// Initialize the VoteUpdater object
	vu := NewVoteUpdater(nil)

	// Register new payment updater
	vu.RegisterVoteUpdatable(context.Background(), mock.MockVoteUpdatable{}, lavasession.RPCEndpoint{NetworkAddress: "127.0.0.1:8545",
		ChainID:      mock.InvalidChainID,
		ApiInterface: "rest",
		Geolocation:  1})

	// Ensure the new updater was added
	assert.Equal(t, len(vu.voteUpdatables), 1)
}

func TestVoteUpdater_Update(t *testing.T) {
	// Initialize the NewVoteUpdater object
	pu := NewVoteUpdater(mock.MockProviderStateQuery{})

	// Register new payment updater
	pu.RegisterVoteUpdatable(context.Background(), mock.MockVoteUpdatable{}, lavasession.RPCEndpoint{NetworkAddress: "127.0.0.1:8545",
		ChainID:      "LAV1",
		ApiInterface: "rest",
		Geolocation:  1})

	// Update
	pu.Update(10)

	// Initialize the EpochUpdater object with error
	pu = NewVoteUpdater(mock.MockErrorProviderStateQuery{})

	// Register new payment updater
	pu.RegisterVoteUpdatable(context.Background(), mock.MockVoteUpdatable{}, lavasession.RPCEndpoint{NetworkAddress: "127.0.0.1:8545",
		ChainID:      mock.InvalidChainID,
		ApiInterface: "rest",
		Geolocation:  1})

	// Update
	pu.Update(10)
}
