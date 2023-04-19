package lavasession

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"net"
	"strconv"
	"testing"
	"time"

	"github.com/lavanet/lava/protocol/provideroptimizer"
	"github.com/lavanet/lava/utils"
	pairingtypes "github.com/lavanet/lava/x/pairing/types"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

const (
	parallelGoRoutines                 = 40
	numberOfProviders                  = 10
	numberOfResetsToTest               = 10
	numberOfAllowedSessionsPerConsumer = 10
	firstEpochHeight                   = 20
	secondEpochHeight                  = 40
	cuForFirstRequest                  = uint64(10)
	grpcListener                       = "localhost:48353"
	servicedBlockNumber                = int64(30)
	relayNumberAfterFirstCall          = uint64(1)
	relayNumberAfterFirstFail          = uint64(1)
	latestRelayCuAfterDone             = uint64(0)
	cuSumOnFailure                     = uint64(0)
)

func CreateConsumerSessionManager() *ConsumerSessionManager {
	rand.Seed(time.Now().UnixNano()) //nolint:all
	return NewConsumerSessionManager(&RPCEndpoint{"stub", "stub", "stub", 0}, provideroptimizer.NewProviderOptimizer(provideroptimizer.STRATEGY_QOS))
}

func createGRPCServer(t *testing.T) *grpc.Server {
	lis, err := net.Listen("tcp", grpcListener)
	require.Nil(t, err)
	s := grpc.NewServer()
	go s.Serve(lis) // serve in a different thread
	return s
}

func createPairingList(providerPrefixAddress string) map[uint64]*ConsumerSessionsWithProvider {
	cswpList := make(map[uint64]*ConsumerSessionsWithProvider, 0)
	pairingEndpoints := make([]*Endpoint, 1)
	// we need a grpc server to connect to. so we use the public rpc endpoint for now.
	pairingEndpoints[0] = &Endpoint{NetworkAddress: grpcListener, Enabled: true, Client: nil, ConnectionRefusals: 0}
	for p := 0; p < numberOfProviders; p++ {
		cswpList[uint64(p)] = &ConsumerSessionsWithProvider{
			PublicLavaAddress: "provider" + providerPrefixAddress + strconv.Itoa(p),
			Endpoints:         pairingEndpoints,
			Sessions:          map[int64]*SingleConsumerSession{},
			MaxComputeUnits:   200,
			ReliabilitySent:   false,
			PairingEpoch:      firstEpochHeight,
		}
	}
	return cswpList
}

// Test the basic functionality of the consumerSessionManager
func TestHappyFlow(t *testing.T) {
	s := createGRPCServer(t) // create a grpcServer so we can connect to its endpoint and validate everything works.
	defer s.Stop()           // stop the server when finished.
	ctx := context.Background()
	csm := CreateConsumerSessionManager()
	pairingList := createPairingList("")
	err := csm.UpdateAllProviders(firstEpochHeight, pairingList) // update the providers.
	require.Nil(t, err)
	cs, epoch, _, _, err := csm.GetSession(ctx, cuForFirstRequest, nil) // get a session
	require.Nil(t, err)
	require.NotNil(t, cs)
	require.Equal(t, epoch, csm.currentEpoch)
	require.Equal(t, cs.LatestRelayCu, cuForFirstRequest)
	err = csm.OnSessionDone(cs, firstEpochHeight, servicedBlockNumber, cuForFirstRequest, time.Millisecond, cs.CalculateExpectedLatency(2*time.Millisecond), (servicedBlockNumber - 1), numberOfProviders, numberOfProviders)
	require.Nil(t, err)
	require.Equal(t, cs.CuSum, cuForFirstRequest)
	require.Equal(t, cs.LatestRelayCu, latestRelayCuAfterDone)
	require.Equal(t, cs.RelayNum, relayNumberAfterFirstCall)
	require.Equal(t, cs.LatestBlock, servicedBlockNumber)
}

