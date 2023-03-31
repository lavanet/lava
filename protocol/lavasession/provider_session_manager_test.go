package lavasession

import (
	"math"
	"math/rand"
	"testing"

	"github.com/lavanet/lava/protocol/common"
	"github.com/lavanet/lava/utils"
	"github.com/stretchr/testify/require"
)

const (
	testNumberOfBlocksKeptInMemory = 100
	relayCu                        = uint64(10)
	dataReliabilityRelayCu         = uint64(0)
	epoch1                         = uint64(10)
	sessionId                      = uint64(123)
	subscriptionID                 = "124"
	subscriptionID2                = "125"
	dataReliabilitySessionId       = uint64(0)
	relayNumber                    = uint64(1)
	maxCu                          = uint64(150)
	epoch2                         = testNumberOfBlocksKeptInMemory + epoch1
	consumerOneAddress             = "consumer1"
	selfProviderIndex              = int64(1)
)

func initProviderSessionManager() *ProviderSessionManager {
	return NewProviderSessionManager(&RPCProviderEndpoint{
		NetworkAddress: "127.0.0.1:6666",
		ChainID:        "LAV1",
		ApiInterface:   "tendermint",
		Geolocation:    1,
		NodeUrls:       []common.NodeUrl{{Url: "http://localhost:666"}, {Url: "ws://localhost:666/websocket"}},
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
	sps, err = psm.RegisterProviderSessionWithConsumer(consumerOneAddress, epoch1, sessionId, relayNumber, maxCu, selfProviderIndex)

	// validate session was added
	require.NotEmpty(t, psm.sessionsWithAllConsumers)
	require.Nil(t, err)
	require.NotNil(t, sps)

	// prepare session for usage
	err = sps.PrepareSessionForUsage(relayCu, relayCu, relayNumber)

	// validate session was prepared successfully
	require.Nil(t, err)
	require.Equal(t, sps.userSessionsParent.atomicReadProviderIndex(), selfProviderIndex)
	require.Equal(t, relayCu, sps.LatestRelayCu)
	require.Equal(t, sps.CuSum, relayCu)
	require.Equal(t, sps.SessionID, sessionId)
	require.Equal(t, sps.RelayNum, relayNumber)
	require.Equal(t, sps.PairingEpoch, epoch1)
	return psm, sps
}

func prepareDRSession(t *testing.T) (*ProviderSessionManager, *SingleProviderSession) {
	// initialize the struct
	psm := initProviderSessionManager()

	// get data reliability session
	sps, err := psm.GetDataReliabilitySession(consumerOneAddress, epoch1, dataReliabilitySessionId, relayNumber, selfProviderIndex)

	// validate results
	require.Nil(t, err)
	require.NotNil(t, sps)

	// validate expected results
	require.Empty(t, psm.sessionsWithAllConsumers)
	require.NotEmpty(t, psm.dataReliabilitySessionsWithAllConsumers)
	require.Empty(t, psm.subscriptionSessionsWithAllConsumers)

	// // prepare session for usage
	sps.PrepareSessionForUsage(relayCu, dataReliabilityRelayCu, relayNumber)

	// validate session was prepared successfully
	require.Equal(t, dataReliabilityRelayCu, sps.LatestRelayCu)
	require.Equal(t, dataReliabilityRelayCu, sps.CuSum)
	require.Equal(t, dataReliabilitySessionId, sps.SessionID)
	require.Equal(t, relayNumber, sps.RelayNum)
	require.Equal(t, epoch1, sps.PairingEpoch)

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

func TestPSMPrepareTwice(t *testing.T) {
	// init test
	_, sps := prepareSession(t)

	// prepare session for usage
	err := sps.PrepareSessionForUsage(relayCu, relayCu, relayNumber)
	require.Error(t, err)
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
	require.Nil(t, err)
	// Update the session CU to reach the limit of the cu allowed
	err = psm.UpdateSessionCU(consumerOneAddress, epoch1, sessionId, maxCu)
	require.Nil(t, err)
	require.Equal(t, sps.userSessionsParent.epochData.UsedComputeUnits, maxCu)

	// get another session, this time sps is not nil as the session ID is already registered
	sps, err = psm.GetSession(consumerOneAddress, epoch1, sessionId, relayNumber+1)
	require.Nil(t, err)
	require.NotNil(t, sps)

	// prepare session with max cu overflow. expect an error
	err = sps.PrepareSessionForUsage(relayCu, maxCu+relayCu, relayNumber)
	require.Error(t, err)
	require.True(t, MaximumCULimitReachedByConsumer.Is(err))
}

func TestPSMCUMisMatch(t *testing.T) {
	// init test
	psm, sps := prepareSession(t)

	// on session done successfully
	err := psm.OnSessionDone(sps)
	require.Nil(t, err)
	// get another session
	sps, err = psm.GetSession(consumerOneAddress, epoch1, sessionId, relayNumber+1)
	require.Nil(t, err)
	require.NotNil(t, sps)

	// prepare session with wrong cu and expect mismatch, consumer wants to pay less than spec requires
	err = sps.PrepareSessionForUsage(relayCu+1, relayCu, relayNumber)
	require.Error(t, err)
	require.True(t, ProviderConsumerCuMisMatch.Is(err))
}

func TestPSMDataReliabilityHappyFlow(t *testing.T) {
	// prepare data reliability session
	psm, sps := prepareDRSession(t)

	// perform session done
	psm.OnSessionDone(sps)

	// validate session done information is valid.
	require.Equal(t, dataReliabilityRelayCu, sps.LatestRelayCu)
	require.Equal(t, dataReliabilityRelayCu, sps.CuSum)
	require.Equal(t, dataReliabilitySessionId, sps.SessionID)
	require.Equal(t, relayNumber, sps.RelayNum)
	require.Equal(t, epoch1, sps.PairingEpoch)
}

func TestPSMDataReliabilityTwicePerEpoch(t *testing.T) {
	// prepare data reliability session
	psm, sps := prepareDRSession(t)

	// perform session done
	psm.OnSessionDone(sps)

	// validate session done information is valid.
	require.Equal(t, dataReliabilityRelayCu, sps.LatestRelayCu)
	require.Equal(t, dataReliabilityRelayCu, sps.CuSum)
	require.Equal(t, dataReliabilitySessionId, sps.SessionID)
	require.Equal(t, relayNumber, sps.RelayNum)
	require.Equal(t, epoch1, sps.PairingEpoch)

	// try to get a data reliability session again.
	sps, err := psm.GetDataReliabilitySession(consumerOneAddress, epoch1, dataReliabilitySessionId, relayNumber, selfProviderIndex)

	// validate we cant get more than one data reliability session per epoch (might change in the future)
	require.Error(t, err)
	require.True(t, DataReliabilitySessionAlreadyUsedError.Is(err)) // validate error is what we expect.
	require.Nil(t, sps)
}

func TestPSMDataReliabilitySessionFailure(t *testing.T) {
	// prepare data reliability session
	psm, sps := prepareDRSession(t)

	// perform session failure.
	psm.OnSessionFailure(sps)

	// validate on session failure that the relay number was subtracted
	require.Equal(t, dataReliabilityRelayCu, sps.LatestRelayCu)
	require.Equal(t, dataReliabilityRelayCu, sps.CuSum)
	require.Equal(t, dataReliabilitySessionId, sps.SessionID)
	require.Equal(t, relayNumber-1, sps.RelayNum)
	require.Equal(t, epoch1, sps.PairingEpoch)
}

func TestPSMDataReliabilityRetryAfterFailure(t *testing.T) {
	// prepare data reliability session
	psm, sps := prepareDRSession(t)

	// perform session failure.
	psm.OnSessionFailure(sps)

	// validate on session failure that the relay number was subtracted
	require.Equal(t, dataReliabilityRelayCu, sps.LatestRelayCu)
	require.Equal(t, dataReliabilityRelayCu, sps.CuSum)
	require.Equal(t, dataReliabilitySessionId, sps.SessionID)
	require.Equal(t, relayNumber-1, sps.RelayNum)
	require.Equal(t, epoch1, sps.PairingEpoch)

	// try to get a data reliability session again.
	sps, err := psm.GetDataReliabilitySession(consumerOneAddress, epoch1, dataReliabilitySessionId, relayNumber, selfProviderIndex)

	// validate we can get a data reliability session if we failed before
	require.Nil(t, err)
	require.NotNil(t, sps)

	// // prepare session for usage
	sps.PrepareSessionForUsage(relayCu, dataReliabilityRelayCu, relayNumber)

	// validate session was prepared successfully
	require.Equal(t, dataReliabilityRelayCu, sps.LatestRelayCu)
	require.Equal(t, dataReliabilityRelayCu, sps.CuSum)
	require.Equal(t, dataReliabilitySessionId, sps.SessionID)
	require.Equal(t, relayNumber, sps.RelayNum)
	require.Equal(t, epoch1, sps.PairingEpoch)

	// perform session done
	psm.OnSessionDone(sps)

	// validate session done information is valid.
	require.Equal(t, dataReliabilityRelayCu, sps.LatestRelayCu)
	require.Equal(t, dataReliabilityRelayCu, sps.CuSum)
	require.Equal(t, dataReliabilitySessionId, sps.SessionID)
	require.Equal(t, relayNumber, sps.RelayNum)
	require.Equal(t, epoch1, sps.PairingEpoch)
}

func TestPSMDataReliabilityEpochChange(t *testing.T) {
	// prepare data reliability session
	psm, sps := prepareDRSession(t)

	// perform session done.
	psm.OnSessionDone(sps)

	// update epoch to epoch2 height
	psm.UpdateEpoch(epoch2)

	// validate epoch update
	require.Equal(t, psm.blockedEpochHeight, epoch1)
	require.Empty(t, psm.dataReliabilitySessionsWithAllConsumers)
}

func TestPSMDataReliabilitySessionFailureEpochChange(t *testing.T) {
	// prepare data reliability session
	psm, sps := prepareDRSession(t)

	// perform session done.
	psm.OnSessionFailure(sps)

	// update epoch to epoch2 height
	psm.UpdateEpoch(epoch2)

	// validate epoch update
	require.Equal(t, psm.blockedEpochHeight, epoch1)
	require.Empty(t, psm.dataReliabilitySessionsWithAllConsumers)
}

func TestPSMSubscribeHappyFlowProcessUnsubscribe(t *testing.T) {
	// init test
	psm, sps := prepareSession(t)

	// validate subscription map is empty
	require.Empty(t, psm.subscriptionSessionsWithAllConsumers)

	// subscribe
	var channel chan interface{}
	subscription := &RPCSubscription{
		Id:                   subscriptionID,
		Sub:                  nil,
		SubscribeRepliesChan: channel,
	}
	psm.ReleaseSessionAndCreateSubscription(sps, subscription, consumerOneAddress, epoch1)

	// verify state after subscription creation
	require.True(t, LockMisUseDetectedError.Is(sps.VerifyLock())) // validating session was unlocked.
	require.NotEmpty(t, psm.sessionsWithAllConsumers)
	require.NotEmpty(t, psm.subscriptionSessionsWithAllConsumers)
	_, foundEpoch := psm.subscriptionSessionsWithAllConsumers[epoch1]
	require.True(t, foundEpoch)
	_, foundConsumer := psm.subscriptionSessionsWithAllConsumers[epoch1].subscriptionMap[consumerOneAddress]
	require.True(t, foundConsumer)
	_, foundSubscription := psm.subscriptionSessionsWithAllConsumers[epoch1].subscriptionMap[consumerOneAddress][subscriptionID]
	require.True(t, foundSubscription)

	err := psm.ProcessUnsubscribe("unsubscribe", subscriptionID, consumerOneAddress, epoch1)
	require.True(t, SubscriptionPointerIsNilError.Is(err))
	require.Empty(t, psm.subscriptionSessionsWithAllConsumers[epoch1].subscriptionMap[consumerOneAddress])
}

func TestPSMSubscribeHappyFlowProcessUnsubscribeUnsubscribeAll(t *testing.T) {
	// init test
	psm, sps := prepareSession(t)

	// validate subscription map is empty
	require.Empty(t, psm.subscriptionSessionsWithAllConsumers)

	// subscribe
	var channel chan interface{}
	subscription := &RPCSubscription{
		Id:                   subscriptionID,
		Sub:                  nil,
		SubscribeRepliesChan: channel,
	}
	subscription2 := &RPCSubscription{
		Id:                   subscriptionID2,
		Sub:                  nil,
		SubscribeRepliesChan: channel,
	}
	psm.ReleaseSessionAndCreateSubscription(sps, subscription, consumerOneAddress, epoch1)

	sps, err := psm.GetSession(consumerOneAddress, epoch1, sessionId, relayNumber+1)
	require.Nil(t, err)
	require.NotNil(t, sps)

	// create 2nd subscription
	psm.ReleaseSessionAndCreateSubscription(sps, subscription2, consumerOneAddress, epoch1)

	// verify state after subscription creation
	require.True(t, LockMisUseDetectedError.Is(sps.VerifyLock())) // validating session was unlocked.
	require.NotEmpty(t, psm.sessionsWithAllConsumers)
	require.NotEmpty(t, psm.subscriptionSessionsWithAllConsumers)
	_, foundEpoch := psm.subscriptionSessionsWithAllConsumers[epoch1]
	require.True(t, foundEpoch)
	_, foundConsumer := psm.subscriptionSessionsWithAllConsumers[epoch1].subscriptionMap[consumerOneAddress]
	require.True(t, foundConsumer)
	_, foundSubscription := psm.subscriptionSessionsWithAllConsumers[epoch1].subscriptionMap[consumerOneAddress][subscriptionID]
	require.True(t, foundSubscription)
	_, foundSubscription2 := psm.subscriptionSessionsWithAllConsumers[epoch1].subscriptionMap[consumerOneAddress][subscriptionID2]
	require.True(t, foundSubscription2)

	err = psm.ProcessUnsubscribe(TendermintUnsubscribeAll, subscriptionID, consumerOneAddress, epoch1)
	require.True(t, SubscriptionPointerIsNilError.Is(err))
	require.Empty(t, psm.subscriptionSessionsWithAllConsumers[epoch1].subscriptionMap[consumerOneAddress])
}
func TestPSMSubscribeHappyFlowProcessUnsubscribeUnsubscribeOneOutOfTwo(t *testing.T) {
	// init test
	psm, sps := prepareSession(t)

	// validate subscription map is empty
	require.Empty(t, psm.subscriptionSessionsWithAllConsumers)

	// subscribe
	var channel chan interface{}
	subscription := &RPCSubscription{
		Id:                   subscriptionID,
		Sub:                  nil,
		SubscribeRepliesChan: channel,
	}
	subscription2 := &RPCSubscription{
		Id:                   subscriptionID2,
		Sub:                  nil,
		SubscribeRepliesChan: channel,
	}
	psm.ReleaseSessionAndCreateSubscription(sps, subscription, consumerOneAddress, epoch1)
	// create 2nd subscription as we release the session we can just ask for it again with relayNumber + 1
	sps, err := psm.GetSession(consumerOneAddress, epoch1, sessionId, relayNumber+1)
	require.Nil(t, err)
	psm.ReleaseSessionAndCreateSubscription(sps, subscription2, consumerOneAddress, epoch1)

	err = psm.ProcessUnsubscribe("unsubscribeOne", subscriptionID, consumerOneAddress, epoch1)
	require.True(t, SubscriptionPointerIsNilError.Is(err))
	require.NotEmpty(t, psm.subscriptionSessionsWithAllConsumers[epoch1].subscriptionMap[consumerOneAddress])
	_, foundId2 := psm.subscriptionSessionsWithAllConsumers[epoch1].subscriptionMap[consumerOneAddress][subscriptionID2]
	require.True(t, foundId2)
}

func TestPSMSubscribeHappyFlowSubscriptionEnded(t *testing.T) {
	// init test
	psm, sps := prepareSession(t)

	// validate subscription map is empty
	require.Empty(t, psm.subscriptionSessionsWithAllConsumers)

	// subscribe
	var channel chan interface{}
	subscription := &RPCSubscription{
		Id:                   subscriptionID,
		Sub:                  nil,
		SubscribeRepliesChan: channel,
	}
	psm.ReleaseSessionAndCreateSubscription(sps, subscription, consumerOneAddress, epoch1)

	// verify state after subscription creation
	require.True(t, LockMisUseDetectedError.Is(sps.VerifyLock())) // validating session was unlocked.
	require.NotEmpty(t, psm.sessionsWithAllConsumers)
	require.NotEmpty(t, psm.subscriptionSessionsWithAllConsumers)
	_, foundEpoch := psm.subscriptionSessionsWithAllConsumers[epoch1]
	require.True(t, foundEpoch)
	_, foundConsumer := psm.subscriptionSessionsWithAllConsumers[epoch1].subscriptionMap[consumerOneAddress]
	require.True(t, foundConsumer)
	_, foundSubscription := psm.subscriptionSessionsWithAllConsumers[epoch1].subscriptionMap[consumerOneAddress][subscriptionID]
	require.True(t, foundSubscription)

	psm.SubscriptionEnded(consumerOneAddress, epoch1, subscriptionID)
	require.Empty(t, psm.subscriptionSessionsWithAllConsumers[epoch1].subscriptionMap[consumerOneAddress])
}

func TestPSMSubscribeHappyFlowSubscriptionEndedOneOutOfTwo(t *testing.T) {
	// init test
	psm, sps := prepareSession(t)

	// validate subscription map is empty
	require.Empty(t, psm.subscriptionSessionsWithAllConsumers)

	// subscribe
	var channel chan interface{}
	subscription := &RPCSubscription{
		Id:                   subscriptionID,
		Sub:                  nil,
		SubscribeRepliesChan: channel,
	}
	subscription2 := &RPCSubscription{
		Id:                   subscriptionID2,
		Sub:                  nil,
		SubscribeRepliesChan: channel,
	}
	psm.ReleaseSessionAndCreateSubscription(sps, subscription, consumerOneAddress, epoch1)
	// create 2nd subscription as we release the session we can just ask for it again with relayNumber + 1
	sps, err := psm.GetSession(consumerOneAddress, epoch1, sessionId, relayNumber+1)
	require.Nil(t, err)
	psm.ReleaseSessionAndCreateSubscription(sps, subscription2, consumerOneAddress, epoch1)

	psm.SubscriptionEnded(consumerOneAddress, epoch1, subscriptionID)
	require.NotEmpty(t, psm.subscriptionSessionsWithAllConsumers[epoch1].subscriptionMap[consumerOneAddress])
	_, foundId2 := psm.subscriptionSessionsWithAllConsumers[epoch1].subscriptionMap[consumerOneAddress][subscriptionID2]
	require.True(t, foundId2)
}

func TestPSMSubscribeEpochChange(t *testing.T) {
	// init test
	psm, sps := prepareSession(t)

	// validate subscription map is empty
	require.Empty(t, psm.subscriptionSessionsWithAllConsumers)

	// subscribe
	var channel chan interface{}
	subscription := &RPCSubscription{
		Id:                   subscriptionID,
		Sub:                  nil,
		SubscribeRepliesChan: channel,
	}
	subscription2 := &RPCSubscription{
		Id:                   subscriptionID2,
		Sub:                  nil,
		SubscribeRepliesChan: channel,
	}
	psm.ReleaseSessionAndCreateSubscription(sps, subscription, consumerOneAddress, epoch1)
	// create 2nd subscription as we release the session we can just ask for it again with relayNumber + 1
	sps, err := psm.GetSession(consumerOneAddress, epoch1, sessionId, relayNumber+1)
	require.Nil(t, err)
	psm.ReleaseSessionAndCreateSubscription(sps, subscription2, consumerOneAddress, epoch1)

	psm.UpdateEpoch(epoch2)
	require.Empty(t, psm.subscriptionSessionsWithAllConsumers)
	require.Empty(t, psm.sessionsWithAllConsumers)
}

type testSessionData struct {
	currentCU uint64
	inUse     bool
	sessionID uint64
	relayNum  uint64
	epoch     uint64
	session   *SingleProviderSession
}

// this test is running sessions and usage in a sync way to see integrity of behavior, opening and closing of sessions is separate
func TestPSMUsageSync(t *testing.T) {
	psm := NewProviderSessionManager(&RPCProviderEndpoint{
		NetworkAddress: "127.0.0.1:6666",
		ChainID:        "LAV1",
		ApiInterface:   "tendermint",
		Geolocation:    1,
		NodeUrls:       []common.NodeUrl{{Url: "http://localhost:666"}, {Url: "ws://localhost:666/websocket"}},
	}, 20)

	consumerAddress := "stub-consumer"
	maxCuForConsumer := uint64(math.MaxInt64)
	selfProviderIndex := int64(0)
	numSessions := 5
	psm.UpdateEpoch(10)
	sessionsStore := initSessionStore(numSessions, 10)
	sessionsStoreTooAdvanced := initSessionStore(numSessions, 15) // sessionIDs will overlap, this is intentional
	// an attempt is either a valid opening, valid closing, invalid opening, erroring session, epoch too advanced usage
	simulateUsageOnSessionsStore := func(attemptsNum int, sessionsStoreArg []*testSessionData, needsRegister bool) {
		for attempts := 0; attempts < attemptsNum; attempts++ {
			// pick scenario:
			sessionIdx := rand.Intn(len(sessionsStoreArg))
			sessionStoreTest := sessionsStoreArg[sessionIdx]
			inUse := sessionStoreTest.inUse
			if inUse {
				// session is in use so either we close it or try to use and fail
				choice := rand.Intn(2)
				if choice == 0 {
					// close it
					choice = rand.Intn(2)
					// proper closing or error closing
					if choice == 0 {
						relayCU := sessionStoreTest.session.LatestRelayCu
						// proper closing
						err := psm.OnSessionDone(sessionStoreTest.session)
						require.NoError(t, err)
						sessionStoreTest.inUse = false
						sessionStoreTest.relayNum += 1
						sessionStoreTest.currentCU += relayCU
					} else {
						// error closing
						err := psm.OnSessionFailure(sessionStoreTest.session)
						require.NoError(t, err)
						sessionStoreTest.inUse = false
					}
				} else {
					// try to use and fail
					relayNumToGet := sessionStoreTest.relayNum + uint64(rand.Intn(3))
					_, err := psm.GetSession(consumerAddress, sessionStoreTest.epoch, sessionStoreTest.sessionID, relayNumToGet)
					require.Error(t, err)
					require.False(t, ConsumerNotRegisteredYet.Is(err))
				}
			} else {
				// session not in use yet, so try to use it. we have several options:
				// 1. proper usage /
				// 2. usage with wrong CU
				// 3. usage with wrong relay number
				// 4. usage with wrong epoch number
				choice := rand.Intn(2)
				if choice == 0 || sessionStoreTest.relayNum == 0 {
					// getSession should work
					session, err := psm.GetSession(consumerAddress, sessionStoreTest.epoch, sessionStoreTest.sessionID, sessionStoreTest.relayNum+1)
					if sessionStoreTest.relayNum > 0 {
						// this is not a first relay so we expect this to work
						require.NoError(t, err)
						require.Same(t, session, sessionStoreTest.session)
					} else {
						// this can be a first relay or after an error, so allow not registered error
						if err != nil {
							// first relay
							require.True(t, ConsumerNotRegisteredYet.Is(err))
							require.True(t, needsRegister)
							needsRegister = false
							utils.LavaFormatInfo("registered session", utils.Attribute{Key: "sessionID", Value: sessionStoreTest.sessionID}, utils.Attribute{Key: "epoch", Value: sessionStoreTest.epoch})
							session, err := psm.RegisterProviderSessionWithConsumer(consumerAddress, sessionStoreTest.epoch, sessionStoreTest.sessionID, sessionStoreTest.relayNum+1, maxCuForConsumer, selfProviderIndex)
							require.NoError(t, err)
							sessionStoreTest.session = session
						} else {
							sessionStoreTest.session = session
						}
					}
					choice := rand.Intn(2)
					switch choice {
					case 0:
						cuToUse := uint64(rand.Intn(10)) + 1
						err = sessionStoreTest.session.PrepareSessionForUsage(cuToUse, cuToUse+sessionStoreTest.currentCU, sessionStoreTest.relayNum+1)
						require.NoError(t, err)
						sessionStoreTest.inUse = true

					case 1:
						cuToUse := uint64(rand.Intn(10)) + 1
						cuMissing := rand.Intn(int(cuToUse)) + 1
						if cuToUse+sessionStoreTest.currentCU <= uint64(cuMissing) {
							cuToUse += 1
						}
						err = sessionStoreTest.session.PrepareSessionForUsage(cuToUse, cuToUse+sessionStoreTest.currentCU-uint64(cuMissing), sessionStoreTest.relayNum+1)
						require.Error(t, err)
					}
				} else {
					// getSession should fail
					relayNumSubs := rand.Intn(int(sessionStoreTest.relayNum) + 1) // [0,relayNum]
					_, err := psm.GetSession(consumerAddress, sessionStoreTest.epoch, sessionStoreTest.sessionID, sessionStoreTest.relayNum-uint64(relayNumSubs))
					require.Error(t, err, "sessionID %d relayNum %d storedRelayNum %d", sessionStoreTest.sessionID, sessionStoreTest.relayNum-uint64(relayNumSubs), sessionStoreTest.session.RelayNum)
					_, err = psm.GetSession(consumerAddress, sessionStoreTest.epoch-1, sessionStoreTest.sessionID, sessionStoreTest.relayNum+1)
					require.Error(t, err)
					_, err = psm.GetSession(consumerAddress, 5, sessionStoreTest.sessionID, sessionStoreTest.relayNum+1)
					require.Error(t, err)
				}
			}
		}
	}

	simulateUsageOnSessionsStore(500, sessionsStore, true)
	// now repeat with epoch advancement on consumer and provider node
	simulateUsageOnSessionsStore(100, sessionsStoreTooAdvanced, true)

	psm.UpdateEpoch(20) // update session, still within size, so shouldn't affect anything

	simulateUsageOnSessionsStore(500, sessionsStore, false)
	simulateUsageOnSessionsStore(100, sessionsStoreTooAdvanced, false)

	psm.UpdateEpoch(40) // update session, still within size, so shouldn't affect anything
	for attempts := 0; attempts < 100; attempts++ {
		// pick scenario:
		sessionIdx := rand.Intn(len(sessionsStore))
		sessionStoreTest := sessionsStore[sessionIdx]
		inUse := sessionStoreTest.inUse
		if inUse {
			err := psm.OnSessionDone(sessionStoreTest.session)
			require.NoError(t, err)
			sessionStoreTest.inUse = false
			sessionStoreTest.relayNum += 1
		} else {
			_, err := psm.GetSession(consumerAddress, sessionStoreTest.epoch, sessionStoreTest.sessionID, sessionStoreTest.relayNum+1)
			require.Error(t, err)
		}
	}
	// .IsValidEpoch(uint64(request.RelaySession.Epoch))
	// .GetSession(consumerAddressString, uint64(request.Epoch), request.SessionId, request.RelayNum)
	// on err: lavasession.ConsumerNotRegisteredYet.Is(err)
	// // .RegisterProviderSessionWithConsumer(consumerAddressString, uint64(request.Epoch), request.SessionId, request.RelayNum, maxCuForConsumer, selfProviderIndex)
	// .PrepareSessionForUsage(relayCU, request.RelaySession.CuSum, request.RelaySession.RelayNum)
	// simulate error: .OnSessionFailure(relaySession)
	// simulate success: .OnSessionDone(relaySession)
}

func initSessionStore(numSessions int, epoch uint64) []*testSessionData {
	retSessions := make([]*testSessionData, numSessions)
	for i := 0; i < numSessions; i++ {
		retSessions[i] = &testSessionData{
			currentCU: 0,
			inUse:     false,
			sessionID: uint64(i) + 1,
			relayNum:  0,
			epoch:     epoch,
			session:   nil,
		}
	}
	utils.LavaFormatInfo("sessions", utils.Attribute{Key: "epoch", Value: epoch}, utils.Attribute{Key: "data", Value: retSessions})
	return retSessions
}
