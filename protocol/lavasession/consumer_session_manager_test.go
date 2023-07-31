package lavasession

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/lavanet/lava/protocol/common"
	"github.com/lavanet/lava/protocol/provideroptimizer"
	"github.com/lavanet/lava/utils"
	pairingtypes "github.com/lavanet/lava/x/pairing/types"
	"github.com/lavanet/lava/x/rand"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
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
	servicedBlockNumber                = int64(30)
	relayNumberAfterFirstCall          = uint64(1)
	relayNumberAfterFirstFail          = uint64(1)
	latestRelayCuAfterDone             = uint64(0)
	cuSumOnFailure                     = uint64(0)
)

// This variable will hold grpc server address
var grpcListener = "localhost:0"

func CreateConsumerSessionManager() *ConsumerSessionManager {
	AllowInsecureConnectionToProviders = true // set to allow insecure for tests purposes
	rand.Seed(time.Now().UnixNano())
	baseLatency := common.AverageWorldLatency / 2 // we want performance to be half our timeout or better
	return NewConsumerSessionManager(&RPCEndpoint{"stub", "stub", "stub", 0}, provideroptimizer.NewProviderOptimizer(provideroptimizer.STRATEGY_BALANCED, 0, baseLatency, 1))
}

var grpcServer *grpc.Server

func TestMain(m *testing.M) {
	serverStarted := make(chan struct{})

	go func() {
		err := createGRPCServer(serverStarted)
		if err != nil {
			fmt.Printf("Failed to start server: %v\n", err)
			os.Exit(1)
		}
	}()

	// Wait for server to start
	<-serverStarted

	// Start running tests.
	code := m.Run()

	os.Exit(code)
}

func createGRPCServer(serverStarted chan struct{}) error {
	if grpcServer != nil {
		close(serverStarted)
		return nil
	}
	lis, err := net.Listen("tcp", grpcListener)
	if err != nil {
		return err
	}

	// Update the grpcListener with the actual address
	grpcListener = lis.Addr().String()

	// Create a new server with insecure credentials
	tlsConfig := GetTlsConfig(NetworkAddressData{})
	s := grpc.NewServer(grpc.Creds(credentials.NewTLS(tlsConfig)))

	go func() {
		if err := s.Serve(lis); err != nil {
			log.Fatalf("Failed to serve: %v", err)
		}
	}()

	grpcServer = s
	close(serverStarted) // Signal that the server has started
	return nil
}

func createPairingList(providerPrefixAddress string, enabled bool) map[uint64]*ConsumerSessionsWithProvider {
	cswpList := make(map[uint64]*ConsumerSessionsWithProvider, 0)
	pairingEndpoints := make([]*Endpoint, 1)
	// we need a grpc server to connect to. so we use the public rpc endpoint for now.
	pairingEndpoints[0] = &Endpoint{NetworkAddress: grpcListener, Enabled: enabled, Client: nil, ConnectionRefusals: 0, Addons: []string{}}
	pairingEndpointsWithAddon := []*Endpoint{{NetworkAddress: grpcListener, Enabled: enabled, Client: nil, ConnectionRefusals: 0, Addons: []string{"addon"}}}
	for p := 0; p < numberOfProviders; p++ {
		endpoints := pairingEndpoints
		if p <= 1 {
			endpoints = pairingEndpointsWithAddon
		}
		cswpList[uint64(p)] = &ConsumerSessionsWithProvider{
			PublicLavaAddress: "provider" + providerPrefixAddress + strconv.Itoa(p),
			Endpoints:         endpoints,
			Sessions:          map[int64]*SingleConsumerSession{},
			MaxComputeUnits:   200,
			PairingEpoch:      firstEpochHeight,
		}
	}
	return cswpList
}