func TestPairingReset(t *testing.T) {
	s := createGRPCServer(t) // create a grpcServer so we can connect to its endpoint and validate everything works.
	defer s.Stop()           // stop the server when finished.
	ctx := context.Background()
	csm := CreateConsumerSessionManager()
	pairingList := createPairingList("")
	err := csm.UpdateAllProviders(firstEpochHeight, pairingList) // update the providers.
	require.Nil(t, err)
	csm.validAddresses = []string{}                                     // set valid addresses to zero
	cs, epoch, _, _, err := csm.GetSession(ctx, cuForFirstRequest, nil) // get a session
	require.Nil(t, err)
	require.Equal(t, len(csm.validAddresses), len(csm.pairingAddresses))
	require.NotNil(t, cs)
	require.Equal(t, epoch, csm.currentEpoch)
	require.Equal(t, cs.LatestRelayCu, cuForFirstRequest)
	err = csm.OnSessionDone(cs, firstEpochHeight, servicedBlockNumber, cuForFirstRequest, time.Millisecond, cs.CalculateExpectedLatency(2*time.Millisecond), (servicedBlockNumber - 1), numberOfProviders, numberOfProviders)
	require.Nil(t, err)
	require.Equal(t, cs.CuSum, cuForFirstRequest)
	require.Equal(t, cs.LatestRelayCu, latestRelayCuAfterDone)
	require.Equal(t, cs.RelayNum, relayNumberAfterFirstCall)
	require.Equal(t, cs.LatestBlock, servicedBlockNumber)
	require.Equal(t, csm.numberOfResets, uint64(0x1)) // verify we had one reset only
}

func TestPairingResetWithFailures(t *testing.T) {
	s := createGRPCServer(t) // create a grpcServer so we can connect to its endpoint and validate everything works.
	defer s.Stop()           // stop the server when finished.
	ctx := context.Background()
	csm := CreateConsumerSessionManager()
	pairingList := createPairingList("")
	err := csm.UpdateAllProviders(firstEpochHeight, pairingList) // update the providers.
	require.Nil(t, err)
	for {
		utils.LavaFormatDebug(fmt.Sprintf("%v", len(csm.validAddresses)))
		if len(csm.validAddresses) == 0 { // wait for all pairings to be blocked.
			break
		}
		cs, _, _, _, err := csm.GetSession(ctx, cuForFirstRequest, nil) // get a session
		require.Nil(t, err)                                             // fail test.
		err = csm.OnSessionFailure(cs, nil)
		require.Nil(t, err) // fail test.
	}
	require.Equal(t, len(csm.validAddresses), 0)
	cs, epoch, _, _, err := csm.GetSession(ctx, cuForFirstRequest, nil) // get a session
	require.Nil(t, err)
	require.Equal(t, len(csm.validAddresses), len(csm.pairingAddresses))
	require.NotNil(t, cs)
	require.Equal(t, epoch, csm.currentEpoch)
	require.Equal(t, cs.LatestRelayCu, cuForFirstRequest)
	require.Equal(t, csm.numberOfResets, uint64(0x1)) // verify we had one reset only
}

func TestPairingResetWithMultipleFailures(t *testing.T) {
	s := createGRPCServer(t) // create a grpcServer so we can connect to its endpoint and validate everything works.
	defer s.Stop()           // stop the server when finished.
	ctx := context.Background()
	csm := CreateConsumerSessionManager()
	pairingList := createPairingList("")
	err := csm.UpdateAllProviders(firstEpochHeight, pairingList) // update the providers.
	require.Nil(t, err)
	for numberOfResets := 0; numberOfResets < numberOfResetsToTest; numberOfResets++ {
		for {
			utils.LavaFormatDebug(fmt.Sprintf("%v", len(csm.validAddresses)))
			if len(csm.validAddresses) == 0 { // wait for all pairings to be blocked.
				break
			}
			cs, _, _, _, err := csm.GetSession(ctx, cuForFirstRequest, nil) // get a session
			if len(csm.validAddresses) == 0 {                               // wait for all pairings to be blocked.
				break
			}
			require.Nil(t, err)
			err = csm.OnSessionFailure(cs, nil)
			require.Nil(t, err)
		}
		require.Equal(t, len(csm.validAddresses), 0)
		cs, epoch, _, _, err := csm.GetSession(ctx, cuForFirstRequest, nil) // get a session
		require.Nil(t, err)
		require.Equal(t, len(csm.validAddresses), len(csm.pairingAddresses))
		require.NotNil(t, cs)
		require.Equal(t, epoch, csm.currentEpoch)
		require.Equal(t, cs.LatestRelayCu, cuForFirstRequest)
		require.Equal(t, csm.numberOfResets, uint64(numberOfResets+1)) // verify we had one reset only
	}

	cs, epoch, _, _, err := csm.GetSession(ctx, cuForFirstRequest, nil) // get a session
	require.Nil(t, err)
	require.NotNil(t, cs)
	require.Equal(t, epoch, csm.currentEpoch)
	require.Equal(t, cs.LatestRelayCu, cuForFirstRequest)
	err = csm.OnSessionDone(cs, firstEpochHeight, servicedBlockNumber, cuForFirstRequest, time.Millisecond, cs.CalculateExpectedLatency(2*time.Millisecond), (servicedBlockNumber - 1), numberOfProviders, numberOfProviders)
	require.Nil(t, err)
	require.Equal(t, cs.CuSum, cuForFirstRequest)
	require.Equal(t, cs.LatestRelayCu, latestRelayCuAfterDone)
	require.GreaterOrEqual(t, relayNumberAfterFirstFail, cs.RelayNum)
	require.Equal(t, cs.LatestBlock, servicedBlockNumber)
}

