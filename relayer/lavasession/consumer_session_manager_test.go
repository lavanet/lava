package lavasession

import (
	"context"
	"math/rand"
	"net"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
)

const (
	numberOfProviders         = 10
	firstEpochHeight          = 20
	secondEpochHeight         = 40
	cuForFirstRequest         = uint64(10)
	grpcListener              = "localhost:48353"
	servicedBlockNumber       = int64(30)
	relayNumberAfterFirstCall = uint64(1)
	relayNumberAfterFirstFail = uint64(0)
	latestRelayCuAfterDone    = uint64(0)
	cuSumOnFailure            = uint64(0)
)

func CreateConsumerSessionManager() *ConsumerSessionManager {
	rand.Seed(time.Now().UnixNano())
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

// Test the basic functionality of the consumerSessionManager
func TestHappyFlow(t *testing.T) {
	s := creategRPCServer(t) // create a grpcServer so we can connect to its endpoint and validate everything works.
	defer s.Stop()           // stop the server when finished.
	ctx := context.Background()
	csm := CreateConsumerSessionManager()
	pairingList := createPairingList()
	err := csm.UpdateAllProviders(ctx, firstEpochHeight, pairingList) // update the providers.
	require.Nil(t, err)
	cs, epoch, err := csm.GetSession(ctx, cuForFirstRequest, nil) // get a sesssion
	require.Nil(t, err)
	require.NotNil(t, cs)
	require.Equal(t, epoch, csm.currentEpoch)
	require.Equal(t, cs.latestRelayCu, uint64(cuForFirstRequest))
	err = csm.OnSessionDone(cs, firstEpochHeight, servicedBlockNumber)
	require.Nil(t, err)
	require.Equal(t, cs.cuSum, cuForFirstRequest)
	require.Equal(t, cs.latestRelayCu, latestRelayCuAfterDone)
	require.Equal(t, cs.relayNum, relayNumberAfterFirstCall)
	require.Equal(t, cs.latestBlock, servicedBlockNumber)
}

// Test the basic functionality of the consumerSessionManager
func TestSessionFaliureAndGetReportedProviders(t *testing.T) {
	s := creategRPCServer(t) // create a grpcServer so we can connect to its endpoint and validate everything works.
	defer s.Stop()           // stop the server when finished.
	ctx := context.Background()
	csm := CreateConsumerSessionManager()
	pairingList := createPairingList()
	err := csm.UpdateAllProviders(ctx, firstEpochHeight, pairingList) // update the providers.
	require.Nil(t, err)
	cs, epoch, err := csm.GetSession(ctx, cuForFirstRequest, nil) // get a sesssion
	require.Nil(t, err)
	require.NotNil(t, cs)
	require.Equal(t, epoch, csm.currentEpoch)
	require.Equal(t, cs.latestRelayCu, uint64(cuForFirstRequest))
	err = csm.OnSessionFailure(cs, ReportAndBlockProviderError, true)
	require.Nil(t, err)
	require.Equal(t, cs.client.UsedComputeUnits, cuSumOnFailure)
	require.Equal(t, cs.cuSum, cuSumOnFailure)
	require.Equal(t, cs.latestRelayCu, latestRelayCuAfterDone)
	require.Equal(t, cs.relayNum, relayNumberAfterFirstFail)

	// verify provider is blocked and reported
	require.Contains(t, csm.addedToPurgeAndReport, cs.client.Acc) // address is reported
	require.Contains(t, csm.providerBlockList, cs.client.Acc)     // address is blocked
	require.NotContains(t, csm.validAddressess, cs.client.Acc)    // address isnt in valid addresses list

	reported := csm.GetReportedProviders(firstEpochHeight)
	require.NotEmpty(t, reported)
	for _, providerReported := range reported {
		require.Contains(t, csm.addedToPurgeAndReport, providerReported)
		require.Contains(t, csm.providerBlockList, providerReported)
	}
}

// Test the basic functionality of the consumerSessionManager
func TestSessionFaliureEpochMisMatch(t *testing.T) {
	s := creategRPCServer(t) // create a grpcServer so we can connect to its endpoint and validate everything works.
	defer s.Stop()           // stop the server when finished.
	ctx := context.Background()
	csm := CreateConsumerSessionManager()
	pairingList := createPairingList()
	err := csm.UpdateAllProviders(ctx, firstEpochHeight, pairingList) // update the providers.
	require.Nil(t, err)
	cs, epoch, err := csm.GetSession(ctx, cuForFirstRequest, nil) // get a sesssion
	require.Nil(t, err)
	require.NotNil(t, cs)
	require.Equal(t, epoch, csm.currentEpoch)
	require.Equal(t, cs.latestRelayCu, uint64(cuForFirstRequest))

	err = csm.UpdateAllProviders(ctx, secondEpochHeight, pairingList) // update the providers again.
	require.Nil(t, err)
	err = csm.OnSessionFailure(cs, ReportAndBlockProviderError, true)
	require.Nil(t, err)
}

func TestAllProvidersEndpoijntsDisabled(t *testing.T) {
	ctx := context.Background()
	csm := CreateConsumerSessionManager()
	pairingList := createPairingList()
	err := csm.UpdateAllProviders(ctx, firstEpochHeight, pairingList) // update the providers.
	require.Nil(t, err)
	cs, _, err := csm.GetSession(ctx, cuForFirstRequest, nil) // get a sesssion
	require.Nil(t, cs)
	require.Error(t, err)
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
