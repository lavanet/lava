package lavasession

import (
	"testing"

	"github.com/stretchr/testify/require"
)

const testNumberOfBlocksKeptInMemory = 100

func initProviderSessionManager() *ProviderSessionManager {
	return NewProviderSessionManager(&RPCProviderEndpoint{
		NetworkAddress: "127.0.0.1:6666",
		ChainID:        "LAV1",
		ApiInterface:   "tendermint",
		Geolocation:    1,
		NodeUrl:        []string{"http://localhost:666", "ws://localhost:666/websocket"},
	}, testNumberOfBlocksKeptInMemory)
}

// Test the basic functionality of the ProviderSessionsManager
func TestHappyFlowPSMWithEpochChange(t *testing.T) {
	// parameters for the test
	relayCu := uint64(10)
	epoch1 := uint64(10)
	sessionId := uint64(123)
	relayNumber := uint64(1)
	maxCu := uint64(150)
	epoch2 := testNumberOfBlocksKeptInMemory + epoch1

	// initialize the struct
	psm := initProviderSessionManager()

	// get session for the first time
	sps, err := psm.GetSession("consumer1", epoch1, sessionId, relayNumber)

	// validate expected results
	require.Empty(t, psm.sessionsWithAllConsumers)
	require.Nil(t, sps)
	require.Error(t, err)
	require.True(t, ConsumerNotRegisteredYet.Is(err))

	// expect session to be missing, so we need to register it for the first time
	sps, err = psm.RegisterProviderSessionWithConsumer("consumer1", epoch1, sessionId, relayNumber, maxCu)

	// validate session was added
	require.NotEmpty(t, psm.sessionsWithAllConsumers)
	require.Nil(t, err)
	require.NotNil(t, sps)

	// prepare session for usage
	sps.PrepareSessionForUsage(relayCu, relayCu)

	// validate session was prepared successfully
	require.Equal(t, relayCu, sps.LatestRelayCu)
	require.Equal(t, sps.CuSum, relayCu)
	require.Equal(t, sps.SessionID, sessionId)
	require.Equal(t, sps.RelayNum, relayNumber)
	require.Equal(t, sps.PairingEpoch, epoch1)

	// on session done successfully
	err = psm.OnSessionDone(sps)

	// validate session done data
	require.Nil(t, err)
	require.Equal(t, sps.LatestRelayCu, uint64(0))
	require.Equal(t, sps.CuSum, relayCu)
	require.Equal(t, sps.SessionID, sessionId)
	require.Equal(t, sps.RelayNum, relayNumber)
	require.Equal(t, sps.PairingEpoch, epoch1)

	// update epoch to epoch2 height
	psm.UpdateEpoch(epoch2)

	// validate epoch update
	require.Equal(t, psm.blockedEpochHeight, epoch1)
	require.Empty(t, psm.dataReliabilitySessionsWithAllConsumers)
	require.Empty(t, psm.sessionsWithAllConsumers)
	require.Empty(t, psm.subscriptionSessionsWithAllConsumers)

	// try to verify we cannot get a session from epoch1 after we blocked it
	sps, err = psm.GetSession("consumer1", epoch1, sessionId, relayNumber)

	// expect an error as we tried to get a session from a blocked epoch
	require.Error(t, err)
	require.True(t, InvalidEpochError.Is(err))
	require.Nil(t, sps)
}
