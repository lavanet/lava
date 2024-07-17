package lavasession

import (
	"context"
	"fmt"
	"math"
	"testing"
	"time"

	"github.com/lavanet/lava/v2/protocol/common"
	"github.com/lavanet/lava/v2/utils"
	"github.com/lavanet/lava/v2/utils/rand"
	pairingtypes "github.com/lavanet/lava/v2/x/pairing/types"
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
	relayNumberBeforeUse           = uint64(0)
	maxCu                          = uint64(150)
	epoch2                         = testNumberOfBlocksKeptInMemory + epoch1
	consumerOneAddress             = "consumer1"
	pairedProviders                = int64(1)
	projectId                      = "projectIdTest"
)

func initProviderSessionManager() *ProviderSessionManager {
	return NewProviderSessionManager(&RPCProviderEndpoint{
		NetworkAddress: NetworkAddressData{Address: "127.0.0.1:6666"},
		ChainID:        "LAV1",
		ApiInterface:   "tendermint",
		Geolocation:    1,
		NodeUrls:       []common.NodeUrl{{Url: "http://localhost:666"}, {Url: "ws://localhost:666/websocket"}},
	}, testNumberOfBlocksKeptInMemory)
}

func prepareSession(t *testing.T, ctx context.Context) (*ProviderSessionManager, *SingleProviderSession) {
	// initialize the struct
	psm := initProviderSessionManager()

	// get session for the first time
	sps, err := psm.GetSession(ctx, consumerOneAddress, epoch1, sessionId, relayNumber, nil)

	// validate expected results
	require.Empty(t, psm.sessionsWithAllConsumers)
	require.Nil(t, sps)
	require.Error(t, err)
	require.True(t, ConsumerNotRegisteredYet.Is(err))

	// expect session to be missing, so we need to register it for the first time
	sps, err = psm.RegisterProviderSessionWithConsumer(ctx, consumerOneAddress, epoch1, sessionId, relayNumber, maxCu, pairedProviders, projectId, nil)

	// validate session was added
	require.NotEmpty(t, psm.sessionsWithAllConsumers)
	require.NoError(t, err)
	require.NotNil(t, sps)

	// prepare session for usage
	err = sps.PrepareSessionForUsage(ctx, relayCu, relayCu, 0, 0)

	// validate session was prepared successfully
	require.NoError(t, err)
	require.Equal(t, relayCu, sps.LatestRelayCu)
	require.Equal(t, sps.CuSum, relayCu)
	require.Equal(t, sps.SessionID, sessionId)
	require.Equal(t, sps.RelayNum, relayNumberBeforeUse)
	require.Equal(t, sps.PairingEpoch, epoch1)
	return psm, sps
}

func prepareSessionForVirtualEpochTests(t *testing.T, ctx context.Context) (*ProviderSessionManager, *SingleProviderSession) {
	// initialize the struct
	psm := initProviderSessionManager()

	// get session for the first time
	sps, err := psm.GetSession(ctx, consumerOneAddress, epoch1, sessionId, relayNumber, nil)

	// validate expected results
	require.Empty(t, psm.sessionsWithAllConsumers)
	require.Nil(t, sps)
	require.Error(t, err)
	require.True(t, ConsumerNotRegisteredYet.Is(err))

	// expect session to be missing, so we need to register it for the first time
	sps, err = psm.RegisterProviderSessionWithConsumer(ctx, consumerOneAddress, epoch1, sessionId, relayNumber, maxCuForVirtualEpoch, pairedProviders, projectId, nil)

	// validate session was added
	require.NotEmpty(t, psm.sessionsWithAllConsumers)
	require.NoError(t, err)
	require.NotNil(t, sps)

	// prepare session for usage and use all available cu = (virtualEpoch+1)*maxCuForVirtualEpoch
	err = sps.PrepareSessionForUsage(ctx, (virtualEpoch+1)*maxCuForVirtualEpoch, (virtualEpoch+1)*maxCuForVirtualEpoch, 0, virtualEpoch)

	// validate session was prepared successfully
	require.NoError(t, err)
	require.Equal(t, (virtualEpoch+1)*maxCuForVirtualEpoch, sps.LatestRelayCu)
	require.Equal(t, sps.CuSum, (virtualEpoch+1)*maxCuForVirtualEpoch)
	require.Equal(t, sps.SessionID, sessionId)
	require.Equal(t, sps.RelayNum, relayNumberBeforeUse)
	require.Equal(t, sps.PairingEpoch, epoch1)
	return psm, sps
}

