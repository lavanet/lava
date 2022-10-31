package lavasession

import (
	"context"
	"fmt"
	"math/rand"
	"net"
	"strconv"
	"testing"
	"time"

	"github.com/lavanet/lava/utils"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
)

const (
	parallelGoRoutines        = 40
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

func createGRPCServer(t *testing.T) *grpc.Server {
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
			Sessions:        map[int64]*SingleConsumerSession{},
			MaxComputeUnits: 200,
			ReliabilitySent: false,
			PairingEpoch:    firstEpochHeight,
		})
	}
	return cswpList
}

// Test the basic functionality of the consumerSessionManager
func TestHappyFlow(t *testing.T) {
	s := createGRPCServer(t) // create a grpcServer so we can connect to its endpoint and validate everything works.
	defer s.Stop()           // stop the server when finished.
	ctx := context.Background()
	csm := CreateConsumerSessionManager()
	pairingList := createPairingList()
	err := csm.UpdateAllProviders(ctx, firstEpochHeight, pairingList) // update the providers.
	require.Nil(t, err)
	cs, epoch, err := csm.GetSession(ctx, cuForFirstRequest, nil) // get a session
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

func successfulSession(ctx context.Context, csm *ConsumerSessionManager, t *testing.T, p int, ch chan int) {
	cs, _, err := csm.GetSession(ctx, cuForFirstRequest, nil) // get a session
	require.Nil(t, err)
	require.NotNil(t, cs)
	err = csm.OnSessionDone(cs, firstEpochHeight, servicedBlockNumber)
	require.Nil(t, err)
	ch <- p
}

func failedSession(ctx context.Context, csm *ConsumerSessionManager, t *testing.T, p int, ch chan int) {
	cs, _, err := csm.GetSession(ctx, cuForFirstRequest, nil) // get a session
	require.Nil(t, err)
	require.NotNil(t, cs)
	err = csm.OnSessionFailure(cs, fmt.Errorf("nothing special"), false)
	require.Nil(t, err)
	ch <- p
}

func TestHappyFlowMultiThreaded(t *testing.T) {
	utils.LavaFormatInfo("Parallel test:", nil)

	s := createGRPCServer(t) // create a grpcServer so we can connect to its endpoint and validate everything works.
	defer s.Stop()           // stop the server when finished.
	ctx := context.Background()
	csm := CreateConsumerSessionManager()
	pairingList := createPairingList()
	err := csm.UpdateAllProviders(ctx, firstEpochHeight, pairingList) // update the providers.
	require.Nil(t, err)
	ch1 := make(chan int)
	ch2 := make(chan int)
	for p := 0; p < parallelGoRoutines; p++ { // we have x amount of successful sessions and x amount of failed. validate compute units
		go successfulSession(ctx, csm, t, p, ch1)
		go failedSession(ctx, csm, t, p, ch2)
	}
	all_chs := make(map[int]struct{}, parallelGoRoutines*2)
	for {
		ch1val := <-ch1
		ch2val := <-ch2 + parallelGoRoutines
		if _, ok := all_chs[ch1val]; !ok {
			all_chs[ch1val] = struct{}{}
		}
		if _, ok := all_chs[ch2val]; !ok {
			all_chs[ch2val] = struct{}{}
		}
		if len(all_chs) >= parallelGoRoutines*2 {
			utils.LavaFormatInfo(fmt.Sprintf("finished routines len(all_chs): %d", len(all_chs)), nil)
			break // routines finished
		} else {
			utils.LavaFormatInfo(fmt.Sprintf("awaiting routines: ch1: %d, ch2: %d", ch1val, ch2val), nil)
		}
	}

	fmt.Printf("%v", csm.pairing)
	var totalUsedCU uint64
	for k := range csm.pairing {
		fmt.Printf("key: %v\n", k)
		fmt.Printf("Sessions: %v\n", csm.pairing[k].Sessions)
		fmt.Printf("UsedComputeUnits: %v\n", csm.pairing[k].UsedComputeUnits)
		totalUsedCU += csm.pairing[k].UsedComputeUnits

	}
	fmt.Printf("Total: %v\n", totalUsedCU)
	fmt.Printf("Expected CU: %v\n", cuForFirstRequest*parallelGoRoutines)
	require.Equal(t, cuForFirstRequest*parallelGoRoutines, totalUsedCU)
}

func TestHappyFlowMultiThreadedWithUpdateSession(t *testing.T) {
	utils.LavaFormatInfo("Parallel test:", nil)

	s := createGRPCServer(t) // create a grpcServer so we can connect to its endpoint and validate everything works.
	defer s.Stop()           // stop the server when finished.
	ctx := context.Background()
	csm := CreateConsumerSessionManager()
	pairingList := createPairingList()
	err := csm.UpdateAllProviders(ctx, firstEpochHeight, pairingList) // update the providers.
	require.Nil(t, err)
	ch1 := make(chan int)
	ch2 := make(chan int)
	for p := 0; p < parallelGoRoutines; p++ { // we have x amount of successful sessions and x amount of failed. validate compute units
		go successfulSession(ctx, csm, t, p, ch1)

		go failedSession(ctx, csm, t, p, ch2)
	}
	all_chs := make(map[int]struct{}, parallelGoRoutines*2)
	for {
		ch1val := <-ch1
		ch2val := <-ch2 + parallelGoRoutines
		if len(all_chs) == parallelGoRoutines { // at half of the go routines launch the swap.
			go func() {
				utils.LavaFormatInfo(fmt.Sprintf("#### UPDATING PROVIDERS ####"), nil)
				err := csm.UpdateAllProviders(ctx, secondEpochHeight, pairingList[0:(numberOfProviders/2)]) // update the providers. with half of them
				require.Nil(t, err)
			}()
		}

		if _, ok := all_chs[ch1val]; !ok {
			all_chs[ch1val] = struct{}{}
		}
		if _, ok := all_chs[ch2val]; !ok {
			all_chs[ch2val] = struct{}{}
		}
		if len(all_chs) >= parallelGoRoutines*2 {
			utils.LavaFormatInfo(fmt.Sprintf("finished routines len(all_chs): %d", len(all_chs)), nil)
			break // routines finished
		} else {
			utils.LavaFormatInfo(fmt.Sprintf("awaiting routines: ch1: %d, ch2: %d", ch1val, ch2val), nil)
		}
	}

	fmt.Printf("%v", csm.pairing)
	var totalUsedCU uint64
	for _, k := range pairingList {
		fmt.Printf("Sessions: %v\n", k.Sessions)
		fmt.Printf("UsedComputeUnits: %v\n", k.UsedComputeUnits)
		totalUsedCU += k.UsedComputeUnits

	}
	fmt.Printf("Total: %v\n", totalUsedCU)
	fmt.Printf("Expected CU: %v\n", cuForFirstRequest*parallelGoRoutines)

	require.Equal(t, cuForFirstRequest*parallelGoRoutines, totalUsedCU)

}

// Test the basic functionality of the consumerSessionManager
func TestSessionFailureAndGetReportedProviders(t *testing.T) {
	s := createGRPCServer(t) // create a grpcServer so we can connect to its endpoint and validate everything works.
	defer s.Stop()           // stop the server when finished.
	ctx := context.Background()
	csm := CreateConsumerSessionManager()
	pairingList := createPairingList()
	err := csm.UpdateAllProviders(ctx, firstEpochHeight, pairingList) // update the providers.
	require.Nil(t, err)
	cs, epoch, err := csm.GetSession(ctx, cuForFirstRequest, nil) // get a session
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
	require.NotContains(t, csm.validAddresses, cs.client.Acc)     // address isn't in valid addresses list

	reported := csm.GetReportedProviders(firstEpochHeight)
	require.NotEmpty(t, reported)
	for _, providerReported := range reported {
		require.Contains(t, csm.addedToPurgeAndReport, providerReported)
		require.Contains(t, csm.providerBlockList, providerReported)
	}
}

// Test the basic functionality of the consumerSessionManager
func TestSessionFailureEpochMisMatch(t *testing.T) {
	s := createGRPCServer(t) // create a grpcServer so we can connect to its endpoint and validate everything works.
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

func TestAllProvidersEndpointsDisabled(t *testing.T) {
	ctx := context.Background()
	csm := CreateConsumerSessionManager()
	pairingList := createPairingList()
	err := csm.UpdateAllProviders(ctx, firstEpochHeight, pairingList) // update the providers.
	require.Nil(t, err)
	cs, _, err := csm.GetSession(ctx, cuForFirstRequest, nil) // get a session
	require.Nil(t, cs)
	require.Error(t, err)
}

func TestUpdateAllProviders(t *testing.T) {
	ctx := context.Background()
	csm := CreateConsumerSessionManager()
	pairingList := createPairingList()
	err := csm.UpdateAllProviders(ctx, firstEpochHeight, pairingList)
	require.Nil(t, err)
	require.Equal(t, len(csm.validAddresses), numberOfProviders) // checking there are 2 valid addresses
	require.Equal(t, len(csm.pairingAddresses), numberOfProviders)
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
	require.Equal(t, len(csm.validAddresses), numberOfProviders) // checking there are 2 valid addresses
	require.Equal(t, len(csm.pairingAddresses), numberOfProviders)
	require.Equal(t, csm.currentEpoch, uint64(firstEpochHeight)) // verify epoch
	for p := 0; p < numberOfProviders; p++ {
		require.Contains(t, csm.pairing, "provider"+strconv.Itoa(p)) // verify pairings
	}
}

func TestGetSession(t *testing.T) {
	s := createGRPCServer(t) // create a grpcServer so we can connect to its endpoint and validate everything works.
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