// Test the basic functionality of the consumerSessionManager
func TestSuccessAndFailureOfSessionWithUpdatePairingsInTheMiddle(t *testing.T) {
	s := createGRPCServer(t) // create a grpcServer so we can connect to its endpoint and validate everything works.
	defer s.Stop()           // stop the server when finished.
	ctx := context.Background()
	csm := CreateConsumerSessionManager()
	pairingList := createPairingList("")
	err := csm.UpdateAllProviders(firstEpochHeight, pairingList) // update the providers.
	require.Nil(t, err)

	type session struct {
		cs    *SingleConsumerSession
		epoch uint64
	}
	type SessTestData struct {
		relayNum uint64
		cuSum    uint64
	}
	sessionList := make([]session, numberOfAllowedSessionsPerConsumer)
	sessionListData := make([]SessTestData, numberOfAllowedSessionsPerConsumer)
	for i := 0; i < numberOfAllowedSessionsPerConsumer; i++ {
		cs, epoch, _, _, err := csm.GetSession(ctx, cuForFirstRequest, nil) // get a session
		require.Nil(t, err)
		require.NotNil(t, cs)
		require.Equal(t, epoch, csm.currentEpoch)
		require.Equal(t, cs.LatestRelayCu, cuForFirstRequest)
		sessionList[i] = session{cs: cs, epoch: epoch}
	}

	for j := 0; j < numberOfAllowedSessionsPerConsumer/2; j++ {
		cs := sessionList[j].cs
		require.NotNil(t, cs)
		epoch := sessionList[j].epoch
		require.Equal(t, epoch, csm.currentEpoch)

		if rand.Intn(2) > 0 {
			err = csm.OnSessionDone(cs, epoch, servicedBlockNumber, cuForFirstRequest, time.Millisecond, cs.CalculateExpectedLatency(2*time.Millisecond), (servicedBlockNumber - 1), numberOfProviders, numberOfProviders)
			require.Nil(t, err)
			require.Equal(t, cs.CuSum, cuForFirstRequest)
			require.Equal(t, cs.LatestRelayCu, latestRelayCuAfterDone)
			require.Equal(t, cs.RelayNum, relayNumberAfterFirstCall)
			require.Equal(t, cs.LatestBlock, servicedBlockNumber)
			sessionListData[j] = SessTestData{cuSum: cuForFirstRequest, relayNum: 1}
		} else {
			err = csm.OnSessionFailure(cs, nil)
			require.Nil(t, err)
			require.Equal(t, cs.CuSum, uint64(0))
			require.Equal(t, cs.RelayNum, relayNumberAfterFirstFail)
			require.Equal(t, cs.LatestRelayCu, latestRelayCuAfterDone)
			sessionListData[j] = SessTestData{cuSum: 0, relayNum: 1}
		}
	}

	for i := 0; i < numberOfAllowedSessionsPerConsumer; i++ {
		cs, epoch, _, _, err := csm.GetSession(ctx, cuForFirstRequest, nil) // get a session
		require.Nil(t, err)
		require.NotNil(t, cs)
		require.Equal(t, epoch, csm.currentEpoch)
		require.Equal(t, cs.LatestRelayCu, cuForFirstRequest)
	}

	err = csm.UpdateAllProviders(secondEpochHeight, createPairingList("test2")) // update the providers. with half of them
	require.Nil(t, err)

	for j := numberOfAllowedSessionsPerConsumer / 2; j < numberOfAllowedSessionsPerConsumer; j++ {
		cs := sessionList[j].cs
		epoch := sessionList[j].epoch
		if rand.Intn(2) > 0 {
			err = csm.OnSessionDone(cs, epoch, servicedBlockNumber, cuForFirstRequest, time.Millisecond, cs.CalculateExpectedLatency(2*time.Millisecond), (servicedBlockNumber - 1), numberOfProviders, numberOfProviders)
			require.Nil(t, err)
			require.Equal(t, sessionListData[j].cuSum+cuForFirstRequest, cs.CuSum)
			require.Equal(t, cs.LatestRelayCu, latestRelayCuAfterDone)
			require.Equal(t, cs.RelayNum, sessionListData[j].relayNum+1)
			require.Equal(t, cs.LatestBlock, servicedBlockNumber)
		} else {
			err = csm.OnSessionFailure(cs, nil)
			require.Nil(t, err)
			require.Equal(t, sessionListData[j].cuSum, cs.CuSum)
			require.Equal(t, cs.RelayNum, sessionListData[j].relayNum+1)
			require.Equal(t, cs.LatestRelayCu, latestRelayCuAfterDone)
		}
	}
}