func prepareBadgeSession(t *testing.T, ctx context.Context, badgeSessionIndex int) (*ProviderSessionManager, *SingleProviderSession) {
	// initialize the struct
	psm := initProviderSessionManager()

	// initialize badge
	badge := &pairingtypes.Badge{}
	// Get unique badgeUser for each goroutine
	badgeUser := fmt.Sprintf("sampleUser%d", badgeSessionIndex)
	// Get random BadgeCuAllocation between 100 and 5000
	rand.InitRandomSeed()
	badgeCuAllocation := rand.Intn(4901) + 100

	badge.CuAllocation = uint64(badgeCuAllocation)
	badge.Address = badgeUser

	// get session for the first time
	sps, err := psm.GetSession(ctx, consumerOneAddress, epoch1, sessionId, relayNumber, badge)

	// validate expected results
	require.Empty(t, psm.sessionsWithAllConsumers)
	require.Nil(t, sps)
	require.Error(t, err)
	require.True(t, ConsumerNotRegisteredYet.Is(err))

	// expect session to be missing, so we need to register it for the first time
	sps, err = psm.RegisterProviderSessionWithConsumer(ctx, consumerOneAddress, epoch1, sessionId, relayNumber, maxCu, pairedProviders, projectId, badge)

	// validate session was added
	require.NotEmpty(t, psm.sessionsWithAllConsumers)
	require.NoError(t, err)
	require.NotNil(t, sps)
	require.NotNil(t, sps.BadgeUserData)

	return psm, sps
}

func prepareBadgeSessionForUsage(t *testing.T, ctx context.Context, sps *SingleProviderSession) {
	// prepare session for usage
	err := sps.PrepareSessionForUsage(ctx, relayCu, relayCu, 0, 0)

	// validate session was prepared successfully
	require.NoError(t, err)
	require.Equal(t, relayCu, sps.LatestRelayCu)
	require.Equal(t, sps.CuSum, relayCu)
	require.Equal(t, sps.SessionID, sessionId)
	require.Equal(t, sps.RelayNum, relayNumberBeforeUse)
	require.Equal(t, sps.PairingEpoch, epoch1)
}

// func prepareDRSession(t *testing.T, ctx context.Context) (*ProviderSessionManager, *SingleProviderSession) {
// 	// initialize the struct
// 	psm := initProviderSessionManager()

// 	// get data reliability session
// 	sps, err := psm.GetDataReliabilitySession(consumerOneAddress, epoch1, dataReliabilitySessionId, relayNumber, pairedProviders)

// 	// validate results
// 	require.NoError(t, err)
// 	require.NotNil(t, sps)

// 	// validate expected results
// 	require.Empty(t, psm.sessionsWithAllConsumers)
// 	require.NotEmpty(t, psm.dataReliabilitySessionsWithAllConsumers)
// 	require.Empty(t, psm.subscriptionSessionsWithAllConsumers)

// 	// // prepare session for usage
// 	err = sps.PrepareSessionForUsage(ctx, relayCu, dataReliabilityRelayCu, 0)
// 	require.NoError(t, err)
// 	// validate session was prepared successfully
// 	require.Equal(t, dataReliabilityRelayCu, sps.LatestRelayCu)
// 	require.Equal(t, dataReliabilityRelayCu, sps.CuSum)
// 	require.Equal(t, dataReliabilitySessionId, sps.SessionID)
// 	require.Equal(t, sps.RelayNum, relayNumberBeforeUse)
// 	require.Equal(t, epoch1, sps.PairingEpoch)

// 	return psm, sps
// }

func TestHappyFlowPSM(t *testing.T) {
	// init test
	psm, sps := prepareSession(t, context.Background())

	// on session done successfully
	err := psm.OnSessionDone(sps, relayNumber)

	// validate session done data
	require.NoError(t, err)
	require.Equal(t, sps.LatestRelayCu, uint64(0))
	require.Equal(t, sps.CuSum, relayCu)
	require.Equal(t, sps.SessionID, sessionId)
	require.Equal(t, sps.RelayNum, relayNumber)
	require.Equal(t, sps.PairingEpoch, epoch1)
}

func TestMissingCu(t *testing.T) {
	ctx := context.Background()
	psm, sps := prepareSession(t, ctx)
	// on session done successfully
	err := psm.OnSessionDone(sps, relayNumber)
	// validate session done data
	require.NoError(t, err)
	// preparing a session again with the same relayRequestTotalCU will cause missing cu to trigger as we didn't provide enough cu
	// (relayCu * number of requests {2}) !=
	for i := 0; i < 10; i++ {
		sps.lock.Lock() // Lock session (usually should be locked by GetSession but in this test we set it manually)
		err = sps.PrepareSessionForUsage(ctx, relayCu, relayCu*uint64(i+1), 0.07, 0)
		require.NoError(t, err)
		err = psm.OnSessionDone(sps, relayNumber+uint64(i)+1)
		require.NoError(t, err)
	}
}

func TestMissingCuFailureOnThreshold(t *testing.T) {
	ctx := context.Background()
	psm, sps := prepareSession(t, ctx)
	// on session done successfully
	err := psm.OnSessionDone(sps, relayNumber)
	// validate session done successfully
	require.NoError(t, err)
	// preparing a session again with the same relayRequestTotalCU will cause missing cu to trigger as we didn't provide enough cu
	// (relayCu * number of requests {2}) !=
	sps.lock.Lock() // Lock session (usually should be locked by GetSession but in this test we set it manually)
	err = sps.PrepareSessionForUsage(ctx, relayCu, relayCu, 0.01, 0)
	require.True(t, ProviderConsumerCuMisMatch.Is(err))
}

