package lavasession

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestHappyFlowE2E(t *testing.T) {
	ctx := context.Background()

	// Consumer Side:
	csm := CreateConsumerSessionManager() // consumer session manager

	// Provider Side:
	psm := initProviderSessionManager() // provider session manager

	// Consumer Side:
	cswpList := make(map[uint64]*ConsumerSessionsWithProvider, 1)
	pairingEndpoints := make([]*Endpoint, 1)
	// we need a grpc server to connect to. so we use the public rpc endpoint for now.
	pairingEndpoints[0] = &Endpoint{NetworkAddress: grpcListener, Enabled: true, Client: nil, ConnectionRefusals: 0}
	cswpList[0] = &ConsumerSessionsWithProvider{
		PublicLavaAddress: "provider",
		Endpoints:         pairingEndpoints,
		Sessions:          map[int64]*SingleConsumerSession{},
		MaxComputeUnits:   200,
		ReliabilitySent:   false,
		PairingEpoch:      epoch1,
	}
	err := csm.UpdateAllProviders(epoch1, cswpList) // update the providers.
	require.NoError(t, err)
	// get single consumer session
	css, err := csm.GetSessions(ctx, cuForFirstRequest, nil, servicedBlockNumber) // get a session
	require.Nil(t, err)

	for _, cs := range css {
		require.NotNil(t, cs)
		require.Equal(t, cs.Epoch, csm.currentEpoch)
		require.Equal(t, cs.Session.LatestRelayCu, cuForFirstRequest)

		// Provider Side:

		sps, err := psm.GetSession(ctx, consumerOneAddress, cs.Session.Client.PairingEpoch, uint64(cs.Session.SessionId), cs.Session.RelayNum, nil)
		// validate expected results
		require.Empty(t, psm.sessionsWithAllConsumers)
		require.Nil(t, sps)
		require.Error(t, err)
		require.True(t, ConsumerNotRegisteredYet.Is(err))
		// expect session to be missing, so we need to register it for the first time
		sps, err = psm.RegisterProviderSessionWithConsumer(ctx, consumerOneAddress, cs.Session.Client.PairingEpoch, uint64(cs.Session.SessionId), cs.Session.RelayNum, cs.Session.Client.MaxComputeUnits, pairedProviders, nil)
		// validate session was added
		require.NotEmpty(t, psm.sessionsWithAllConsumers)
		require.Nil(t, err)
		require.NotNil(t, sps)

		// prepare session for usage
		err = sps.PrepareSessionForUsage(ctx, cuForFirstRequest, cs.Session.LatestRelayCu, 0)

		// validate session was prepared successfully
		require.Nil(t, err)
		require.Equal(t, cs.Session.LatestRelayCu, sps.LatestRelayCu)
		require.Equal(t, sps.CuSum, cs.Session.LatestRelayCu)
		require.Equal(t, sps.SessionID, uint64(cs.Session.SessionId))
		require.Equal(t, sps.PairingEpoch, cs.Session.Client.PairingEpoch)

		err = psm.OnSessionDone(sps, cs.Session.RelayNum)
		require.Equal(t, sps.RelayNum, cs.Session.RelayNum)
		require.NoError(t, err)

		// Consumer Side:
		err = csm.OnSessionDone(cs.Session, epoch1, servicedBlockNumber, cuForFirstRequest, time.Millisecond, cs.Session.CalculateExpectedLatency(2*time.Millisecond), (servicedBlockNumber - 1), 1, 1, false)
		require.Nil(t, err)
		require.Equal(t, cs.Session.CuSum, cuForFirstRequest)
		require.Equal(t, cs.Session.LatestRelayCu, latestRelayCuAfterDone)
		require.Equal(t, cs.Session.RelayNum, relayNumberAfterFirstCall)
		require.Equal(t, cs.Session.LatestBlock, servicedBlockNumber)
	}
}