// Test the basic functionality of the consumerSessionManager
func TestHappyFlow(t *testing.T) {
	ctx := context.Background()
	csm := CreateConsumerSessionManager()
	pairingList := createPairingList("", true)
	err := csm.UpdateAllProviders(firstEpochHeight, pairingList) // update the providers.
	require.Nil(t, err)
	css, err := csm.GetSessions(ctx, cuForFirstRequest, nil, servicedBlockNumber, "") // get a session
	require.Nil(t, err)

	for _, cs := range css {
		require.NotNil(t, cs)
		require.Equal(t, cs.Epoch, csm.currentEpoch)
		require.Equal(t, cs.Session.LatestRelayCu, cuForFirstRequest)
		err = csm.OnSessionDone(cs.Session, servicedBlockNumber, cuForFirstRequest, time.Millisecond, cs.Session.CalculateExpectedLatency(2*time.Millisecond), (servicedBlockNumber - 1), numberOfProviders, numberOfProviders, false)
		require.Nil(t, err)
		require.Equal(t, cs.Session.CuSum, cuForFirstRequest)
		require.Equal(t, cs.Session.LatestRelayCu, latestRelayCuAfterDone)
		require.Equal(t, cs.Session.RelayNum, relayNumberAfterFirstCall)
		require.Equal(t, cs.Session.LatestBlock, servicedBlockNumber)
	}
}

func TestPairingReset(t *testing.T) {
	ctx := context.Background()
	csm := CreateConsumerSessionManager()
	pairingList := createPairingList("", true)
	err := csm.UpdateAllProviders(firstEpochHeight, pairingList) // update the providers.
	require.Nil(t, err)
	csm.validAddresses = []string{}                                                   // set valid addresses to zero
	css, err := csm.GetSessions(ctx, cuForFirstRequest, nil, servicedBlockNumber, "") // get a session
	require.Nil(t, err)
	require.Equal(t, len(csm.validAddresses), len(csm.pairingAddresses))

	for _, cs := range css {
		require.NotNil(t, cs)
		require.Equal(t, cs.Epoch, csm.currentEpoch)
		require.Equal(t, cs.Session.LatestRelayCu, cuForFirstRequest)
		err = csm.OnSessionDone(cs.Session, servicedBlockNumber, cuForFirstRequest, time.Millisecond, cs.Session.CalculateExpectedLatency(2*time.Millisecond), (servicedBlockNumber - 1), numberOfProviders, numberOfProviders, false)
		require.Nil(t, err)
		require.Equal(t, cs.Session.CuSum, cuForFirstRequest)
		require.Equal(t, cs.Session.LatestRelayCu, latestRelayCuAfterDone)
		require.Equal(t, cs.Session.RelayNum, relayNumberAfterFirstCall)
		require.Equal(t, cs.Session.LatestBlock, servicedBlockNumber)
		require.Equal(t, csm.numberOfResets, uint64(0x1)) // verify we had one reset only
	}
}

func TestPairingResetWithFailures(t *testing.T) {
	ctx := context.Background()
	csm := CreateConsumerSessionManager()
	pairingList := createPairingList("", true)
	err := csm.UpdateAllProviders(firstEpochHeight, pairingList) // update the providers.
	require.Nil(t, err)
	for {
		utils.LavaFormatDebug(fmt.Sprintf("%v", len(csm.validAddresses)))
		if len(csm.validAddresses) == 0 { // wait for all pairings to be blocked.
			break
		}
		css, err := csm.GetSessions(ctx, cuForFirstRequest, nil, servicedBlockNumber, "") // get a session
		require.Nil(t, err)

		for _, cs := range css {
			err = csm.OnSessionFailure(cs.Session, nil)
			require.Nil(t, err) // fail test.
		}
	}
	require.Equal(t, len(csm.validAddresses), 0)
	css, err := csm.GetSessions(ctx, cuForFirstRequest, nil, servicedBlockNumber, "") // get a session
	require.Nil(t, err)
	require.Equal(t, len(csm.validAddresses), len(csm.pairingAddresses))

	for _, cs := range css {
		require.NotNil(t, cs)
		require.Equal(t, cs.Epoch, csm.currentEpoch)
		require.Equal(t, cs.Session.LatestRelayCu, cuForFirstRequest)
		require.Equal(t, csm.numberOfResets, uint64(0x1)) // verify we had one reset only
	}
}