func TestMissingMultipleMissingAttempts(t *testing.T) {
	ctx := context.Background()
	psm, sps := prepareSession(t, ctx)
	// on session done successfully
	err := psm.OnSessionDone(sps, relayNumber)
	// validate session done data
	require.NoError(t, err)
	// preparing a session again with the same relayRequestTotalCU will cause missing cu to trigger as we didn't provide enough cu
	// (relayCu * number of requests {2}) !=
	sps.lock.Lock() // Lock session (usually should be locked by GetSession but in this test we set it manually)
	err = sps.PrepareSessionForUsage(ctx, 1, relayCu, 0.5, 0)
	require.NoError(t, err)
	err = psm.OnSessionDone(sps, relayNumber+1)
	require.NoError(t, err)
	for i := 0; i < 10; i++ {
		sps.lock.Lock() // Lock session (usually should be locked by GetSession but in this test we set it manually)
		err = sps.PrepareSessionForUsage(ctx, 1, relayCu*uint64(i+1), 0.5, 0)
		require.NoError(t, err)
		err = psm.OnSessionDone(sps, relayNumber+uint64(i)+2)
		require.NoError(t, err)
	}
}

func TestHappyFlowBadgePSM(t *testing.T) {
	// init test
	psm, sps := prepareBadgeSession(t, context.Background(), 0)
	// prepare session for usage
	prepareBadgeSessionForUsage(t, context.Background(), sps)
	// on session done successfully
	err := psm.OnSessionDone(sps, relayNumber)

	// validate session done data
	require.NoError(t, err)
	require.Equal(t, sps.LatestRelayCu, uint64(0))
	require.Equal(t, sps.CuSum, relayCu)
	require.Equal(t, sps.BadgeUserData.UsedComputeUnits, relayCu)
	require.Equal(t, sps.SessionID, sessionId)
	require.Equal(t, sps.RelayNum, relayNumber)
	require.Equal(t, sps.PairingEpoch, epoch1)
}

func TestHappyFlowBadgePSMMultipleRoutines(t *testing.T) {
	// Num of goroutines to run
	numRoutines := 1000

	// Set the seed for the random number generator
	seed := time.Now().UnixNano()
	rand.SetSpecificSeed(seed)
	utils.LavaFormatInfo("started test with randomness, to reproduce use seed", utils.Attribute{Key: "seed", Value: seed})

	// A channel to track goroutine completion
	done := make(chan struct{})

	var totalRelayCU uint64 // total served CU for onSessionDone
	badgeSessionsUsedCU := make([]*ProviderSessionsEpochData, numRoutines)

	// Testing in multiple goroutines
	for i := 0; i < numRoutines; i++ {
		index := i // Create a new variable and assign loop variable's value to it
		go func() {
			// Decrement the wait group counter when the goroutine finishes
			defer func() {
				done <- struct{}{}
			}()
			// Random sleep before starting the goroutine
			sleepDuration := time.Duration(rand.Intn(3)) * time.Millisecond
			time.Sleep(sleepDuration)

			// Initialize the test
			psm, sps := prepareBadgeSession(t, context.Background(), index)
			// Put session into global array
			badgeSessionsUsedCU[index] = sps.BadgeUserData

			// Random sleep between GetSession and PrepareSessionForUsage
			sleepDuration = time.Duration(rand.Intn(3)) * time.Millisecond
			time.Sleep(sleepDuration)

			// Prepare session for usage
			prepareBadgeSessionForUsage(t, context.Background(), sps)
			// Randomly determine test success or failure with equal probability
			isSuccess := rand.Intn(2) == 0

			if isSuccess == true {
				// On session done successfully
				err := psm.OnSessionDone(sps, relayNumber)
				// Validate session done data
				require.NoError(t, err)
				require.Equal(t, sps.LatestRelayCu, uint64(0))
				require.Equal(t, sps.CuSum, relayCu)
				require.Equal(t, sps.BadgeUserData.UsedComputeUnits, relayCu)
				require.Equal(t, sps.SessionID, sessionId)
				require.Equal(t, sps.RelayNum, relayNumber)
				require.Equal(t, sps.PairingEpoch, epoch1)

				// Update used CU global variable
				totalRelayCU += relayCu
			} else {
				// On session failure
				err := psm.OnSessionFailure(sps, relayNumber)
				// validate session failure data
				require.NoError(t, err)
				require.Equal(t, sps.LatestRelayCu, uint64(0))
				require.Equal(t, sps.CuSum, uint64(0))
				require.Equal(t, sps.BadgeUserData.UsedComputeUnits, uint64(0))
				require.Equal(t, sps.SessionID, sessionId)
				require.Equal(t, sps.RelayNum, relayNumberBeforeUse)
				require.Equal(t, sps.PairingEpoch, epoch1)
			}
		}()
	}
	// Wait for all goroutines to finish
	for i := 0; i < numRoutines; i++ {
		<-done
	}

	// Calculate total used CU for all badge users
	var badgeUsedCU uint64
	for i := 0; i < numRoutines; i++ {
		badgeUsedCU += badgeSessionsUsedCU[i].UsedComputeUnits
	}

	require.Equal(t, totalRelayCU, badgeUsedCU)
}

