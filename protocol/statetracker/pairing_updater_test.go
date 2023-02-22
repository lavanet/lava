package statetracker

import (
	"context"
	"testing"

	"github.com/lavanet/lava/protocol/lavasession"
	"github.com/lavanet/lava/protocol/statetracker/mock"
	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
	"github.com/stretchr/testify/assert"
)

func TestPairingUpdater_UpdaterKey(t *testing.T) {
	// Initialize the FinalizationConsensusUpdater object
	fcu := NewPairingUpdater(nil)

	// Ensure the UpdaterKey() function returns the correct key
	assert.Equal(t, fcu.UpdaterKey(), CallbackKeyForPairingUpdate)
}

func TestPairingUpdater_FilterPairingListByEndpoint(t *testing.T) {
	// Define some test input values
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pairingList := []epochstoragetypes.StakeEntry{
		epochstoragetypes.StakeEntry{
			Address: "0x1234",
			Chain:   "eth",
			Endpoints: []epochstoragetypes.Endpoint{
				epochstoragetypes.Endpoint{
					UseType:     "eth",
					IPPORT:      "127.0.0.1:8545",
					Geolocation: 123,
				},
			},
		},
	}
	rpcEndpoint := lavasession.RPCEndpoint{
		ApiInterface: "eth",
		ChainID:      "eth",
		Geolocation:  123,
	}
	epoch := uint64(10)

	// Create the PairingUpdater and call the method
	pu := &PairingUpdater{
		stateQuery: mock.MockPairingStateQuery{},
	}
	filteredPairingList, err := pu.filterPairingListByEndpoint(ctx, pairingList, rpcEndpoint, epoch)

	// Assert that the output is as expected
	expectedPairingList := []*lavasession.ConsumerSessionsWithProvider{
		&lavasession.ConsumerSessionsWithProvider{
			PublicLavaAddress: "0x1234",
			Endpoints: []*lavasession.Endpoint{
				&lavasession.Endpoint{
					NetworkAddress:     "127.0.0.1:8545",
					Enabled:            true,
					Client:             nil,
					ConnectionRefusals: 0,
				},
			},
			Sessions:        make(map[int64]*lavasession.SingleConsumerSession),
			MaxComputeUnits: uint64(10),
			ReliabilitySent: false,
			PairingEpoch:    epoch,
		},
	}
	assert.Nil(t, err)
	assert.Equal(t, expectedPairingList, filteredPairingList)

	// Expect error ip maxCU return an error
	filteredPairingList, err = pu.filterPairingListByEndpoint(ctx, pairingList, rpcEndpoint, uint64(mock.InvalidBlockNumber))

	assert.Error(t, err)

	// Change apiInterface so no endpoint match
	rpcEndpointNoMatch := lavasession.RPCEndpoint{
		ApiInterface: "",
		ChainID:      "",
		Geolocation:  1,
	}

	// Expect error if no relevant endpoints
	filteredPairingList, err = pu.filterPairingListByEndpoint(ctx, pairingList, rpcEndpointNoMatch, uint64(mock.InvalidBlockNumber))

	assert.Error(t, err)

	// remove endpoints from pairingList
	noEndpointsPairingList := make([]epochstoragetypes.StakeEntry, len(pairingList))
	copy(noEndpointsPairingList, pairingList)
	noEndpointsPairingList[0].Endpoints = []epochstoragetypes.Endpoint{}

	// Expect error if maxCU return an error
	filteredPairingList, err = pu.filterPairingListByEndpoint(ctx, noEndpointsPairingList, rpcEndpoint, uint64(mock.InvalidBlockNumber))

	assert.Error(t, err)
}