func successfulSession(ctx context.Context, csm *ConsumerSessionManager, t *testing.T, p int, ch chan int) {
	cs, _, _, _, err := csm.GetSession(ctx, cuForFirstRequest, nil) // get a session
	require.Nil(t, err)
	require.NotNil(t, cs)
	time.Sleep(time.Duration((rand.Intn(500) + 1)) * time.Millisecond)
	err = csm.OnSessionDone(cs, firstEpochHeight, servicedBlockNumber, cuForFirstRequest, time.Millisecond, cs.CalculateExpectedLatency(2*time.Millisecond), (servicedBlockNumber - 1), numberOfProviders, numberOfProviders)
	require.Nil(t, err)
	ch <- p
}

func failedSession(ctx context.Context, csm *ConsumerSessionManager, t *testing.T, p int, ch chan int) {
	cs, _, _, _, err := csm.GetSession(ctx, cuForFirstRequest, nil) // get a session
	require.Nil(t, err)
	require.NotNil(t, cs)
	time.Sleep(time.Duration((rand.Intn(500) + 1)) * time.Millisecond)
	err = csm.OnSessionFailure(cs, fmt.Errorf("nothing special"))
	require.Nil(t, err)
	ch <- p
}

func TestHappyFlowMultiThreaded(t *testing.T) {
	utils.LavaFormatInfo("Parallel test:")

	s := createGRPCServer(t) // create a grpcServer so we can connect to its endpoint and validate everything works.
	defer s.Stop()           // stop the server when finished.
	ctx := context.Background()
	csm := CreateConsumerSessionManager()
	pairingList := createPairingList("")
	err := csm.UpdateAllProviders(firstEpochHeight, pairingList) // update the providers.
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
			utils.LavaFormatInfo(fmt.Sprintf("finished routines len(all_chs): %d", len(all_chs)))
			break // routines finished
		} else {
			utils.LavaFormatInfo(fmt.Sprintf("awaiting routines: ch1: %d, ch2: %d", ch1val, ch2val))
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
	utils.LavaFormatInfo("Parallel test:")

	s := createGRPCServer(t) // create a grpcServer so we can connect to its endpoint and validate everything works.
	defer s.Stop()           // stop the server when finished.
	ctx := context.Background()
	csm := CreateConsumerSessionManager()
	pairingList := createPairingList("")
	err := csm.UpdateAllProviders(firstEpochHeight, pairingList) // update the providers.
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
				utils.LavaFormatInfo("#### UPDATING PROVIDERS ####")
				err := csm.UpdateAllProviders(secondEpochHeight, createPairingList("test2")) // update the providers. with half of them
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
			utils.LavaFormatInfo(fmt.Sprintf("finished routines len(all_chs): %d", len(all_chs)))
			break // routines finished
		} else {
			utils.LavaFormatInfo(fmt.Sprintf("awaiting routines: ch1: %d, ch2: %d", ch1val, ch2val))
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
	pairingList := createPairingList("")
	err := csm.UpdateAllProviders(firstEpochHeight, pairingList) // update the providers.
	require.Nil(t, err)
	cs, epoch, _, _, err := csm.GetSession(ctx, cuForFirstRequest, nil) // get a session
	require.Nil(t, err)
	require.NotNil(t, cs)
	require.Equal(t, epoch, csm.currentEpoch)
	require.Equal(t, cs.LatestRelayCu, cuForFirstRequest)
	err = csm.OnSessionFailure(cs, ReportAndBlockProviderError)
	require.Nil(t, err)
	require.Equal(t, cs.Client.UsedComputeUnits, cuSumOnFailure)
	require.Equal(t, cs.CuSum, cuSumOnFailure)
	require.Equal(t, cs.LatestRelayCu, latestRelayCuAfterDone)
	require.Equal(t, cs.RelayNum, relayNumberAfterFirstFail)

	// verify provider is blocked and reported
	require.Contains(t, csm.addedToPurgeAndReport, cs.Client.PublicLavaAddress) // address is reported
	require.NotContains(t, csm.validAddresses, cs.Client.PublicLavaAddress)     // address isn't in valid addresses list

	reported, err := csm.GetReportedProviders(firstEpochHeight)
	require.Nil(t, err)
	require.NotEmpty(t, reported)
	reportedSlice := make([]string, 0, len(reported))
	err = json.Unmarshal(reported, &reportedSlice)
	require.Nil(t, err)
	for _, providerReported := range reportedSlice {
		require.Contains(t, csm.addedToPurgeAndReport, providerReported)
	}
}