func TestBadgePSMOnSessionFailure(t *testing.T) {
	// init test
	psm, sps := prepareBadgeSession(t, context.Background(), 0)
	// prepare session for usage
	prepareBadgeSessionForUsage(t, context.Background(), sps)
	// on session done successfully
	err := psm.OnSessionFailure(sps, relayNumber)

	// validate session done data
	require.NoError(t, err)
	require.Equal(t, sps.LatestRelayCu, uint64(0))
	require.Equal(t, sps.CuSum, uint64(0))
	require.Equal(t, sps.BadgeUserData.UsedComputeUnits, uint64(0))
	require.Equal(t, sps.SessionID, sessionId)
	require.Equal(t, sps.RelayNum, relayNumberBeforeUse)
	require.Equal(t, sps.PairingEpoch, epoch1)
}

func TestPSMPrepareTwice(t *testing.T) {
	// init test
	_, sps := prepareSession(t, context.Background())

	// prepare session for usage
	err := sps.PrepareSessionForUsage(context.Background(), relayCu, relayCu, 0, 0)
	require.Error(t, err)
	sps.lock.Unlock()
}

// Test the basic functionality of the ProviderSessionsManager
func TestPSMEpochChange(t *testing.T) {
	// init test
	psm, sps := prepareSession(t, context.Background())

	// on session done successfully
	err := psm.OnSessionDone(sps, relayNumber)

	// validate session done data
	require.NoError(t, err)
	require.Equal(t, sps.LatestRelayCu, uint64(0))
	require.Equal(t, sps.CuSum, relayCu)
	require.Equal(t, sps.SessionID, sessionId)
	require.Equal(t, sps.RelayNum, relayNumber)
	require.Equal(t, sps.PairingEpoch, epoch1)

	// update epoch to epoch2 height
	psm.UpdateEpoch(epoch2)

	// validate epoch update
	require.Equal(t, psm.blockedEpochHeight, epoch1)
	require.Empty(t, psm.sessionsWithAllConsumers)

	// try to verify we cannot get a session from epoch1 after we blocked it
	sps, err = psm.GetSession(context.Background(), consumerOneAddress, epoch1, sessionId, relayNumber, nil)

	// expect an error as we tried to get a session from a blocked epoch
	require.Error(t, err)
	require.True(t, InvalidEpochError.Is(err))
	require.Nil(t, sps)
}

func TestPSMOnSessionFailure(t *testing.T) {
	// init test
	psm, sps := prepareSession(t, context.Background())

	// on session done successfully
	err := psm.OnSessionFailure(sps, relayNumber)

	// validate session done data
	require.NoError(t, err)
	require.Equal(t, sps.LatestRelayCu, uint64(0))
	require.Equal(t, sps.CuSum, uint64(0))
	require.Equal(t, sps.SessionID, sessionId)
	require.Equal(t, sps.RelayNum, relayNumberBeforeUse)
	require.Equal(t, sps.PairingEpoch, epoch1)
}

func TestPSMUpdateCu(t *testing.T) {
	// init test
	psm, sps := prepareSession(t, context.Background())

	// on session done successfully
	err := psm.OnSessionDone(sps, relayNumber)

	// validate session done data
	require.NoError(t, err)

	err = psm.UpdateSessionCU(consumerOneAddress, epoch1, sessionId, maxCu)
	require.NoError(t, err)
	require.Equal(t, sps.userSessionsParent.epochData.UsedComputeUnits, maxCu)
}

func TestPSMUpdateCuMaxCuReached(t *testing.T) {
	ctx := context.Background()
	// init test
	psm, sps := prepareSession(t, ctx)

	// on session done successfully
	err := psm.OnSessionDone(sps, relayNumber)
	require.NoError(t, err)
	// Update the session CU to reach the limit of the cu allowed
	err = psm.UpdateSessionCU(consumerOneAddress, epoch1, sessionId, maxCu)
	require.NoError(t, err)
	require.Equal(t, sps.userSessionsParent.epochData.UsedComputeUnits, maxCu)

	// get another session, this time sps is not nil as the session ID is already registered
	sps, err = psm.GetSession(ctx, consumerOneAddress, epoch1, sessionId, relayNumber+1, nil)
	require.NoError(t, err)
	require.NotNil(t, sps)

	// prepare session with max cu overflow. expect an error
	err = sps.PrepareSessionForUsage(ctx, relayCu, maxCu+relayCu, 0, 0)
	require.Error(t, err)
	sps.lock.Unlock()
	require.True(t, MaximumCULimitReachedByConsumer.Is(err))
}

