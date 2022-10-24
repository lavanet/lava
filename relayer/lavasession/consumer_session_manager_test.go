package lavasession

import (
	"context"
	"net"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
)

const (
	numberOfProviders = 10
	firstEpochHeight  = 20
	cuForFirstRequest = 10
	grpcListener      = "localhost:48353"
)

func CreateConsumerSessionManager() *ConsumerSessionManager {
	return &ConsumerSessionManager{}
}

func creategRPCServer(t *testing.T) *grpc.Server {
	lis, err := net.Listen("tcp", grpcListener)
	require.Nil(t, err)
	s := grpc.NewServer()
	go s.Serve(lis) // serve in a different thread
	return s
}

func createPairingList() []*ConsumerSessionsWithProvider {
	cswpList := make([]*ConsumerSessionsWithProvider, 0)
	pairingEndpoints := make([]*Endpoint, 1)
	// we need a grpc server to connect to. so we use the public rpc endpoint for now.
	pairingEndpoints[0] = &Endpoint{Addr: grpcListener, Enabled: true, Client: nil, ConnectionRefusals: 0}
	for p := 0; p < numberOfProviders; p++ {
		cswpList = append(cswpList, &ConsumerSessionsWithProvider{
			Acc:             "provider" + strconv.Itoa(p),
			Endpoints:       pairingEndpoints,
			Sessions:        map[int64]*ConsumerSession{},
			MaxComputeUnits: 200,
			ReliabilitySent: false,
			PairingEpoch:    firstEpochHeight,
		})
	}
	return cswpList
}

func TestUpdateAllProviders(t *testing.T) {
	ctx := context.Background()
	csm := CreateConsumerSessionManager()
	pairingList := createPairingList()
	err := csm.UpdateAllProviders(ctx, firstEpochHeight, pairingList)
	require.Nil(t, err)
	require.Equal(t, len(csm.validAddressess), numberOfProviders) // checking there are 2 valid addressess
	require.Equal(t, len(csm.pairingAdressess), numberOfProviders)
	require.Equal(t, csm.currentEpoch, uint64(firstEpochHeight)) // verify epoch
	for p := 0; p < numberOfProviders; p++ {
		require.Contains(t, csm.pairing, "provider"+strconv.Itoa(p)) // verify pairings
	}
}

func TestUpdateAllProvidersWithSameEpoch(t *testing.T) {
	ctx := context.Background()
	csm := CreateConsumerSessionManager()
	pairingList := createPairingList()
	err := csm.UpdateAllProviders(ctx, firstEpochHeight, pairingList)
	require.Nil(t, err)
	err = csm.UpdateAllProviders(ctx, firstEpochHeight, pairingList)
	require.Error(t, err)
	// perform same validations as normal usage
	require.Equal(t, len(csm.validAddressess), numberOfProviders) // checking there are 2 valid addressess
	require.Equal(t, len(csm.pairingAdressess), numberOfProviders)
	require.Equal(t, csm.currentEpoch, uint64(firstEpochHeight)) // verify epoch
	for p := 0; p < numberOfProviders; p++ {
		require.Contains(t, csm.pairing, "provider"+strconv.Itoa(p)) // verify pairings
	}
}

func TestGetSession(t *testing.T) {
	s := creategRPCServer(t) // create a grpcServer so we can connect to its endpoint and validate everything works.
	defer s.Stop()           // stop the server when finished.
	ctx := context.Background()
	csm := CreateConsumerSessionManager()
	pairingList := createPairingList()
	err := csm.UpdateAllProviders(ctx, firstEpochHeight, pairingList)
	require.Nil(t, err)
	cs, epoch, err := csm.GetSession(ctx, cuForFirstRequest, nil)
	require.Nil(t, err)
	require.NotNil(t, cs)
	require.Equal(t, epoch, csm.currentEpoch)
	require.Equal(t, cs.latestRelayCu, uint64(cuForFirstRequest))
}