func TestPairingUpdater_RegisterPairing(t *testing.T) {
	// Create a new PairingUpdater instance with a mocked StateQuery.
	pu := &PairingUpdater{
		consumerSessionManagersMap: map[string][]*lavasession.ConsumerSessionManager{
			"LAV1": {},
		},
		stateQuery: mock.MockPairingStateQuery{},
	}

	// Create a mock context and consumer session manager.
	ctx := context.Background()
	consumerSessionManager := lavasession.NewConsumerSessionManager(&lavasession.RPCEndpoint{
		NetworkAddress: "127.0.0.1:8545",
		ChainID:        "LAV1",
		ApiInterface:   "rest",
		Geolocation:    1,
	})

	// Call RegisterPairing and verify that it returns no error.
	err := pu.RegisterPairing(ctx, consumerSessionManager)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify that the consumerSessionManager was added to the consumerSessionManagersMap.
	chainID := consumerSessionManager.RPCEndpoint().ChainID
	consumerSessionsManagersList, ok := pu.consumerSessionManagersMap[chainID]
	if !ok {
		t.Fatalf("expected consumerSessionManagersMap to contain chainID %s, but it does not", chainID)
	}
	if len(consumerSessionsManagersList) != 1 {
		t.Fatalf("expected consumerSessionManagersList to have length 1, but got %d", len(consumerSessionsManagersList))
	}
	if consumerSessionsManagersList[0] != consumerSessionManager {
		t.Fatalf("expected consumerSessionsManagersList[0] to be the same as consumerSessionManager, but they are different")
	}

	consumerSessionManager = lavasession.NewConsumerSessionManager(&lavasession.RPCEndpoint{
		NetworkAddress: "127.0.0.1:8545",
		ChainID:        "RANDOM",
		ApiInterface:   "rest",
		Geolocation:    1,
	})

	// Call RegisterPairing and verify that it returns no error.
	err = pu.RegisterPairing(ctx, consumerSessionManager)

	assert.Nil(t, err)

	// Create consumerSessionManager with invalid chainID
	consumerSessionManager = lavasession.NewConsumerSessionManager(&lavasession.RPCEndpoint{
		NetworkAddress: "127.0.0.1:8545",
		ChainID:        mock.InvalidChainID,
		ApiInterface:   "rest",
		Geolocation:    1,
	})

	// Call RegisterPairing and verify that it returns no error.
	err = pu.RegisterPairing(ctx, consumerSessionManager)

	assert.Error(t, err)
}

func TestPairingUpdater_UpdateAllProviders(t *testing.T) {
	// Define some test input values
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	consumerSessionManager := lavasession.NewConsumerSessionManager(&lavasession.RPCEndpoint{
		NetworkAddress: "127.0.0.1:8545",
		ChainID:        mock.InvalidChainID,
		ApiInterface:   "rest",
		Geolocation:    1,
	})

	pu := &PairingUpdater{
		stateQuery: mock.MockPairingStateQuery{},
	}

	// Call updateConsummerSessionManager it should return an error because no pairingList
	err := pu.updateConsummerSessionManager(ctx, []epochstoragetypes.StakeEntry{}, consumerSessionManager, 50)

	assert.Error(t, err)
}

func TestPairingUpdater_Update(t *testing.T) {
	// Initialize the PairingUpdater object with mock consumer state query
	pu := NewPairingUpdater(mock.MockPairingStateQuery{})

	pu.consumerSessionManagersMap = map[string][]*lavasession.ConsumerSessionManager{
		"LAV1": {lavasession.NewConsumerSessionManager(&lavasession.RPCEndpoint{
			NetworkAddress: "127.0.0.1:8545",
			ChainID:        "RANDOM",
			ApiInterface:   "rest",
			Geolocation:    1,
		})},
	}

	lastBlock := int64(100)

	// If not consumer session return 0
	pu.Update(lastBlock)

	// Ensure the nextBlockForUpdate is equal to lastBlock
	assert.Equal(t, uint64(lastBlock), pu.nextBlockForUpdate)

	// On invalid block number expect error
	pu.Update(mock.InvalidBlockNumber)

	assert.Equal(t, uint64(lastBlock+1), pu.nextBlockForUpdate)

	// Lower last block should not change
	lowerLastBlock := int64(1)

	// Set nextBlockForUpdate to latest block
	pu.Update(lowerLastBlock)

	// Ensure the nextBlockForUpdate is equal to lastBlock
	assert.Equal(t, pu.nextBlockForUpdate, uint64(lastBlock+1))
}