func TestPairingResetWithMultipleFailures(t *testing.T) {
	ctx := context.Background()
	csm := CreateConsumerSessionManager()
	pairingList := createPairingList("", true)
	err := csm.UpdateAllProviders(firstEpochHeight, pairingList) // update the providers.
	require.Nil(t, err)

	for numberOfResets := 0; numberOfResets < numberOfResetsToTest; numberOfResets++ {
		for {
			utils.LavaFormatDebug(fmt.Sprintf("%v", len(csm.validAddresses)))
			if len(csm.validAddresses) == 0 { // wait for all pairings to be blocked.
				break
			}
			css, err := csm.GetSessions(ctx, cuForFirstRequest, nil, servicedBlockNumber, "") // get a session

			for _, cs := range css {
				err = csm.OnSessionFailure(cs.Session, nil)
				require.Nil(t, err)
			}

			if len(csm.validAddresses) == 0 && PairingListEmptyError.Is(err) { // wait for all pairings to be blocked.
				break
			}
		}
		require.Equal(t, len(csm.validAddresses), 0)
		css, err := csm.GetSessions(ctx, cuForFirstRequest, nil, servicedBlockNumber, "") // get a session
		require.Nil(t, err)
		require.Equal(t, len(csm.validAddresses), len(csm.pairingAddresses))

		for _, cs := range css {
			require.NotNil(t, cs)
			require.Equal(t, cs.Epoch, csm.currentEpoch)
			require.Equal(t, cs.Session.LatestRelayCu, cuForFirstRequest)
			require.Equal(t, csm.numberOfResets, uint64(numberOfResets+1)) // verify we had one reset only
		}
	}

	css, err := csm.GetSessions(ctx, cuForFirstRequest, nil, servicedBlockNumber, "") // get a session
	require.Nil(t, err)

	for _, cs := range css {
		require.NotNil(t, cs)
		require.Equal(t, cs.Epoch, csm.currentEpoch)
		require.Equal(t, cs.Session.LatestRelayCu, cuForFirstRequest)
		err = csm.OnSessionDone(cs.Session, servicedBlockNumber, cuForFirstRequest, time.Millisecond, cs.Session.CalculateExpectedLatency(2*time.Millisecond), (servicedBlockNumber - 1), numberOfProviders, numberOfProviders, false)
		require.Nil(t, err)
		require.Equal(t, cs.Session.CuSum, cuForFirstRequest)
		require.Equal(t, cs.Session.LatestRelayCu, latestRelayCuAfterDone)
		require.GreaterOrEqual(t, cs.Session.RelayNum, relayNumberAfterFirstFail)
		require.Equal(t, cs.Session.LatestBlock, servicedBlockNumber)
	}
}