// Test the basic functionality of the consumerSessionManager
func TestSessionFailureEpochMisMatch(t *testing.T) {
	s := createGRPCServer(t) // create a grpcServer so we can connect to its endpoint and validate everything works.
	defer s.Stop()           // stop the server when finished.
	ctx := context.Background()
	csm := CreateConsumerSessionManager()
	pairingList := createPairingList("")
	err := csm.UpdateAllProviders(firstEpochHeight, pairingList) // update the providers.
	require.Nil(t, err)
	cs, epoch, _, _, err := csm.GetSession(ctx, cuForFirstRequest, nil) // get a sesssion
	require.Nil(t, err)
	require.NotNil(t, cs)
	require.Equal(t, epoch, csm.currentEpoch)
	require.Equal(t, cs.LatestRelayCu, cuForFirstRequest)

	err = csm.UpdateAllProviders(secondEpochHeight, pairingList) // update the providers again.
	require.Nil(t, err)
	err = csm.OnSessionFailure(cs, ReportAndBlockProviderError)
	require.Nil(t, err)
}

func TestAllProvidersEndpointsDisabled(t *testing.T) {
	ctx := context.Background()
	csm := CreateConsumerSessionManager()
	pairingList := createPairingList("")
	err := csm.UpdateAllProviders(firstEpochHeight, pairingList) // update the providers.
	require.Nil(t, err)
	cs, _, _, _, err := csm.GetSession(ctx, cuForFirstRequest, nil) // get a session
	require.Nil(t, cs)
	require.Error(t, err)
}

func TestUpdateAllProviders(t *testing.T) {
	csm := CreateConsumerSessionManager()
	pairingList := createPairingList("")
	err := csm.UpdateAllProviders(firstEpochHeight, pairingList)
	require.Nil(t, err)
	require.Equal(t, len(csm.validAddresses), numberOfProviders) // checking there are 2 valid addresses
	require.Equal(t, len(csm.pairingAddresses), numberOfProviders)
	require.Equal(t, csm.currentEpoch, uint64(firstEpochHeight)) // verify epoch
	for p := 0; p < numberOfProviders; p++ {
		require.Contains(t, csm.pairing, "provider"+strconv.Itoa(p)) // verify pairings
	}
}

func TestUpdateAllProvidersWithSameEpoch(t *testing.T) {
	csm := CreateConsumerSessionManager()
	pairingList := createPairingList("")
	err := csm.UpdateAllProviders(firstEpochHeight, pairingList)
	require.Nil(t, err)
	err = csm.UpdateAllProviders(firstEpochHeight, pairingList)
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
	pairingList := createPairingList("")
	err := csm.UpdateAllProviders(firstEpochHeight, pairingList)
	require.Nil(t, err)
	cs, epoch, _, _, err := csm.GetSession(ctx, cuForFirstRequest, nil)
	require.Nil(t, err)
	require.NotNil(t, cs)
	require.Equal(t, epoch, csm.currentEpoch)
	require.Equal(t, cs.LatestRelayCu, cuForFirstRequest)
}

func TestContext(t *testing.T) {
	ctx := context.Background()
	ctxTO, cancel := context.WithTimeout(ctx, time.Millisecond)
	time.Sleep(2 * time.Millisecond)
	require.Equal(t, ctxTO.Err(), context.DeadlineExceeded)
	cancel()
}

func TestGrpcClientHang(t *testing.T) {
	ctx := context.Background()
	s := createGRPCServer(t) // create a grpcServer so we can connect to its endpoint and validate everything works.
	defer s.Stop()           // stop the server when finished.
	conn, err := grpc.DialContext(ctx, grpcListener, grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock())
	require.NoError(t, err)
	client := pairingtypes.NewRelayerClient(conn)
	err = conn.Close()
	require.NoError(t, err)
	err = conn.Close()
	require.Error(t, err)
	_, err = client.Probe(ctx, &wrapperspb.UInt64Value{})
	fmt.Println(err)
	require.Error(t, err)
}
