package lavasession

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestHappyFlowE2E(t *testing.T) {
	s := createGRPCServer(t) // create a grpcServer so we can connect to its endpoint and validate everything works.
	defer s.Stop()           // stop the server when finished.
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
	cs, epoch, _, _, err := csm.GetSession(ctx, cuForFirstRequest, nil) // get a session
	require.Nil(t, err)
	require.NotNil(t, cs)
	require.Equal(t, epoch, csm.currentEpoch)
	require.Equal(t, cs.LatestRelayCu, uint64(cuForFirstRequest))

	// Provider Side:

	sps, err := psm.GetSession(ctx, consumerOneAddress, uint64(cs.Client.PairingEpoch), uint64(cs.SessionId), cs.RelayNum)
	// validate expected results
	require.Empty(t, psm.sessionsWithAllConsumers)
	require.Nil(t, sps)
	require.Error(t, err)
	require.True(t, ConsumerNotRegisteredYet.Is(err))
	// expect session to be missing, so we need to register it for the first time
	sps, err = psm.RegisterProviderSessionWithConsumer(ctx, consumerOneAddress, uint64(cs.Client.PairingEpoch), uint64(cs.SessionId), cs.RelayNum, cs.Client.MaxComputeUnits, selfProviderIndex, pairedProviders)
	// validate session was added
	require.NotEmpty(t, psm.sessionsWithAllConsumers)
	require.Nil(t, err)
	require.NotNil(t, sps)

	// prepare session for usage
	err = sps.PrepareSessionForUsage(ctx, cuForFirstRequest, cs.LatestRelayCu, 0)

	// validate session was prepared successfully
	require.Nil(t, err)
	require.Equal(t, sps.userSessionsParent.atomicReadProviderIndex(), selfProviderIndex)
	require.Equal(t, cs.LatestRelayCu, sps.LatestRelayCu)
	require.Equal(t, sps.CuSum, cs.LatestRelayCu)
	require.Equal(t, sps.SessionID, uint64(cs.SessionId))
	require.Equal(t, sps.PairingEpoch, cs.Client.PairingEpoch)

	err = psm.OnSessionDone(sps, cs.RelayNum)
	require.Equal(t, sps.RelayNum, cs.RelayNum)
	require.NoError(t, err)

	// Consumer Side:
	err = csm.OnSessionDone(cs, epoch1, servicedBlockNumber, cuForFirstRequest, time.Duration(time.Millisecond), cs.CalculateExpectedLatency(2*time.Duration(time.Millisecond)), (servicedBlockNumber - 1), 1, 1)
	require.Nil(t, err)
	require.Equal(t, cs.CuSum, cuForFirstRequest)
	require.Equal(t, cs.LatestRelayCu, latestRelayCuAfterDone)
	require.Equal(t, cs.RelayNum, relayNumberAfterFirstCall)
	require.Equal(t, cs.LatestBlock, servicedBlockNumber)
}