func TestHappyFlowPSMVirtualEpoch(t *testing.T) {
	// init test
	psm, sps := prepareSessionForVirtualEpochTests(t, context.Background())

	// on session done successfully
	err := psm.OnSessionDone(sps, relayNumber)

	// validate session done data
	require.NoError(t, err)
	require.Equal(t, sps.LatestRelayCu, uint64(0))
	require.Equal(t, sps.CuSum, (virtualEpoch+1)*maxCuForVirtualEpoch)
	require.Equal(t, sps.SessionID, sessionId)
	require.Equal(t, sps.RelayNum, relayNumber)
	require.Equal(t, sps.PairingEpoch, epoch1)
}

func TestPSMVirtualEpochUpdateCuMaxCuReached(t *testing.T) {
	ctx := context.Background()
	// init test
	psm, sps := prepareSessionForVirtualEpochTests(t, ctx)

	// on session done successfully
	err := psm.OnSessionDone(sps, relayNumber)
	require.NoError(t, err)
	// Update the session CU to reach the limit of the cu allowed
	err = psm.UpdateSessionCU(consumerOneAddress, epoch1, sessionId, (virtualEpoch+1)*maxCuForVirtualEpoch)
	require.NoError(t, err)
	require.Equal(t, sps.userSessionsParent.epochData.UsedComputeUnits, (virtualEpoch+1)*maxCuForVirtualEpoch)

	// get another session, this time sps is not nil as the session ID is already registered
	sps, err = psm.GetSession(ctx, consumerOneAddress, epoch1, sessionId, relayNumber+1, nil)
	require.NoError(t, err)
	require.NotNil(t, sps)

	// prepare session with max cu overflow. expect an error
	// as virtual epoch = 1, cu limit = (virtualEpoch+1)*maxCuForVirtualEpoch
	err = sps.PrepareSessionForUsage(ctx, relayCu, (virtualEpoch+1)*maxCuForVirtualEpoch+relayCu, 0, virtualEpoch)
	require.Error(t, err)
	sps.lock.Unlock()
	require.True(t, MaximumCULimitReachedByConsumer.Is(err))
}

func TestVirtualEpochMissingCu(t *testing.T) {
	ctx := context.Background()
	psm, sps := prepareSessionForVirtualEpochTests(t, ctx)
	// on session done successfully
	err := psm.OnSessionDone(sps, relayNumber)
	// validate session done data
	require.NoError(t, err)
	// preparing a session again with the same relayRequestTotalCU will cause missing cu to trigger as we didn't provide enough cu

	for i := 1; i <= 10; i++ {
		sps.lock.Lock() // Lock session (usually should be locked by GetSession but in this test we set it manually)
		err = sps.PrepareSessionForUsage(ctx, maxCuForVirtualEpoch, maxCuForVirtualEpoch*(virtualEpoch+uint64(i)+1), 0.07, virtualEpoch+uint64(i))
		require.NoError(t, err)
		err = psm.OnSessionDone(sps, relayNumber+uint64(i))
		require.NoError(t, err)
	}
}

func TestPSMCUMisMatch(t *testing.T) {
	ctx := context.Background()
	// init test
	psm, sps := prepareSession(t, ctx)

	// on session done successfully
	err := psm.OnSessionDone(sps, relayNumber)
	require.NoError(t, err)
	// get another session
	sps, err = psm.GetSession(ctx, consumerOneAddress, epoch1, sessionId, relayNumber+1, nil)
	require.NoError(t, err)
	require.NotNil(t, sps)

	// prepare session with wrong cu and expect mismatch, consumer wants to pay less than spec requires
	err = sps.PrepareSessionForUsage(ctx, relayCu+1, relayCu, 0, 0)
	require.Error(t, err)
	sps.lock.Unlock()
	require.True(t, ProviderConsumerCuMisMatch.Is(err))
}

func TestPSMSubscribeHappyFlowProcessUnsubscribe(t *testing.T) {
	// init test
	psm, sps := prepareSession(t, context.Background())

	// validate subscription map is empty
	require.Empty(t, psm.sessionsWithAllConsumers[epoch1].sessionMap[projectId].ongoingSubscriptions)

	// subscribe
	var channel chan interface{}
	subscription := &RPCSubscription{
		Id:                   subscriptionID,
		Sub:                  nil,
		SubscribeRepliesChan: channel,
	}
	psm.ReleaseSessionAndCreateSubscription(sps, subscription, consumerOneAddress, epoch1, relayNumber)

	// verify state after subscription creation
	require.True(t, LockMisUseDetectedError.Is(sps.VerifyLock())) // validating session was unlocked.
	require.NotEmpty(t, psm.sessionsWithAllConsumers)
	_, foundSubscription := psm.sessionsWithAllConsumers[epoch1].sessionMap[projectId].ongoingSubscriptions[subscriptionID]
	require.True(t, foundSubscription)

	err := psm.ProcessUnsubscribe("unsubscribe", subscriptionID, consumerOneAddress, epoch1)
	require.True(t, SubscriptionPointerIsNilError.Is(err))
	require.Empty(t, psm.sessionsWithAllConsumers[epoch1].sessionMap[projectId].ongoingSubscriptions)
}