// Test the basic functionality of the consumerSessionManager
func TestSuccessAndFailureOfSessionWithUpdatePairingsInTheMiddle(t *testing.T) {
	ctx := context.Background()
	csm := CreateConsumerSessionManager()
	pairingList := createPairingList("", true)
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
		css, err := csm.GetSessions(ctx, cuForFirstRequest, nil, servicedBlockNumber, "") // get a session
		require.Nil(t, err)

		for _, cs := range css { // get a session
			require.NotNil(t, cs)
			require.Equal(t, cs.Epoch, csm.currentEpoch)
			require.Equal(t, cs.Session.LatestRelayCu, cuForFirstRequest)
			sessionList[i] = session{cs: cs.Session, epoch: cs.Epoch}
		}
	}

	for j := 0; j < numberOfAllowedSessionsPerConsumer/2; j++ {
		cs := sessionList[j].cs
		require.NotNil(t, cs)
		epoch := sessionList[j].epoch
		require.Equal(t, epoch, csm.currentEpoch)

		if rand.Intn(2) > 0 {
			err = csm.OnSessionDone(cs, servicedBlockNumber, cuForFirstRequest, time.Millisecond, cs.CalculateExpectedLatency(2*time.Millisecond), (servicedBlockNumber - 1), numberOfProviders, numberOfProviders, false)
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
		css, err := csm.GetSessions(ctx, cuForFirstRequest, nil, servicedBlockNumber, "") // get a session
		require.Nil(t, err)

		for _, cs := range css { // get a session
			require.NotNil(t, cs)
			require.Equal(t, cs.Epoch, csm.currentEpoch)
			require.Equal(t, cs.Session.LatestRelayCu, cuForFirstRequest)
		}
	}

	err = csm.UpdateAllProviders(secondEpochHeight, createPairingList("test2", true)) // update the providers. with half of them
	require.Nil(t, err)

	for j := numberOfAllowedSessionsPerConsumer / 2; j < numberOfAllowedSessionsPerConsumer; j++ {
		cs := sessionList[j].cs
		if rand.Intn(2) > 0 {
			err = csm.OnSessionDone(cs, servicedBlockNumber, cuForFirstRequest, time.Millisecond, cs.CalculateExpectedLatency(2*time.Millisecond), (servicedBlockNumber - 1), numberOfProviders, numberOfProviders, false)
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
	css, err := csm.GetSessions(ctx, cuForFirstRequest, nil, servicedBlockNumber, "") // get a session
	require.Nil(t, err)

	for _, cs := range css {
		require.NotNil(t, cs)
		time.Sleep(time.Duration((rand.Intn(500) + 1)) * time.Millisecond)
		err = csm.OnSessionDone(cs.Session, servicedBlockNumber, cuForFirstRequest, time.Millisecond, cs.Session.CalculateExpectedLatency(2*time.Millisecond), (servicedBlockNumber - 1), numberOfProviders, numberOfProviders, false)
		require.Nil(t, err)
		ch <- p
	}
}

func failedSession(ctx context.Context, csm *ConsumerSessionManager, t *testing.T, p int, ch chan int) {
	css, err := csm.GetSessions(ctx, cuForFirstRequest, nil, servicedBlockNumber, "") // get a session
	require.Nil(t, err)

	for _, cs := range css {
		require.NotNil(t, cs)
		time.Sleep(time.Duration((rand.Intn(500) + 1)) * time.Millisecond)
		err = csm.OnSessionFailure(cs.Session, fmt.Errorf("nothing special"))
		require.Nil(t, err)
		ch <- p
	}
}

func TestHappyFlowMultiThreaded(t *testing.T) {
	utils.LavaFormatInfo("Parallel test:")

	ctx := context.Background()
	csm := CreateConsumerSessionManager()
	pairingList := createPairingList("", true)
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

	ctx := context.Background()
	csm := CreateConsumerSessionManager()
	pairingList := createPairingList("", true)
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
				err := csm.UpdateAllProviders(secondEpochHeight, createPairingList("test2", true)) // update the providers. with half of them
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
	ctx := context.Background()
	csm := CreateConsumerSessionManager()
	pairingList := createPairingList("", true)
	err := csm.UpdateAllProviders(firstEpochHeight, pairingList) // update the providers.
	require.Nil(t, err)
	css, err := csm.GetSessions(ctx, cuForFirstRequest, nil, servicedBlockNumber, "") // get a session
	require.Nil(t, err)

	for _, cs := range css {
		require.NotNil(t, cs)
		require.Equal(t, cs.Epoch, csm.currentEpoch)
		require.Equal(t, cs.Session.LatestRelayCu, cuForFirstRequest)
		err = csm.OnSessionFailure(cs.Session, ReportAndBlockProviderError)
		require.Nil(t, err)
		require.Equal(t, cs.Session.Client.UsedComputeUnits, cuSumOnFailure)
		require.Equal(t, cs.Session.CuSum, cuSumOnFailure)
		require.Equal(t, cs.Session.LatestRelayCu, latestRelayCuAfterDone)
		require.Equal(t, cs.Session.RelayNum, relayNumberAfterFirstFail)

		// verify provider is blocked and reported
		require.Contains(t, csm.addedToPurgeAndReport, cs.Session.Client.PublicLavaAddress) // address is reported
		require.NotContains(t, csm.validAddresses, cs.Session.Client.PublicLavaAddress)     // address isn't in valid addresses list

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
}

// Test the basic functionality of the consumerSessionManager
func TestSessionFailureEpochMisMatch(t *testing.T) {
	ctx := context.Background()
	csm := CreateConsumerSessionManager()
	pairingList := createPairingList("", true)
	err := csm.UpdateAllProviders(firstEpochHeight, pairingList) // update the providers.
	require.Nil(t, err)
	css, err := csm.GetSessions(ctx, cuForFirstRequest, nil, servicedBlockNumber, "") // get a sesssion
	require.Nil(t, err)

	for _, cs := range css {
		require.NotNil(t, cs)
		require.Equal(t, cs.Epoch, csm.currentEpoch)
		require.Equal(t, cs.Session.LatestRelayCu, cuForFirstRequest)

		err = csm.UpdateAllProviders(secondEpochHeight, pairingList) // update the providers again.
		require.Nil(t, err)
		err = csm.OnSessionFailure(cs.Session, ReportAndBlockProviderError)
		require.Nil(t, err)
	}
}

func TestAllProvidersEndpointsDisabled(t *testing.T) {
	ctx := context.Background()
	csm := CreateConsumerSessionManager()
	pairingList := createPairingList("", false)
	err := csm.UpdateAllProviders(firstEpochHeight, pairingList) // update the providers.
	require.Nil(t, err)
	cs, err := csm.GetSessions(ctx, cuForFirstRequest, nil, servicedBlockNumber, "") // get a session
	require.Nil(t, cs)
	require.Error(t, err)
}

func TestUpdateAllProviders(t *testing.T) {
	csm := CreateConsumerSessionManager()
	pairingList := createPairingList("", true)
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
	pairingList := createPairingList("", true)
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
	ctx := context.Background()
	csm := CreateConsumerSessionManager()
	pairingList := createPairingList("", true)
	err := csm.UpdateAllProviders(firstEpochHeight, pairingList)
	require.Nil(t, err)
	css, err := csm.GetSessions(ctx, cuForFirstRequest, nil, servicedBlockNumber, "")
	require.Nil(t, err)

	for _, cs := range css {
		require.NotNil(t, cs)
		require.Equal(t, cs.Epoch, csm.currentEpoch)
		require.Equal(t, cs.Session.LatestRelayCu, cuForFirstRequest)
	}
}

func TestContext(t *testing.T) {
	ctx := context.Background()
	ctxTO, cancel := context.WithTimeout(ctx, time.Millisecond)
	time.Sleep(20 * time.Millisecond)
	require.Equal(t, ctxTO.Err(), context.DeadlineExceeded)
	cancel()
}

func TestGrpcClientHang(t *testing.T) {
	ctx := context.Background()
	conn, err := ConnectgRPCClient(ctx, grpcListener, true)
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

func TestPairingWithAddons(t *testing.T) {
	ctx := context.Background()
	for _, addon := range []string{"", "addon"} {
		t.Run(addon, func(t *testing.T) {
			csm := CreateConsumerSessionManager()
			pairingList := createPairingList("", true)
			err := csm.UpdateAllProviders(firstEpochHeight, pairingList) // update the providers.
			require.Nil(t, err)
			time.Sleep(5 * time.Millisecond) // let probes finish
			utils.LavaFormatDebug("valid providers::::", utils.Attribute{Key: "length", Value: len(csm.getValidAddresses(addon))}, utils.Attribute{Key: "valid addresses", Value: csm.getValidAddresses(addon)}, utils.Attribute{Key: "addon", Value: addon})
			require.NotEqual(t, 0, len(csm.getValidAddresses(addon)), "valid addresses: %#v addonAddresses %#v", csm.getValidAddresses(addon), csm.addonAddresses)
			// block all providers
			initialProvidersLen := len(csm.getValidAddresses(addon))
			for i := 0; i < initialProvidersLen; i++ {
				css, err := csm.GetSessions(ctx, cuForFirstRequest, nil, servicedBlockNumber, addon) // get a session
				require.Nil(t, err, i)
				for _, cs := range css {
					err = csm.OnSessionFailure(cs.Session, ReportAndBlockProviderError)
					require.Nil(t, err)
				}
				utils.LavaFormatDebug("length!", utils.Attribute{Key: "length", Value: len(csm.getValidAddresses(addon))}, utils.Attribute{Key: "valid addresses", Value: csm.getValidAddresses(addon)})
			}
			require.Equal(t, 0, len(csm.getValidAddresses(addon)), csm.validAddresses)
			if addon != "" {
				require.NotEqual(t, csm.getValidAddresses(addon), csm.getValidAddresses(""))
			}
			css, err := csm.GetSessions(ctx, cuForFirstRequest, nil, servicedBlockNumber, addon) // get a session
			require.Nil(t, err)
			for _, cs := range css {
				err = csm.OnSessionDone(cs.Session, servicedBlockNumber, cuForFirstRequest, time.Millisecond, cs.Session.CalculateExpectedLatency(2*time.Millisecond), (servicedBlockNumber - 1), numberOfProviders, numberOfProviders, false)
				require.Nil(t, err)
			}
		})
	}
}
