package lavasession

import (
	"testing"

	"github.com/stretchr/testify/require"
)

const (
	testNumberOfBlocksKeptInMemory = 100
	relayCu                        = uint64(10)
	epoch1                         = uint64(10)
	sessionId                      = uint64(123)
	dataReliabilitySessionId       = uint64(0)
	relayNumber                    = uint64(1)
	maxCu                          = uint64(150)
	epoch2                         = testNumberOfBlocksKeptInMemory + epoch1
	consumerOneAddress             = "consumer1"
)

func initProviderSessionManager() *ProviderSessionManager {
	return NewProviderSessionManager(&RPCProviderEndpoint{
		NetworkAddress: "127.0.0.1:6666",
		ChainID:        "LAV1",
		ApiInterface:   "tendermint",
		Geolocation:    1,
		NodeUrl:        []string{"http://localhost:666", "ws://localhost:666/websocket"},
	}, testNumberOfBlocksKeptInMemory)
}

func prepareSession(t *testing.T) (*ProviderSessionManager, *SingleProviderSession) {
	// initialize the struct
	psm := initProviderSessionManager()

	// get session for the first time
	sps, err := psm.GetSession(consumerOneAddress, epoch1, sessionId, relayNumber)

	// validate expected results
	require.Empty(t, psm.sessionsWithAllConsumers)
	require.Nil(t, sps)
	require.Error(t, err)
	require.True(t, ConsumerNotRegisteredYet.Is(err))

	// expect session to be missing, so we need to register it for the first time
	sps, err = psm.RegisterProviderSessionWithConsumer(consumerOneAddress, epoch1, sessionId, relayNumber, maxCu)

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
	return psm, sps
}

func TestHappyFlowPSM(t *testing.T) {
	// init test
	psm, sps := prepareSession(t)

	// on session done successfully
	err := psm.OnSessionDone(sps)

	// validate session done data
	require.Nil(t, err)
	require.Equal(t, sps.LatestRelayCu, uint64(0))
	require.Equal(t, sps.CuSum, relayCu)
	require.Equal(t, sps.SessionID, sessionId)
	require.Equal(t, sps.RelayNum, relayNumber)
	require.Equal(t, sps.PairingEpoch, epoch1)
}

// Test the basic functionality of the ProviderSessionsManager
func TestPSMEpochChange(t *testing.T) {
	// init test
	psm, sps := prepareSession(t)

	// on session done successfully
	err := psm.OnSessionDone(sps)

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
	sps, err = psm.GetSession(consumerOneAddress, epoch1, sessionId, relayNumber)

	// expect an error as we tried to get a session from a blocked epoch
	require.Error(t, err)
	require.True(t, InvalidEpochError.Is(err))
	require.Nil(t, sps)
}

func TestPSMOnSessionFailure(t *testing.T) {
	// init test
	psm, sps := prepareSession(t)

	// on session done successfully
	err := psm.OnSessionFailure(sps)

	// validate session done data
	require.Nil(t, err)
	require.Equal(t, sps.LatestRelayCu, uint64(0))
	require.Equal(t, sps.CuSum, uint64(0))
	require.Equal(t, sps.SessionID, sessionId)
	require.Equal(t, sps.RelayNum, uint64(0))
	require.Equal(t, sps.PairingEpoch, epoch1)
}

func TestPSMUpdateCu(t *testing.T) {
	// init test
	psm, sps := prepareSession(t)

	// on session done successfully
	err := psm.OnSessionDone(sps)

	// validate session done data
	require.Nil(t, err)

	err = psm.UpdateSessionCU(consumerOneAddress, epoch1, sessionId, maxCu)
	require.Nil(t, err)
	require.Equal(t, sps.userSessionsParent.epochData.UsedComputeUnits, maxCu)
}

func TestPSMUpdateCuMaxCuReached(t *testing.T) {
	// init test
	psm, sps := prepareSession(t)

	// on session done successfully
	err := psm.OnSessionDone(sps)

	// Update the session CU to reach the limit of the cu allowed
	err = psm.UpdateSessionCU(consumerOneAddress, epoch1, sessionId, maxCu)
	require.Nil(t, err)
	require.Equal(t, sps.userSessionsParent.epochData.UsedComputeUnits, maxCu)

	// get another session, this time sps is not nil as the session ID is already registered
	sps, err = psm.GetSession(consumerOneAddress, epoch1, sessionId, relayNumber+1)
	require.Nil(t, err)
	require.NotNil(t, sps)

	// prepare session with max cu overflow. expect an error
	err = sps.PrepareSessionForUsage(relayCu, maxCu+relayCu)
	require.Error(t, err)
	require.True(t, MaximumCULimitReachedByConsumer.Is(err))
}

func TestPSMCUMisMatch(t *testing.T) {
	// init test
	psm, sps := prepareSession(t)

	// on session done successfully
	err := psm.OnSessionDone(sps)

	// get another session
	sps, err = psm.GetSession(consumerOneAddress, epoch1, sessionId, relayNumber+1)
	require.Nil(t, err)
	require.NotNil(t, sps)

	// prepare session with wrong cu and expect mismatch, consumer wants to pay less than spec requires
	err = sps.PrepareSessionForUsage(relayCu+1, relayCu)
	require.Error(t, err)
	require.True(t, ProviderConsumerCuMisMatch.Is(err))
}

func TestPSMDataReliabilityHappyFlow(t *testing.T) {
	// initialize the struct
	psm := initProviderSessionManager()

	// get data reliability session
	sps, err := psm.GetDataReliabilitySession(consumerOneAddress, epoch1, dataReliabilitySessionId, relayNumber)

	// validate results
	require.Nil(t, err)
	require.NotNil(t, sps)

	// validate expected results
	require.Empty(t, psm.sessionsWithAllConsumers)
	require.NotEmpty(t, psm.dataReliabilitySessionsWithAllConsumers)
	require.Empty(t, psm.subscriptionSessionsWithAllConsumers)

	// // prepare session for usage
	sps.PrepareSessionForUsage(relayCu, relayCu)

	// validate session was prepared successfully
	require.Equal(t, relayCu, sps.LatestRelayCu)
	require.Equal(t, sps.CuSum, relayCu)
	require.Equal(t, sps.SessionID, sessionId)
	require.Equal(t, sps.RelayNum, relayNumber)
	require.Equal(t, sps.PairingEpoch, epoch1)
}