func TestPSMSubscribeHappyFlowProcessUnsubscribeUnsubscribeAll(t *testing.T) {
	// init test
	psm, sps := prepareSession(t, context.Background())

	// validate subscription map is empty
	require.Empty(t, psm.sessionsWithAllConsumers[epoch1].sessionMap[projectId].ongoingSubscriptions)

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
	psm.ReleaseSessionAndCreateSubscription(sps, subscription, consumerOneAddress, epoch1, relayNumber)

	sps, err := psm.GetSession(context.Background(), consumerOneAddress, epoch1, sessionId, relayNumber+1, nil)
	require.NoError(t, err)
	require.NotNil(t, sps)

	// create 2nd subscription
	psm.ReleaseSessionAndCreateSubscription(sps, subscription2, consumerOneAddress, epoch1, relayNumber+1)

	// verify state after subscription creation
	require.True(t, LockMisUseDetectedError.Is(sps.VerifyLock())) // validating session was unlocked.
	require.NotEmpty(t, psm.sessionsWithAllConsumers)
	_, foundSubscription := psm.sessionsWithAllConsumers[epoch1].sessionMap[projectId].ongoingSubscriptions[subscriptionID]
	require.True(t, foundSubscription)
	_, foundSubscription2 := psm.sessionsWithAllConsumers[epoch1].sessionMap[projectId].ongoingSubscriptions[subscriptionID2]
	require.True(t, foundSubscription2)

	err = psm.ProcessUnsubscribe(TendermintUnsubscribeAll, subscriptionID, consumerOneAddress, epoch1)
	require.True(t, SubscriptionPointerIsNilError.Is(err))
	require.Empty(t, psm.sessionsWithAllConsumers[epoch1].sessionMap[projectId].ongoingSubscriptions)
}

func TestPSMSubscribeHappyFlowProcessUnsubscribeUnsubscribeOneOutOfTwo(t *testing.T) {
	// init test
	psm, sps := prepareSession(t, context.Background())

	// validate subscription map is empty
	require.Empty(t, psm.sessionsWithAllConsumers[epoch1].sessionMap[projectId].ongoingSubscriptions)

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
	psm.ReleaseSessionAndCreateSubscription(sps, subscription, consumerOneAddress, epoch1, relayNumber)
	// create 2nd subscription as we release the session we can just ask for it again with relayNumber + 1
	sps, err := psm.GetSession(context.Background(), consumerOneAddress, epoch1, sessionId, relayNumber+1, nil)
	require.NoError(t, err)
	psm.ReleaseSessionAndCreateSubscription(sps, subscription2, consumerOneAddress, epoch1, relayNumber+1)

	err = psm.ProcessUnsubscribe("unsubscribeOne", subscriptionID, consumerOneAddress, epoch1)
	require.True(t, SubscriptionPointerIsNilError.Is(err))
	require.NotEmpty(t, psm.sessionsWithAllConsumers[epoch1].sessionMap[projectId].ongoingSubscriptions)
	_, foundId2 := psm.sessionsWithAllConsumers[epoch1].sessionMap[projectId].ongoingSubscriptions[subscriptionID2]
	require.True(t, foundId2)
}

func TestPSMSubscribeHappyFlowSubscriptionEnded(t *testing.T) {
	// init test
	psm, sps := prepareSession(t, context.Background())

	// validate subscription map is empty
	require.Empty(t, psm.sessionsWithAllConsumers[epoch1].sessionMap[projectId].ongoingSubscriptions)

	// subscribe
	var channel chan interface{}
	subscription := &RPCSubscription{
		Id:                   subscriptionID,
		Sub:                  nil,
		SubscribeRepliesChan: channel,
	}
	psm.ReleaseSessionAndCreateSubscription(sps, subscription, consumerOneAddress, epoch1, relayNumber)

	// verify state after subscription creation
	require.True(t, LockMisUseDetectedError.Is(sps.VerifyLock())) // validating session was unlocked.
	require.NotEmpty(t, psm.sessionsWithAllConsumers)
	_, foundSubscription := psm.sessionsWithAllConsumers[epoch1].sessionMap[projectId].ongoingSubscriptions[subscriptionID]
	require.True(t, foundSubscription)

	psm.SubscriptionEnded(consumerOneAddress, epoch1, subscriptionID)
	require.Empty(t, psm.sessionsWithAllConsumers[epoch1].sessionMap[projectId].ongoingSubscriptions)
}

