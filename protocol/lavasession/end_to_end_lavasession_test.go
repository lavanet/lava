package lavasession

import (
	"context"
	"testing"
	"time"

	"github.com/lavanet/lava/v2/protocol/common"
	"github.com/stretchr/testify/require"
)

func TestHappyFlowE2E(t *testing.T) {
	prepareSessionsWithFirstRelay(t, cuForFirstRequest)
}

func TestHappyFlowE2EEmergency(t *testing.T) {
	var skippedRelays uint64
	var successfulRelays uint64

	// 1 - consumer in emergency, provider not - fail
	// 2 both in emergency - ok
	// 3 - increase consumer virtual epoch - fail
	// 4 - increase provider virtual epoch - ok
	consumerVirtualEpochs := []uint64{1, 1, 2, 2}
	providerVirtualEpochs := []uint64{0, 1, 1, 2}

	csm, psm, ctx := prepareSessionsWithFirstRelay(t, maxCuForVirtualEpoch)
	successfulRelays++

	for i := 0; i < len(consumerVirtualEpochs); i++ {
		css, err := csm.GetSessions(ctx, maxCuForVirtualEpoch, NewUsedProviders(nil), servicedBlockNumber, "", nil, common.NO_STATE, consumerVirtualEpochs[i]) // get a session
		require.NoError(t, err)

		for _, cs := range css {
			require.NotNil(t, cs)
			require.Equal(t, cs.Epoch, csm.currentEpoch)
			require.Equal(t, cs.Session.LatestRelayCu, maxCuForVirtualEpoch)

			// Provider Side:

			sps, err := psm.GetSession(ctx, consumerOneAddress, cs.Session.Parent.PairingEpoch, uint64(cs.Session.SessionId), cs.Session.RelayNum-skippedRelays, nil)
			// validate expected results
			require.NotEmpty(t, psm.sessionsWithAllConsumers)
			require.NotNil(t, sps)
			require.NoError(t, err)
			require.False(t, ConsumerNotRegisteredYet.Is(err))

			// prepare session for usage
			// as some iterations are skipped so usedCu doesn't increase and relayRequestTotalCU won't change

			err = sps.PrepareSessionForUsage(ctx, maxCuForVirtualEpoch, cs.Session.LatestRelayCu+(skippedRelays*maxCuForVirtualEpoch), 0, providerVirtualEpochs[i])

			if consumerVirtualEpochs[i] != providerVirtualEpochs[i] {
				// provider is not in emergency mode, so prepare session will trigger an error
				require.Error(t, err)
				require.Equal(t, uint64(0), sps.LatestRelayCu)

				skippedRelays++

				err := csm.OnSessionFailure(cs.Session, nil)
				require.NoError(t, err)

				err = psm.OnSessionFailure(sps, cs.Session.RelayNum-skippedRelays)
				require.NoError(t, err)
			} else {
				successfulRelays++

				require.NoError(t, err)
				require.NoError(t, err)
				require.Equal(t, cs.Session.LatestRelayCu, sps.LatestRelayCu)

				err = psm.OnSessionDone(sps, cs.Session.RelayNum-skippedRelays)
				require.NoError(t, err)

				err = csm.OnSessionDone(cs.Session, servicedBlockNumber, maxCuForVirtualEpoch, time.Millisecond, cs.Session.CalculateExpectedLatency(2*time.Millisecond), (servicedBlockNumber - 1), 1, 1, false)
				require.NoError(t, err)
			}

			require.Equal(t, sps.CuSum, successfulRelays*maxCuForVirtualEpoch)
			require.Equal(t, sps.RelayNum, cs.Session.RelayNum-skippedRelays)

			// Consumer Side:

			require.Equal(t, cs.Session.CuSum, successfulRelays*maxCuForVirtualEpoch)
			require.Equal(t, cs.Session.LatestRelayCu, latestRelayCuAfterDone)
			require.Equal(t, cs.Session.RelayNum, successfulRelays+skippedRelays)
			require.Equal(t, cs.Session.LatestBlock, servicedBlockNumber)
		}
	}
}

func TestHappyFlowEmergencyInConsumer(t *testing.T) {
	csm, psm, ctx := prepareSessionsWithFirstRelay(t, maxCuForVirtualEpoch)

	css, err := csm.GetSessions(ctx, maxCuForVirtualEpoch, NewUsedProviders(nil), servicedBlockNumber, "", nil, common.NO_STATE, virtualEpoch) // get a session
	require.NoError(t, err)

	for _, cs := range css {
		require.NotNil(t, cs)
		require.Equal(t, cs.Epoch, csm.currentEpoch)
		require.Equal(t, cs.Session.LatestRelayCu, maxCuForVirtualEpoch)

		// Provider Side:

		sps, err := psm.GetSession(ctx, consumerOneAddress, cs.Session.Parent.PairingEpoch, uint64(cs.Session.SessionId), cs.Session.RelayNum, nil)
		// validate expected results
		require.NotEmpty(t, psm.sessionsWithAllConsumers)
		require.NotNil(t, sps)
		require.NoError(t, err)
		require.False(t, ConsumerNotRegisteredYet.Is(err))

		// prepare session for usage
		err = sps.PrepareSessionForUsage(ctx, maxCuForVirtualEpoch, cs.Session.LatestRelayCu, 0, 0)

		// validate session was not prepared successfully
		require.Error(t, err)
		require.Equal(t, cs.Session.LatestRelayCu, maxCuForVirtualEpoch)
		require.Equal(t, sps.CuSum, maxCuForVirtualEpoch)
		require.Equal(t, sps.SessionID, uint64(cs.Session.SessionId))
		require.Equal(t, sps.PairingEpoch, cs.Session.Parent.PairingEpoch)

		err = psm.OnSessionFailure(sps, cs.Session.RelayNum-1)
		require.Equal(t, sps.RelayNum, cs.Session.RelayNum-1)
		require.NoError(t, err)

		// Consumer Side:
		err = csm.OnSessionFailure(cs.Session, nil)
		require.NoError(t, err)
		require.Equal(t, cs.Session.CuSum, maxCuForVirtualEpoch)
		require.Equal(t, cs.Session.LatestRelayCu, latestRelayCuAfterDone)
		require.Equal(t, cs.Session.RelayNum, relayNumberAfterFirstCall+1)
		require.Equal(t, cs.Session.LatestBlock, servicedBlockNumber)
	}
}