func TestPSMSubscribeHappyFlowSubscriptionEndedOneOutOfTwo(t *testing.T) {
	// init test
	psm, sps := prepareSession(t, context.Background())

	// validate subscription map is empty
	require.Empty(t, psm.sessionsWithAllConsumers[epoch1].sessionMap[projectId].ongoingSubscriptions)

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
	psm.ReleaseSessionAndCreateSubscription(sps, subscription, consumerOneAddress, epoch1, relayNumber)
	// create 2nd subscription as we release the session we can just ask for it again with relayNumber + 1
	sps, err := psm.GetSession(context.Background(), consumerOneAddress, epoch1, sessionId, relayNumber+1, nil)
	require.NoError(t, err)
	psm.ReleaseSessionAndCreateSubscription(sps, subscription2, consumerOneAddress, epoch1, relayNumber)

	psm.SubscriptionEnded(consumerOneAddress, epoch1, subscriptionID)
	require.NotEmpty(t, psm.sessionsWithAllConsumers[epoch1].sessionMap[projectId].ongoingSubscriptions)
	_, foundId2 := psm.sessionsWithAllConsumers[epoch1].sessionMap[projectId].ongoingSubscriptions[subscriptionID2]
	require.True(t, foundId2)
}

func TestPSMSubscribeEpochChange(t *testing.T) {
	// init test
	psm, sps := prepareSession(t, context.Background())

	// validate subscription map is empty
	require.Empty(t, psm.sessionsWithAllConsumers[epoch1].sessionMap[projectId].ongoingSubscriptions)

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
	psm.ReleaseSessionAndCreateSubscription(sps, subscription, consumerOneAddress, epoch1, relayNumber)
	// create 2nd subscription as we release the session we can just ask for it again with relayNumber + 1
	sps, err := psm.GetSession(context.Background(), consumerOneAddress, epoch1, sessionId, relayNumber+1, nil)
	require.NoError(t, err)
	psm.ReleaseSessionAndCreateSubscription(sps, subscription2, consumerOneAddress, epoch1, relayNumber+1)

	psm.UpdateEpoch(epoch2)
	require.Empty(t, psm.sessionsWithAllConsumers[epoch2])
}

type testSessionData struct {
	currentCU uint64
	inUse     bool
	sessionID uint64
	relayNum  uint64
	epoch     uint64
	session   *SingleProviderSession
	history   []string
}

// this test is running sessions and usage in a sync way to see integrity of behavior, opening and closing of sessions is separate
func TestPSMUsageSync(t *testing.T) {
	psm := NewProviderSessionManager(&RPCProviderEndpoint{
		NetworkAddress: NetworkAddressData{Address: "127.0.0.1:6666"},
		ChainID:        "LAV1",
		ApiInterface:   "tendermint",
		Geolocation:    1,
		NodeUrls:       []common.NodeUrl{{Url: "http://localhost:666"}, {Url: "ws://localhost:666/websocket"}},
	}, 20)
	seed := time.Now().UnixNano()
	rand.SetSpecificSeed(seed)
	utils.LavaFormatInfo("started test with randomness, to reproduce use seed", utils.Attribute{Key: "seed", Value: seed})
	consumerAddress := "stub-consumer"
	maxCuForConsumer := uint64(math.MaxInt64)
	numSessions := 5
	psm.UpdateEpoch(10)
	sessionsStore := initSessionStore(numSessions, 10)
	sessionsStoreTooAdvanced := initSessionStore(numSessions, 15) // sessionIDs will overlap, this is intentional
	// an attempt is either a valid opening, valid closing, invalid opening, erroring session, epoch too advanced usage
	simulateUsageOnSessionsStore := func(attemptsNum int, sessionsStoreArg []*testSessionData, needsRegister bool) {
		ctx := context.Background()
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
						err := psm.OnSessionDone(sessionStoreTest.session, sessionStoreTest.relayNum+1)
						require.NoError(t, err)
						sessionStoreTest.inUse = false
						sessionStoreTest.relayNum += 1
						sessionStoreTest.currentCU += relayCU
						sessionStoreTest.history = append(sessionStoreTest.history, ",OnSessionDone")
					} else {
						// error closing
						err := psm.OnSessionFailure(sessionStoreTest.session, sessionStoreTest.relayNum)
						require.NoError(t, err)
						sessionStoreTest.inUse = false
						sessionStoreTest.history = append(sessionStoreTest.history, ",OnSessionFailure")
					}
				} else {
					// try to use and fail
					relayNumToGet := sessionStoreTest.relayNum + uint64(rand.Intn(3))
					_, err := psm.GetSession(ctx, consumerAddress, sessionStoreTest.epoch, sessionStoreTest.sessionID, relayNumToGet, nil)
					require.Error(t, err)
					require.False(t, ConsumerNotRegisteredYet.Is(err))
					sessionStoreTest.history = append(sessionStoreTest.history, ",TryToUseAgain")
				}
			} else {
				// session not in use yet, so try to use it. we have several options:
				// 1. proper usage
				// 2. usage with wrong CU
				// 3. usage with wrong relay number
				// 4. usage with wrong epoch number
				choice := rand.Intn(2)
				if choice == 0 || sessionStoreTest.relayNum == 0 {
					// getSession should work
					session, err := psm.GetSession(ctx, consumerAddress, sessionStoreTest.epoch, sessionStoreTest.sessionID, sessionStoreTest.relayNum+1, nil)
					if sessionStoreTest.relayNum > 0 {
						// this is not a first relay so we expect this to work
						require.NoError(t, err, "sessionID %d relayNum %d storedRelayNum %d epoch %d, history %s", sessionStoreTest.sessionID, sessionStoreTest.relayNum+1, sessionStoreTest.session.RelayNum, sessionStoreTest.epoch, sessionStoreTest.history)
						require.Same(t, session, sessionStoreTest.session)
						sessionStoreTest.history = append(sessionStoreTest.history, ",GetSession")
					} else {
						// this can be a first relay or after an error, so allow not registered error
						if err != nil {
							// first relay
							require.True(t, ConsumerNotRegisteredYet.Is(err))
							require.True(t, needsRegister)
							needsRegister = false
							utils.LavaFormatInfo("registered session", utils.Attribute{Key: "sessionID", Value: sessionStoreTest.sessionID}, utils.Attribute{Key: "epoch", Value: sessionStoreTest.epoch})
							session, err := psm.RegisterProviderSessionWithConsumer(ctx, consumerAddress, sessionStoreTest.epoch, sessionStoreTest.sessionID, sessionStoreTest.relayNum+1, maxCuForConsumer, pairedProviders, projectId, nil)
							require.NoError(t, err)
							sessionStoreTest.session = session
							sessionStoreTest.history = append(sessionStoreTest.history, ",RegisterGet")
						} else {
							sessionStoreTest.session = session
							sessionStoreTest.history = append(sessionStoreTest.history, ",GetSession")
						}
					}
					choice := rand.Intn(2)
					switch choice {
					case 0:
						cuToUse := uint64(rand.Intn(10)) + 1
						err = sessionStoreTest.session.PrepareSessionForUsage(ctx, cuToUse, cuToUse+sessionStoreTest.currentCU, 0, 0)
						require.NoError(t, err)
						sessionStoreTest.inUse = true
						sessionStoreTest.history = append(sessionStoreTest.history, ",PrepareForUsage")
					case 1:
						cuToUse := uint64(rand.Intn(10)) + 1
						cuMissing := rand.Intn(int(cuToUse)) + 1
						if cuToUse+sessionStoreTest.currentCU <= uint64(cuMissing) {
							cuToUse += 1
						}
						err = sessionStoreTest.session.PrepareSessionForUsage(ctx, cuToUse, cuToUse+sessionStoreTest.currentCU-uint64(cuMissing), 0, 0)
						require.Error(t, err)
						sessionStoreTest.session.lock.Unlock()
						sessionStoreTest.history = append(sessionStoreTest.history, ",ErrCUPrepareForUsage")
					}
				} else {
					// getSession should fail
					relayNumSubs := rand.Intn(int(sessionStoreTest.relayNum) + 1) // [0,relayNum]
					_, err := psm.GetSession(context.Background(), consumerAddress, sessionStoreTest.epoch, sessionStoreTest.sessionID, sessionStoreTest.relayNum-uint64(relayNumSubs), nil)
					require.Error(t, err, "sessionID %d relayNum %d storedRelayNum %d", sessionStoreTest.sessionID, sessionStoreTest.relayNum-uint64(relayNumSubs), sessionStoreTest.session.RelayNum)
					_, err = psm.GetSession(context.Background(), consumerAddress, sessionStoreTest.epoch-1, sessionStoreTest.sessionID, sessionStoreTest.relayNum+1, nil)
					require.Error(t, err)
					_, err = psm.GetSession(context.Background(), consumerAddress, 5, sessionStoreTest.sessionID, sessionStoreTest.relayNum+1, nil)
					require.Error(t, err)
					sessionStoreTest.history = append(sessionStoreTest.history, ",ErrGet")
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
			err := psm.OnSessionDone(sessionStoreTest.session, sessionStoreTest.relayNum+1)
			require.NoError(t, err)
			sessionStoreTest.inUse = false
			sessionStoreTest.relayNum += 1
		} else {
			_, err := psm.GetSession(context.Background(), consumerAddress, sessionStoreTest.epoch, sessionStoreTest.sessionID, sessionStoreTest.relayNum+1, nil)
			require.Error(t, err)
		}
	}
	// .IsValidEpoch(uint64(request.RelaySession.Epoch))
	// .GetSession(context.Background(),consumerAddressString, uint64(request.Epoch), request.SessionId, request.RelayNum)
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
			history:   []string{},
		}
		utils.LavaFormatInfo("session", utils.Attribute{Key: "epoch", Value: epoch}, utils.Attribute{Key: "sessionID", Value: retSessions[i].sessionID})
	}
	return retSessions
}