func prepareSessionsWithFirstRelay(t *testing.T, cuForFirstRequest uint64) (*ConsumerSessionManager, *ProviderSessionManager, context.Context) {
	ctx := context.Background()

	// Consumer Side:
	csm := CreateConsumerSessionManager() // consumer session manager

	// Provider Side:
	psm := initProviderSessionManager() // provider session manager

	// Consumer Side:
	cswpList := make(map[uint64]*ConsumerSessionsWithProvider, 1)
	pairingEndpoints := make([]*Endpoint, 1)
	// we need a grpc server to connect to. so we use the public rpc endpoint for now.
	pairingEndpoints[0] = &Endpoint{NetworkAddress: grpcListener, Enabled: true, Connections: []*EndpointConnection{}, ConnectionRefusals: 0}
	cswpList[0] = &ConsumerSessionsWithProvider{
		PublicLavaAddress: "provider",
		Endpoints:         pairingEndpoints,
		Sessions:          map[int64]*SingleConsumerSession{},
		MaxComputeUnits:   200,
		PairingEpoch:      epoch1,
	}
	err := csm.UpdateAllProviders(epoch1, cswpList) // update the providers.
	require.NoError(t, err)
	// get single consumer session
	css, err := csm.GetSessions(ctx, cuForFirstRequest, NewUsedProviders(nil), servicedBlockNumber, "", nil, common.NO_STATE, 0) // get a session
	require.NoError(t, err)

	for _, cs := range css {
		require.NotNil(t, cs)
		require.Equal(t, cs.Epoch, csm.currentEpoch)
		require.Equal(t, cs.Session.LatestRelayCu, cuForFirstRequest)

		// Provider Side:

		sps, err := psm.GetSession(ctx, consumerOneAddress, cs.Session.Parent.PairingEpoch, uint64(cs.Session.SessionId), cs.Session.RelayNum, nil)
		// validate expected results
		require.Empty(t, psm.sessionsWithAllConsumers)
		require.Nil(t, sps)
		require.Error(t, err)
		require.True(t, ConsumerNotRegisteredYet.Is(err))
		// expect session to be missing, so we need to register it for the first time
		sps, err = psm.RegisterProviderSessionWithConsumer(ctx, consumerOneAddress, cs.Session.Parent.PairingEpoch, uint64(cs.Session.SessionId), cs.Session.RelayNum, cs.Session.Parent.MaxComputeUnits, pairedProviders, "projectIdTest", nil)
		// validate session was added
		require.NotEmpty(t, psm.sessionsWithAllConsumers)
		require.NoError(t, err)
		require.NotNil(t, sps)

		// prepare session for usage
		err = sps.PrepareSessionForUsage(ctx, cuForFirstRequest, cs.Session.LatestRelayCu, 0, 0)

		// validate session was prepared successfully
		require.NoError(t, err)
		require.Equal(t, cs.Session.LatestRelayCu, sps.LatestRelayCu)
		require.Equal(t, sps.CuSum, cs.Session.LatestRelayCu)
		require.Equal(t, sps.SessionID, uint64(cs.Session.SessionId))
		require.Equal(t, sps.PairingEpoch, cs.Session.Parent.PairingEpoch)

		err = psm.OnSessionDone(sps, cs.Session.RelayNum)
		require.Equal(t, sps.RelayNum, cs.Session.RelayNum)
		require.NoError(t, err)

		// Consumer Side:
		err = csm.OnSessionDone(cs.Session, servicedBlockNumber, cuForFirstRequest, time.Millisecond, cs.Session.CalculateExpectedLatency(2*time.Millisecond), (servicedBlockNumber - 1), 1, 1, false)
		require.NoError(t, err)
		require.Equal(t, cs.Session.CuSum, cuForFirstRequest)
		require.Equal(t, cs.Session.LatestRelayCu, latestRelayCuAfterDone)
		require.Equal(t, cs.Session.RelayNum, relayNumberAfterFirstCall)
		require.Equal(t, cs.Session.LatestBlock, servicedBlockNumber)
	}
	return csm, psm, ctx
}
