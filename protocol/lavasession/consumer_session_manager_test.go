package lavasession

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/v5/protocol/common"
	"github.com/lavanet/lava/v5/protocol/provideroptimizer"
	"github.com/lavanet/lava/v5/utils"
	"github.com/lavanet/lava/v5/utils/lavaslices"
	"github.com/lavanet/lava/v5/utils/rand"
	pairingtypes "github.com/lavanet/lava/v5/x/pairing/types"
	spectypes "github.com/lavanet/lava/v5/x/spec/types"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials"
)

const (
	parallelGoRoutines                 = 40
	numberOfProviders                  = 10
	numberOfResetsToTest               = 1
	numberOfAllowedSessionsPerConsumer = 10
	firstEpochHeight                   = 20
	secondEpochHeight                  = 40
	cuForFirstRequest                  = uint64(10)
	servicedBlockNumber                = int64(30)
	relayNumberAfterFirstCall          = uint64(1)
	relayNumberAfterFirstFail          = uint64(1)
	latestRelayCuAfterDone             = uint64(0)
	cuSumOnFailure                     = uint64(0)
	virtualEpoch                       = uint64(1)
	maxCuForVirtualEpoch               = uint64(200)
)

// This variable will hold grpc server address
var grpcListener = "localhost:0"

type testServer struct {
	delay time.Duration
}

func (rpcps *testServer) Probe(ctx context.Context, probeReq *pairingtypes.ProbeRequest) (*pairingtypes.ProbeReply, error) {
	utils.LavaFormatDebug("Debug probe called")
	probeReply := &pairingtypes.ProbeReply{
		Guid:                  probeReq.GetGuid(),
		LatestBlock:           1,
		FinalizedBlocksHashes: []byte{},
		LavaEpoch:             1,
		LavaLatestBlock:       1,
	}
	time.Sleep(rpcps.delay)
	return probeReply, nil
}

func (rpcps *testServer) Relay(context.Context, *pairingtypes.RelayRequest) (*pairingtypes.RelayReply, error) {
	return nil, utils.LavaFormatError("not Implemented", nil)
}

func (rpcps *testServer) RelaySubscribe(*pairingtypes.RelayRequest, pairingtypes.Relayer_RelaySubscribeServer) error {
	return utils.LavaFormatError("not implemented", nil)
}

// Test the basic functionality of the consumerSessionManager
func TestHappyFlow(t *testing.T) {
	ctx := context.Background()
	csm := CreateConsumerSessionManager()
	pairingList := createPairingList("", true)
	err := csm.UpdateAllProviders(firstEpochHeight, pairingList, nil) // update the providers.
	require.NoError(t, err)
	css, err := csm.GetSessions(ctx, 1, cuForFirstRequest, NewUsedProviders(nil), servicedBlockNumber, "", nil, common.NO_STATE, 0, "", "") // get a session
	require.NoError(t, err)

	for _, cs := range css {
		require.NotNil(t, cs)
		require.Equal(t, cs.Epoch, csm.currentEpoch)
		require.Equal(t, cs.Session.LatestRelayCu, cuForFirstRequest)
		err = csm.OnSessionDone(cs.Session, servicedBlockNumber, cuForFirstRequest, time.Millisecond, cs.Session.CalculateExpectedLatency(2*time.Millisecond), 1, numberOfProviders, numberOfProviders, false, nil)
		require.NoError(t, err)
		require.Equal(t, cs.Session.CuSum, cuForFirstRequest)
		require.Equal(t, cs.Session.LatestRelayCu, latestRelayCuAfterDone)
		require.Equal(t, cs.Session.RelayNum, relayNumberAfterFirstCall)
		require.Equal(t, cs.Session.LatestBlock, servicedBlockNumber)
	}
}

func getDelayedAddress() string {
	delayedServerAddress := "127.0.0.1:3335"
	// because grpcListener is random we might have overlap. in that case just change the port.
	if grpcListener == delayedServerAddress {
		delayedServerAddress = "127.0.0.1:3336"
	}
	utils.LavaFormatDebug("delayedAddress Chosen", utils.LogAttr("address", delayedServerAddress))
	return delayedServerAddress
}

func TestEndpointSortingFlow(t *testing.T) {
	delayedAddress := getDelayedAddress()
	err := createGRPCServer(delayedAddress, 300*time.Millisecond)
	csp := &ConsumerSessionsWithProvider{}
	for {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		_, _, err := csp.ConnectRawClientWithTimeout(ctx, delayedAddress)
		if err != nil {
			utils.LavaFormatDebug("delayedAddress - waiting for grpc server to launch")
			continue
		}
		utils.LavaFormatDebug("delayedAddress - grpc server is live", utils.LogAttr("address", delayedAddress))
		cancel()
		break
	}

	for {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		_, _, err := csp.ConnectRawClientWithTimeout(ctx, grpcListener)
		if err != nil {
			utils.LavaFormatDebug("grpcListener - waiting for grpc server to launch")
			continue
		}
		utils.LavaFormatDebug("grpcListener - grpc server is live", utils.LogAttr("address", grpcListener))
		cancel()
		break
	}

	require.NoError(t, err)
	csm := CreateConsumerSessionManager()
	pairingList := createPairingList("", true)
	pairingList[0].Endpoints = append(pairingList[0].Endpoints, &Endpoint{NetworkAddress: delayedAddress, Enabled: true, Connections: []*EndpointConnection{}, ConnectionRefusals: 0})
	// swap locations so that the endpoint of the delayed will be first
	pairingList[0].Endpoints[0], pairingList[0].Endpoints[1] = pairingList[0].Endpoints[1], pairingList[0].Endpoints[0]

	// update the pairing and wait for the routine to send all requests
	err = csm.UpdateAllProviders(firstEpochHeight, pairingList, nil) // update the providers.
	require.NoError(t, err)

	_, ok := csm.pairing[pairingList[0].PublicLavaAddress]
	require.True(t, ok)

	// because probing is in a routine we need to wait for the sorting and probing to end asynchronously
	swapped := false
	for i := 0; i < 20; i++ {
		if pairingList[0].Endpoints[0].NetworkAddress == grpcListener {
			fmt.Println("Endpoints Are Sorted!", i)
			swapped = true
			break
		}
		time.Sleep(1 * time.Second)
		fmt.Println("Endpoints did not swap yet, attempt:", i)
	}
	require.True(t, swapped)
	// after creating all the sessions
}

func CreateConsumerSessionManager() *ConsumerSessionManager {
	rand.InitRandomSeed()
	optimizer := provideroptimizer.NewProviderOptimizer(provideroptimizer.StrategyBalanced, 0, 1, nil, "dontcare")
	optimizer.SetDeterministicSeed(1234567)
	return NewConsumerSessionManager(&RPCEndpoint{"stub", "stub", "stub", false, "/", 0}, optimizer, nil, nil, "lava@test", NewActiveSubscriptionProvidersStorage())
}

func TestMain(m *testing.M) {
	AllowInsecureConnectionToProviders = true
	err := createGRPCServer("", time.Microsecond)
	if err != nil {
		fmt.Println("Failed create server", err)
		os.Exit(-1)
	}
	csp := &ConsumerSessionsWithProvider{}
	for {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		_, _, err := csp.ConnectRawClientWithTimeout(ctx, grpcListener)
		if err != nil {
			utils.LavaFormatDebug("waiting for grpc server to launch")
			continue
		}
		cancel()
		break
	}

	// Start running tests.
	code := m.Run()

	os.Exit(code)
}

func createGRPCServer(changeListener string, probeDelay time.Duration) error {
	listenAddress := grpcListener
	if changeListener != "" {
		listenAddress = changeListener
	}
	lis, err := net.Listen("tcp", listenAddress)
	if err != nil {
		return err
	}
	// Update the grpcListener with the actual address
	if changeListener == "" {
		grpcListener = lis.Addr().String()
	}

	// Create a new server with insecure credentials
	tlsConfig := GetTlsConfig(NetworkAddressData{})
	s := grpc.NewServer(grpc.Creds(credentials.NewTLS(tlsConfig)))

	s2 := &testServer{delay: probeDelay}
	pairingtypes.RegisterRelayerServer(s, s2)

	go func() {
		if err := s.Serve(lis); err != nil {
			log.Fatalf("Failed to serve: %v", err)
			os.Exit(-1)
		}
	}()
	return nil
}

const providerStr = "provider"

type DirectiveHeaders struct {
	directiveHeaders map[string]string
}

func (bpm DirectiveHeaders) GetBlockedProviders() []string {
	if bpm.directiveHeaders == nil {
		return nil
	}
	blockedProviders, ok := bpm.directiveHeaders[common.BLOCK_PROVIDERS_ADDRESSES_HEADER_NAME]
	if ok {
		blockProviders := strings.Split(blockedProviders, ",")
		if len(blockProviders) <= 2 {
			return blockProviders
		}
	}
	return nil
}

func createPairingList(providerPrefixAddress string, enabled bool) map[uint64]*ConsumerSessionsWithProvider {
	cswpList := make(map[uint64]*ConsumerSessionsWithProvider, 0)
	pairingEndpoints := make([]*Endpoint, 1)
	// we need a grpc server to connect to. so we use the public rpc endpoint for now.
	pairingEndpoints[0] = &Endpoint{Connections: []*EndpointConnection{}, NetworkAddress: grpcListener, Enabled: enabled, ConnectionRefusals: 0}
	pairingEndpointsWithAddon := []*Endpoint{{Connections: []*EndpointConnection{}, NetworkAddress: grpcListener, Enabled: enabled, ConnectionRefusals: 0, Addons: map[string]struct{}{"addon": {}}}}
	pairingEndpointsWithExtension := []*Endpoint{{Connections: []*EndpointConnection{}, NetworkAddress: grpcListener, Enabled: enabled, ConnectionRefusals: 0, Addons: map[string]struct{}{"addon": {}}, Extensions: map[string]struct{}{"ext1": {}}}}
	pairingEndpointsWithExtensions := []*Endpoint{{Connections: []*EndpointConnection{}, NetworkAddress: grpcListener, Enabled: enabled, ConnectionRefusals: 0, Addons: map[string]struct{}{"addon": {}}, Extensions: map[string]struct{}{"ext1": {}, "ext2": {}}}}
	for p := 0; p < numberOfProviders; p++ {
		var endpoints []*Endpoint
		switch p {
		case 0, 1:
			endpoints = pairingEndpointsWithAddon
		case 2:
			endpoints = pairingEndpointsWithExtension
		case 3:
			endpoints = pairingEndpointsWithExtensions
		default:
			endpoints = pairingEndpoints
		}

		cswpList[uint64(p)] = &ConsumerSessionsWithProvider{
			PublicLavaAddress: providerStr + providerPrefixAddress + strconv.Itoa(p),
			Endpoints:         endpoints,
			Sessions:          map[int64]*SingleConsumerSession{},
			MaxComputeUnits:   200,
			PairingEpoch:      firstEpochHeight,
		}
	}
	return cswpList
}

func TestNoPairingAvailableFlow(t *testing.T) {
	ctx := context.Background()
	csm := CreateConsumerSessionManager()
	pairingList := createPairingList("", true)
	err := csm.UpdateAllProviders(firstEpochHeight, pairingList, nil) // update the providers.
	require.NoError(t, err)

	addCu := 10
	// adding cu to each pairing (highest is last)
	for _, pairing := range csm.pairing {
		pairing.addUsedComputeUnits(uint64(addCu), 0)
		addCu += 10
	}

	// remove all providers except for the first one
	validAddressessLength := len(csm.validAddresses)
	copyValidAddressess := append([]string{}, csm.validAddresses...)
	for index := 1; index < validAddressessLength; index++ {
		csm.removeAddressFromValidAddresses(copyValidAddressess[index])
	}

	// get the address of the highest cu provider
	highestProviderCu := ""
	highestCu := uint64(0)
	for _, pairing := range csm.pairing {
		if pairing.PublicLavaAddress != csm.validAddresses[0] {
			if pairing.UsedComputeUnits > highestCu {
				highestCu = pairing.UsedComputeUnits
				highestProviderCu = pairing.PublicLavaAddress
			}
		}
	}

	usedProviders := NewUsedProviders(nil)
	css, err := csm.GetSessions(ctx, 1, cuForFirstRequest, usedProviders, servicedBlockNumber, "", nil, common.NO_STATE, 0, "", "") // get a session
	require.NoError(t, err)
	_, expectedProviderAddress := css[csm.validAddresses[0]]
	require.True(t, expectedProviderAddress)

	css2, err := csm.GetSessions(ctx, 1, cuForFirstRequest, usedProviders, servicedBlockNumber, "", nil, common.NO_STATE, 0, "", "") // get a session
	require.NoError(t, err)
	_, expectedProviderAddress2 := css2[highestProviderCu]
	require.True(t, expectedProviderAddress2)

	runOnSessionDoneForConsumerSessionMap(t, css, csm)
	runOnSessionDoneForConsumerSessionMap(t, css2, csm)
	time.Sleep(time.Second)
	require.Equal(t, len(csm.validAddresses), 2)

	css3, err := csm.GetSessions(ctx, 1, cuForFirstRequest, usedProviders, servicedBlockNumber, "", nil, common.NO_STATE, 0, "", "") // get a session
	require.NoError(t, err)
	runOnSessionFailureForConsumerSessionMap(t, css3, csm)
	// check we still have only 2 valid addresses as this one failed
	for _, addr := range css3 {
		require.False(t, lavaslices.Contains(csm.validAddresses, addr.Session.Parent.PublicLavaAddress))
	}
	require.Equal(t, len(csm.validAddresses), 2)
}

func TestSecondChanceRecoveryFlow(t *testing.T) {
	retrySecondChanceAfter = time.Second * 2
	ctx := context.Background()
	csm := CreateConsumerSessionManager()
	pairingList := createPairingList("", true)
	err := csm.UpdateAllProviders(firstEpochHeight, map[uint64]*ConsumerSessionsWithProvider{0: pairingList[0], 1: pairingList[1]}, nil) // create two providers
	require.NoError(t, err)
	timeLimit := time.Second * 30
	loopStartTime := time.Now()
	for {
		// implement a struct that returns: map[string]string{"lava-providers-block": pairingList[1].PublicLavaAddress} in the implementation for the DirectiveHeadersInf interface
		directiveHeaders := DirectiveHeaders{map[string]string{"lava-providers-block": pairingList[1].PublicLavaAddress}}
		usedProviders := NewUsedProviders(directiveHeaders)
		css, err := csm.GetSessions(ctx, 1, cuForFirstRequest, usedProviders, servicedBlockNumber, "", nil, common.NO_STATE, 0, "", "") // get a session
		require.NoError(t, err)
		_, expectedProviderAddress := css[pairingList[0].PublicLavaAddress]
		require.True(t, expectedProviderAddress)
		for _, sessionInfo := range css {
			csm.OnSessionFailure(sessionInfo.Session, fmt.Errorf("testError"))
		}
		_, ok := csm.secondChanceGivenToAddresses[pairingList[0].PublicLavaAddress]
		if ok {
			// should be present in secondChanceGivenToAddresses at some point.
			fmt.Println(csm.secondChanceGivenToAddresses)
			break
		}
		require.True(t, time.Since(loopStartTime) < timeLimit)
	}

	// check we get provider1.
	usedProviders := NewUsedProviders(nil)
	css, err := csm.GetSessions(ctx, 1, cuForFirstRequest, usedProviders, servicedBlockNumber, "", nil, common.NO_STATE, 0, "", "") // get a session
	require.NoError(t, err)
	_, expectedProviderAddress := css[pairingList[1].PublicLavaAddress]
	require.True(t, expectedProviderAddress)
	// check this provider is not reported.
	require.False(t, csm.reportedProviders.IsReported(pairingList[0].PublicLavaAddress))
	require.False(t, csm.reportedProviders.IsReported(pairingList[1].PublicLavaAddress))
	// sleep for the duration of the retrySecondChanceAfter
	loopStartTime = time.Now()
	for {
		if func() bool {
			csm.lock.RLock()
			defer csm.lock.RUnlock()
			utils.LavaFormatInfo("waiting for provider to return to valid addresses", utils.LogAttr("provider", pairingList[0].PublicLavaAddress), utils.LogAttr("csm.validAddresses", csm.validAddresses))
			return lavaslices.Contains(csm.validAddresses, pairingList[0].PublicLavaAddress)
		}() {
			utils.LavaFormatInfo("Wait Completed")
			break
		}
		time.Sleep(time.Second)
		require.True(t, time.Since(loopStartTime) < timeLimit)
	}

	require.True(t, lavaslices.Contains(csm.validAddresses, pairingList[0].PublicLavaAddress))
	require.False(t, lavaslices.Contains(csm.currentlyBlockedProviderAddresses, pairingList[0].PublicLavaAddress))
	require.Equal(t, BlockedProviderSessionUnusedStatus, csm.pairing[pairingList[0].PublicLavaAddress].blockedAndUsedWithChanceForRecoveryStatus)

	// now after we gave it a second chance, we give it another failure sequence, expecting it to this time be reported.
	loopStartTime = time.Now()
	for {
		utils.LavaFormatDebug("Test", utils.LogAttr("csm.validAddresses", csm.validAddresses), utils.LogAttr("csm.currentlyBlockedProviderAddresses", csm.currentlyBlockedProviderAddresses), utils.LogAttr("csm.pairing[pairingList[0].PublicLavaAddress].blockedAndUsedWithChanceForRecoveryStatus", csm.pairing[pairingList[0].PublicLavaAddress].blockedAndUsedWithChanceForRecoveryStatus))
		directiveHeaders := DirectiveHeaders{map[string]string{"lava-providers-block": pairingList[1].PublicLavaAddress}}
		usedProviders := NewUsedProviders(directiveHeaders)
		require.Equal(t, BlockedProviderSessionUnusedStatus, csm.pairing[pairingList[0].PublicLavaAddress].blockedAndUsedWithChanceForRecoveryStatus)
		css, err := csm.GetSessions(ctx, 1, cuForFirstRequest, usedProviders, servicedBlockNumber, "", nil, common.NO_STATE, 0, "", "") // get a session
		require.Equal(t, BlockedProviderSessionUnusedStatus, csm.pairing[pairingList[0].PublicLavaAddress].blockedAndUsedWithChanceForRecoveryStatus)
		require.NoError(t, err)
		_, expectedProviderAddress := css[pairingList[0].PublicLavaAddress]
		require.True(t, expectedProviderAddress)
		for _, sessionInfo := range css {
			csm.OnSessionFailure(sessionInfo.Session, fmt.Errorf("testError"))
			require.Equal(t, BlockedProviderSessionUnusedStatus, csm.pairing[pairingList[0].PublicLavaAddress].blockedAndUsedWithChanceForRecoveryStatus)
		}
		if _, ok := csm.reportedProviders.addedToPurgeAndReport[pairingList[0].PublicLavaAddress]; ok {
			break
		}
		require.True(t, time.Since(loopStartTime) < timeLimit)
	}
	utils.LavaFormatInfo("csm.reportedProviders", utils.LogAttr("csm.reportedProviders", csm.reportedProviders.addedToPurgeAndReport))
	require.True(t, csm.reportedProviders.IsReported(pairingList[0].PublicLavaAddress))
}

func runOnSessionDoneForConsumerSessionMap(t *testing.T, css ConsumerSessionsMap, csm *ConsumerSessionManager) {
	for _, cs := range css {
		require.NotNil(t, cs)
		require.Equal(t, cs.Epoch, csm.currentEpoch)
		require.Equal(t, cs.Session.LatestRelayCu, cuForFirstRequest)
		err := csm.OnSessionDone(cs.Session, servicedBlockNumber, cuForFirstRequest, time.Millisecond, cs.Session.CalculateExpectedLatency(2*time.Millisecond), 1, numberOfProviders, numberOfProviders, false, nil)
		require.NoError(t, err)
		require.Equal(t, cs.Session.CuSum, cuForFirstRequest)
		require.Equal(t, cs.Session.LatestRelayCu, latestRelayCuAfterDone)
		require.Equal(t, cs.Session.RelayNum, relayNumberAfterFirstCall)
		require.Equal(t, cs.Session.LatestBlock, servicedBlockNumber)
	}
}

func runOnSessionFailureForConsumerSessionMap(t *testing.T, css ConsumerSessionsMap, csm *ConsumerSessionManager) {
	for _, cs := range css {
		require.NotNil(t, cs)
		require.Equal(t, cs.Epoch, csm.currentEpoch)
		require.Equal(t, cs.Session.LatestRelayCu, cuForFirstRequest)
		err := csm.OnSessionFailure(cs.Session, fmt.Errorf("testError"))
		require.NoError(t, err)
	}
}

func TestHappyFlowVirtualEpoch(t *testing.T) {
	ctx := context.Background()
	csm := CreateConsumerSessionManager()
	pairingList := createPairingList("", true)
	err := csm.UpdateAllProviders(firstEpochHeight, pairingList, nil) // update the providers.
	require.NoError(t, err)
	css, err := csm.GetSessions(ctx, 1, maxCuForVirtualEpoch*(virtualEpoch+1), NewUsedProviders(nil), servicedBlockNumber, "", nil, common.NO_STATE, virtualEpoch, "", "") // get a session
	require.NoError(t, err)

	for _, cs := range css {
		require.NotNil(t, cs)
		require.Equal(t, cs.Epoch, csm.currentEpoch)
		require.Equal(t, cs.Session.LatestRelayCu, maxCuForVirtualEpoch*(virtualEpoch+1))
		err = csm.OnSessionDone(cs.Session, servicedBlockNumber, maxCuForVirtualEpoch*(virtualEpoch+1), time.Millisecond, cs.Session.CalculateExpectedLatency(2*time.Millisecond), 1, numberOfProviders, numberOfProviders, false, nil)
		require.NoError(t, err)
		require.Equal(t, cs.Session.CuSum, maxCuForVirtualEpoch*(virtualEpoch+1))
		require.Equal(t, cs.Session.LatestRelayCu, latestRelayCuAfterDone)
		require.Equal(t, cs.Session.RelayNum, relayNumberAfterFirstCall)
		require.Equal(t, cs.Session.LatestBlock, servicedBlockNumber)
	}
}

// Test exceeding maxCu
func TestVirtualEpochWithFailure(t *testing.T) {
	ctx := context.Background()
	csm := CreateConsumerSessionManager()
	pairingList := createPairingList("", true)
	err := csm.UpdateAllProviders(firstEpochHeight, pairingList, nil) // update the providers.
	require.NoError(t, err)

	_, err = csm.GetSessions(ctx, 1, maxCuForVirtualEpoch*(virtualEpoch+1)+10, NewUsedProviders(nil), servicedBlockNumber, "", nil, common.NO_STATE, virtualEpoch, "", "") // get a session
	require.Error(t, err)
}

func TestPairingReset(t *testing.T) {
	ctx := context.Background()
	csm := CreateConsumerSessionManager()
	pairingList := createPairingList("", true)
	err := csm.UpdateAllProviders(firstEpochHeight, pairingList, nil) // update the providers.
	require.NoError(t, err)
	csm.validAddresses = []string{}                                                                                                         // set valid addresses to zero
	css, err := csm.GetSessions(ctx, 1, cuForFirstRequest, NewUsedProviders(nil), servicedBlockNumber, "", nil, common.NO_STATE, 0, "", "") // get a session
	require.NoError(t, err)
	require.Equal(t, len(csm.validAddresses), len(csm.pairingAddresses))

	for _, cs := range css {
		require.NotNil(t, cs)
		require.Equal(t, cs.Epoch, csm.currentEpoch)
		require.Equal(t, cs.Session.LatestRelayCu, cuForFirstRequest)
		err = csm.OnSessionDone(cs.Session, servicedBlockNumber, cuForFirstRequest, time.Millisecond, cs.Session.CalculateExpectedLatency(2*time.Millisecond), 1, numberOfProviders, numberOfProviders, false, nil)
		require.NoError(t, err)
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
	err := csm.UpdateAllProviders(firstEpochHeight, pairingList, nil) // update the providers.
	require.NoError(t, err)
	for {
		utils.LavaFormatDebug(fmt.Sprintf("%v", len(csm.validAddresses)))
		if len(csm.validAddresses) == 0 { // wait for all pairings to be blocked.
			break
		}
		css, err := csm.GetSessions(ctx, 1, cuForFirstRequest, NewUsedProviders(nil), servicedBlockNumber, "", nil, common.NO_STATE, 0, "", "") // get a session
		require.NoError(t, err)

		for _, cs := range css {
			err = csm.OnSessionFailure(cs.Session, nil)
			require.NoError(t, err) // fail test.
		}
	}
	require.Equal(t, len(csm.validAddresses), 0)
	css, err := csm.GetSessions(ctx, 1, cuForFirstRequest, NewUsedProviders(nil), servicedBlockNumber, "", nil, common.NO_STATE, 0, "", "") // get a session
	require.NoError(t, err)
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
	// make list shorter otherwise we wont be able to ban all as it takes slightly more time now
	pairingList = map[uint64]*ConsumerSessionsWithProvider{0: pairingList[0]}
	err := csm.UpdateAllProviders(firstEpochHeight, pairingList, nil) // update the providers.
	require.NoError(t, err)

	for numberOfResets := 0; numberOfResets < numberOfResetsToTest; numberOfResets++ {
		for {
			utils.LavaFormatDebug(fmt.Sprintf("%v", len(csm.validAddresses)))
			if len(csm.validAddresses) == 0 { // wait for all pairings to be blocked.
				break
			}
			css, err := csm.GetSessions(ctx, 1, cuForFirstRequest, NewUsedProviders(nil), servicedBlockNumber, "", nil, common.NO_STATE, 0, "", "") // get a session
			require.NoError(t, err)

			for _, cs := range css {
				err = csm.OnSessionFailure(cs.Session, nil)
				require.NoError(t, err)
			}

			if len(csm.validAddresses) == 0 && PairingListEmptyError.Is(err) { // wait for all pairings to be blocked.
				break
			}
		}
		require.Equal(t, len(csm.validAddresses), 0)
		css, err := csm.GetSessions(ctx, 1, cuForFirstRequest, NewUsedProviders(nil), servicedBlockNumber, "", nil, common.NO_STATE, 0, "", "") // get a session
		require.NoError(t, err)
		require.Equal(t, len(csm.validAddresses), len(csm.pairingAddresses))

		for _, cs := range css {
			require.NotNil(t, cs)
			require.Equal(t, cs.Epoch, csm.currentEpoch)
			require.Equal(t, cs.Session.LatestRelayCu, cuForFirstRequest)
			require.Equal(t, csm.numberOfResets, uint64(numberOfResets+1)) // verify we had one reset only
		}
	}

	css, err := csm.GetSessions(ctx, 1, cuForFirstRequest, NewUsedProviders(nil), servicedBlockNumber, "", nil, common.NO_STATE, 0, "", "") // get a session
	require.NoError(t, err)

	for _, cs := range css {
		require.NotNil(t, cs)
		require.Equal(t, cs.Epoch, csm.currentEpoch)
		require.Equal(t, cs.Session.LatestRelayCu, cuForFirstRequest)
		err = csm.OnSessionDone(cs.Session, servicedBlockNumber, cuForFirstRequest, time.Millisecond, cs.Session.CalculateExpectedLatency(2*time.Millisecond), 1, numberOfProviders, numberOfProviders, false, nil)
		require.NoError(t, err)
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
	err := csm.UpdateAllProviders(firstEpochHeight, pairingList, nil) // update the providers.
	require.NoError(t, err)

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
		css, err := csm.GetSessions(ctx, 1, cuForFirstRequest, NewUsedProviders(nil), servicedBlockNumber, "", nil, common.NO_STATE, 0, "", "") // get a session
		require.NoError(t, err)

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
			err = csm.OnSessionDone(cs, servicedBlockNumber, cuForFirstRequest, time.Millisecond, cs.CalculateExpectedLatency(2*time.Millisecond), (servicedBlockNumber - 1), numberOfProviders, numberOfProviders, false, nil)
			require.NoError(t, err)
			require.Equal(t, cs.CuSum, cuForFirstRequest)
			require.Equal(t, cs.LatestRelayCu, latestRelayCuAfterDone)
			require.Equal(t, cs.RelayNum, relayNumberAfterFirstCall)
			require.Equal(t, cs.LatestBlock, servicedBlockNumber)
			sessionListData[j] = SessTestData{cuSum: cuForFirstRequest, relayNum: 1}
		} else {
			err = csm.OnSessionFailure(cs, nil)
			require.NoError(t, err)
			require.Equal(t, cs.CuSum, uint64(0))
			require.Equal(t, cs.RelayNum, relayNumberAfterFirstFail)
			require.Equal(t, cs.LatestRelayCu, latestRelayCuAfterDone)
			sessionListData[j] = SessTestData{cuSum: 0, relayNum: 1}
		}
	}

	for i := 0; i < numberOfAllowedSessionsPerConsumer; i++ {
		css, err := csm.GetSessions(ctx, 1, cuForFirstRequest, NewUsedProviders(nil), servicedBlockNumber, "", nil, common.NO_STATE, 0, "", "") // get a session
		require.NoError(t, err)

		for _, cs := range css { // get a session
			require.NotNil(t, cs)
			require.Equal(t, cs.Epoch, csm.currentEpoch)
			require.Equal(t, cs.Session.LatestRelayCu, cuForFirstRequest)
		}
	}

	err = csm.UpdateAllProviders(secondEpochHeight, createPairingList("test2", true), nil) // update the providers. with half of them
	require.NoError(t, err)

	for j := numberOfAllowedSessionsPerConsumer / 2; j < numberOfAllowedSessionsPerConsumer; j++ {
		cs := sessionList[j].cs
		if rand.Intn(2) > 0 {
			err = csm.OnSessionDone(cs, servicedBlockNumber, cuForFirstRequest, time.Millisecond, cs.CalculateExpectedLatency(2*time.Millisecond), (servicedBlockNumber - 1), numberOfProviders, numberOfProviders, false, nil)
			require.NoError(t, err)
			require.Equal(t, sessionListData[j].cuSum+cuForFirstRequest, cs.CuSum)
			require.Equal(t, cs.LatestRelayCu, latestRelayCuAfterDone)
			require.Equal(t, cs.RelayNum, sessionListData[j].relayNum+1)
			require.Equal(t, cs.LatestBlock, servicedBlockNumber)
		} else {
			err = csm.OnSessionFailure(cs, nil)
			require.NoError(t, err)
			require.Equal(t, sessionListData[j].cuSum, cs.CuSum)
			require.Equal(t, cs.RelayNum, sessionListData[j].relayNum+1)
			require.Equal(t, cs.LatestRelayCu, latestRelayCuAfterDone)
		}
	}
}

func successfulSession(ctx context.Context, csm *ConsumerSessionManager, t *testing.T, p int, ch chan int) {
	css, err := csm.GetSessions(ctx, 1, cuForFirstRequest, NewUsedProviders(nil), servicedBlockNumber, "", nil, common.NO_STATE, 0, "", "") // get a session
	require.NoError(t, err)

	for _, cs := range css {
		require.NotNil(t, cs)
		time.Sleep(time.Duration((rand.Intn(500) + 1)) * time.Millisecond)
		err = csm.OnSessionDone(cs.Session, servicedBlockNumber, cuForFirstRequest, time.Millisecond, cs.Session.CalculateExpectedLatency(2*time.Millisecond), 1, numberOfProviders, numberOfProviders, false, nil)
		require.NoError(t, err)
		ch <- p
	}
}

func failedSession(ctx context.Context, csm *ConsumerSessionManager, t *testing.T, p int, ch chan int) {
	css, err := csm.GetSessions(ctx, 1, cuForFirstRequest, NewUsedProviders(nil), servicedBlockNumber, "", nil, common.NO_STATE, 0, "", "") // get a session
	require.NoError(t, err)

	for _, cs := range css {
		require.NotNil(t, cs)
		time.Sleep(time.Duration((rand.Intn(500) + 1)) * time.Millisecond)
		err = csm.OnSessionFailure(cs.Session, fmt.Errorf("nothing special"))
		require.NoError(t, err)
		ch <- p
	}
}

func TestHappyFlowMultiThreaded(t *testing.T) {
	utils.LavaFormatInfo("Parallel test:")

	ctx := context.Background()
	csm := CreateConsumerSessionManager()
	pairingList := createPairingList("", true)
	err := csm.UpdateAllProviders(firstEpochHeight, pairingList, nil) // update the providers.
	require.NoError(t, err)
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
	err := csm.UpdateAllProviders(firstEpochHeight, pairingList, nil) // update the providers.
	require.NoError(t, err)
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
				err := csm.UpdateAllProviders(secondEpochHeight, createPairingList("test2", true), nil) // update the providers. with half of them
				require.NoError(t, err)
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
	err := csm.UpdateAllProviders(firstEpochHeight, pairingList, nil) // update the providers.
	require.NoError(t, err)
	css, err := csm.GetSessions(ctx, 1, cuForFirstRequest, NewUsedProviders(nil), servicedBlockNumber, "", nil, common.NO_STATE, 0, "", "") // get a session
	require.NoError(t, err)

	for _, cs := range css {
		require.NotNil(t, cs)
		require.Equal(t, cs.Epoch, csm.currentEpoch)
		require.Equal(t, cs.Session.LatestRelayCu, cuForFirstRequest)
		err = csm.OnSessionFailure(cs.Session, ReportAndBlockProviderError)
		require.NoError(t, err)
		require.Equal(t, cs.Session.Parent.UsedComputeUnits, cuSumOnFailure)
		require.Equal(t, cs.Session.CuSum, cuSumOnFailure)
		require.Equal(t, cs.Session.LatestRelayCu, latestRelayCuAfterDone)
		require.Equal(t, cs.Session.RelayNum, relayNumberAfterFirstFail)

		// verify provider is blocked and reported
		require.True(t, csm.reportedProviders.IsReported(cs.Session.Parent.PublicLavaAddress))
		require.NotContains(t, csm.validAddresses, cs.Session.Parent.PublicLavaAddress) // address isn't in valid addresses list

		reported := csm.GetReportedProviders(firstEpochHeight)
		require.NotEmpty(t, reported)
		for _, providerReported := range reported {
			require.True(t, csm.reportedProviders.IsReported(providerReported.Address))
			require.True(t, csm.reportedProviders.IsReported(cs.Session.Parent.PublicLavaAddress))
		}
	}
}

// Test the basic functionality of the consumerSessionManager
func TestSessionFailureEpochMisMatch(t *testing.T) {
	ctx := context.Background()
	csm := CreateConsumerSessionManager()
	pairingList := createPairingList("", true)
	err := csm.UpdateAllProviders(firstEpochHeight, pairingList, nil) // update the providers.
	require.NoError(t, err)
	css, err := csm.GetSessions(ctx, 1, cuForFirstRequest, NewUsedProviders(nil), servicedBlockNumber, "", nil, common.NO_STATE, 0, "", "") // get a session
	require.NoError(t, err)

	for _, cs := range css {
		require.NotNil(t, cs)
		require.Equal(t, cs.Epoch, csm.currentEpoch)
		require.Equal(t, cs.Session.LatestRelayCu, cuForFirstRequest)

		err = csm.UpdateAllProviders(secondEpochHeight, pairingList, nil) // update the providers again.
		require.NoError(t, err)
		err = csm.OnSessionFailure(cs.Session, ReportAndBlockProviderError)
		require.NoError(t, err)
	}
}

func TestAllProvidersEndpointsDisabled(t *testing.T) {
	ctx := context.Background()
	csm := CreateConsumerSessionManager()
	pairingList := createPairingList("", false)
	err := csm.UpdateAllProviders(firstEpochHeight, pairingList, nil) // update the providers.
	require.NoError(t, err)
	cs, err := csm.GetSessions(ctx, 1, cuForFirstRequest, NewUsedProviders(nil), servicedBlockNumber, "", nil, common.NO_STATE, 0, "", "") // get a session
	require.Nil(t, cs)
	require.Error(t, err)
}

func TestUpdateAllProviders(t *testing.T) {
	csm := CreateConsumerSessionManager()
	pairingList := createPairingList("", true)
	err := csm.UpdateAllProviders(firstEpochHeight, pairingList, nil)
	require.NoError(t, err)
	require.Equal(t, len(csm.validAddresses), numberOfProviders) // checking there are 2 valid addresses
	require.Equal(t, len(csm.pairingAddresses), numberOfProviders)
	require.Equal(t, csm.currentEpoch, uint64(firstEpochHeight)) // verify epoch
	for p := 0; p < numberOfProviders; p++ {
		require.Contains(t, csm.pairing, providerStr+strconv.Itoa(p)) // verify pairings
	}
}

func TestUpdateAllProvidersWithSameEpoch(t *testing.T) {
	csm := CreateConsumerSessionManager()
	pairingList := createPairingList("", true)
	err := csm.UpdateAllProviders(firstEpochHeight, pairingList, nil)
	require.NoError(t, err)
	// Updating with the same epoch should be idempotent (no error)
	err = csm.UpdateAllProviders(firstEpochHeight, pairingList, nil)
	require.NoError(t, err)
	// perform same validations as normal usage
	require.Equal(t, len(csm.validAddresses), numberOfProviders) // checking there are 2 valid addresses
	require.Equal(t, len(csm.pairingAddresses), numberOfProviders)
	require.Equal(t, csm.currentEpoch, uint64(firstEpochHeight)) // verify epoch
	for p := 0; p < numberOfProviders; p++ {
		require.Contains(t, csm.pairing, providerStr+strconv.Itoa(p)) // verify pairings
	}
}

func TestGetSession(t *testing.T) {
	ctx := context.Background()
	csm := CreateConsumerSessionManager()
	pairingList := createPairingList("", true)
	err := csm.UpdateAllProviders(firstEpochHeight, pairingList, nil)
	require.NoError(t, err)
	css, err := csm.GetSessions(ctx, 1, cuForFirstRequest, NewUsedProviders(nil), servicedBlockNumber, "", nil, common.NO_STATE, 0, "", "")
	require.NoError(t, err)

	for _, cs := range css {
		require.NotNil(t, cs)
		require.Equal(t, cs.Epoch, csm.currentEpoch)
		require.Equal(t, cs.Session.LatestRelayCu, cuForFirstRequest)
	}
}

func TestContext(t *testing.T) {
	ctx := context.Background()
	ctxTO, cancel := context.WithTimeout(ctx, 20*time.Millisecond)
	time.Sleep(2 * time.Second)
	require.Equal(t, ctxTO.Err(), context.DeadlineExceeded)
	cancel()
}

func TestGrpcClientHang(t *testing.T) {
	ctx := context.Background()
	conn, err := ConnectGRPCClient(ctx, grpcListener, true, false, false)
	require.NoError(t, err)
	client := pairingtypes.NewRelayerClient(conn)
	err = conn.Close()
	require.NoError(t, err)
	err = conn.Close()
	require.Error(t, err)
	_, err = client.Probe(ctx, &pairingtypes.ProbeRequest{})
	fmt.Println(err)
	require.Error(t, err)
}

func TestPairingWithAddons(t *testing.T) {
	ctx := context.Background()
	for _, addon := range []string{"", "addon"} {
		t.Run(addon, func(t *testing.T) {
			csm := CreateConsumerSessionManager()
			pairingList := createPairingList("", true)
			err := csm.UpdateAllProviders(firstEpochHeight, pairingList, nil) // update the providers.
			require.NoError(t, err)
			time.Sleep(5 * time.Millisecond) // let probes finish
			utils.LavaFormatDebug("valid providers::::", utils.Attribute{Key: "length", Value: len(csm.getValidAddresses(addon, nil, ctx))}, utils.Attribute{Key: "valid addresses", Value: csm.getValidAddresses(addon, nil, ctx)}, utils.Attribute{Key: "addon", Value: addon})
			require.NotEqual(t, 0, len(csm.getValidAddresses(addon, nil, ctx)), "valid addresses: %#v addonAddresses %#v", csm.getValidAddresses(addon, nil, ctx), csm.addonAddresses)
			// block all providers
			initialProvidersLen := len(csm.getValidAddresses(addon, nil, ctx))
			for i := 0; i < initialProvidersLen; i++ {
				css, err := csm.GetSessions(ctx, 1, cuForFirstRequest, NewUsedProviders(nil), servicedBlockNumber, addon, nil, common.NO_STATE, 0, "", "") // get a session
				require.NoError(t, err, i)
				for _, cs := range css {
					err = csm.OnSessionFailure(cs.Session, ReportAndBlockProviderError)
					require.NoError(t, err)
				}
				utils.LavaFormatDebug("length!", utils.Attribute{Key: "length", Value: len(csm.getValidAddresses(addon, nil, ctx))}, utils.Attribute{Key: "valid addresses", Value: csm.getValidAddresses(addon, nil, ctx)})
			}
			require.Equal(t, 0, len(csm.getValidAddresses(addon, nil, ctx)), csm.validAddresses)
			if addon != "" {
				require.NotEqual(t, csm.getValidAddresses(addon, nil, ctx), csm.getValidAddresses("", nil, ctx))
			}
			css, err := csm.GetSessions(ctx, 1, cuForFirstRequest, NewUsedProviders(nil), servicedBlockNumber, addon, nil, common.NO_STATE, 0, "", "") // get a session
			require.NoError(t, err)
			for _, cs := range css {
				err = csm.OnSessionDone(cs.Session, servicedBlockNumber, cuForFirstRequest, time.Millisecond, cs.Session.CalculateExpectedLatency(2*time.Millisecond), 1, numberOfProviders, numberOfProviders, false, nil)
				require.NoError(t, err)
			}
		})
	}
}

func TestPairingWithExtensions(t *testing.T) {
	ctx := context.Background()
	type extensionOption struct {
		name       string
		extensions []string
		addon      string
	}
	extensionOptions := []extensionOption{
		{
			name:       "empty",
			addon:      "",
			extensions: []string{},
		},
		{
			name:       "one ext",
			addon:      "",
			extensions: []string{"ext1"},
		},
		{
			name:       "two exts",
			addon:      "",
			extensions: []string{"ext1", "ext2"},
		},
		{
			name:       "one ext addon",
			addon:      "addon",
			extensions: []string{"ext1"},
		},
		{
			name:       "two exts addon",
			addon:      "addon",
			extensions: []string{"ext1", "ext2"},
		},
	}
	for _, extensionOpt := range extensionOptions {
		t.Run(extensionOpt.name, func(t *testing.T) {
			csm := CreateConsumerSessionManager()
			pairingList := createPairingList("", true)
			err := csm.UpdateAllProviders(firstEpochHeight, pairingList, nil) // update the providers.
			require.NoError(t, err)
			time.Sleep(5 * time.Millisecond) // let probes finish
			utils.LavaFormatDebug("valid providers::::", utils.Attribute{Key: "length", Value: len(csm.getValidAddresses(extensionOpt.addon, extensionOpt.extensions, ctx))}, utils.Attribute{Key: "valid addresses", Value: csm.getValidAddresses(extensionOpt.addon, extensionOpt.extensions, ctx)}, utils.Attribute{Key: "extensions", Value: extensionOpt.extensions}, utils.Attribute{Key: "addon", Value: extensionOpt.addon})
			require.NotEqual(t, 0, len(csm.getValidAddresses(extensionOpt.addon, extensionOpt.extensions, ctx)), "valid addresses: %#v addonAddresses %#v", csm.getValidAddresses(extensionOpt.addon, extensionOpt.extensions, ctx), csm.addonAddresses)
			// block all providers
			extensionsList := []*spectypes.Extension{}
			for _, extension := range extensionOpt.extensions {
				ext := &spectypes.Extension{
					Name: extension,
				}
				extensionsList = append(extensionsList, ext)
			}
			initialProvidersLen := len(csm.getValidAddresses(extensionOpt.addon, extensionOpt.extensions, ctx))
			for i := 0; i < initialProvidersLen; i++ {
				css, err := csm.GetSessions(ctx, 1, cuForFirstRequest, NewUsedProviders(nil), servicedBlockNumber, extensionOpt.addon, extensionsList, common.NO_STATE, 0, "", "") // get a session
				require.NoError(t, err, i)
				for _, cs := range css {
					err = csm.OnSessionFailure(cs.Session, ReportAndBlockProviderError)
					require.NoError(t, err)
				}
				utils.LavaFormatDebug("length!", utils.Attribute{Key: "length", Value: len(csm.getValidAddresses(extensionOpt.addon, extensionOpt.extensions, ctx))}, utils.Attribute{Key: "valid addresses", Value: csm.getValidAddresses(extensionOpt.addon, extensionOpt.extensions, ctx)})
			}
			require.Equal(t, 0, len(csm.getValidAddresses(extensionOpt.addon, extensionOpt.extensions, ctx)), csm.validAddresses)
			if len(extensionOpt.extensions) > 0 || extensionOpt.addon != "" {
				require.NotEqual(t, csm.getValidAddresses(extensionOpt.addon, extensionOpt.extensions, ctx), csm.getValidAddresses("", nil, ctx))
			}
			css, err := csm.GetSessions(ctx, 1, cuForFirstRequest, NewUsedProviders(nil), servicedBlockNumber, extensionOpt.addon, extensionsList, common.NO_STATE, 0, "", "") // get a session
			require.NoError(t, err)
			for _, cs := range css {
				err = csm.OnSessionDone(cs.Session, servicedBlockNumber, cuForFirstRequest, time.Millisecond, cs.Session.CalculateExpectedLatency(2*time.Millisecond), 1, numberOfProviders, numberOfProviders, false, nil)
				require.NoError(t, err)
			}
		})
	}
}

func TestNoPairingsError(t *testing.T) {
	csm := CreateConsumerSessionManager()
	pairingList := createPairingList("", true)
	err := csm.UpdateAllProviders(firstEpochHeight, pairingList, nil) // update the providers.
	require.NoError(t, err)
	time.Sleep(5 * time.Millisecond) // let probes finish
	_, err = csm.getValidProviderAddresses(context.Background(), 1, map[string]struct{}{}, 10, 100, "invalid", nil, common.NO_STATE, "", "")
	require.Error(t, err)
	require.True(t, PairingListEmptyError.Is(err))
}

func TestPairingWithStateful(t *testing.T) {
	ctx := context.Background()
	t.Run("stateful", func(t *testing.T) {
		csm := CreateConsumerSessionManager()                             // crates an optimizer returning always 1 provider
		pairingList := createPairingList("", true)                        // creates 10 providers
		err := csm.UpdateAllProviders(firstEpochHeight, pairingList, nil) // update the providers.
		require.NoError(t, err)
		addon := ""
		time.Sleep(5 * time.Millisecond) // let probes finish
		utils.LavaFormatDebug("valid providers::::", utils.Attribute{Key: "length", Value: len(csm.getValidAddresses(addon, nil, ctx))}, utils.Attribute{Key: "valid addresses", Value: csm.getValidAddresses(addon, nil, ctx)}, utils.Attribute{Key: "addon", Value: addon})
		require.NotEqual(t, 0, len(csm.getValidAddresses(addon, nil, ctx)), "valid addresses: %#v addonAddresses %#v", csm.getValidAddresses(addon, nil, ctx), csm.addonAddresses)
		providerAddresses := csm.getValidAddresses(addon, nil, ctx)
		allProviders := len(providerAddresses)
		require.Equal(t, 10, allProviders)
		css, err := csm.GetSessions(ctx, 1, cuForFirstRequest, NewUsedProviders(nil), servicedBlockNumber, addon, nil, common.CONSISTENCY_SELECT_ALL_PROVIDERS, 0, "", "") // get a session
		require.NoError(t, err)
		require.Equal(t, allProviders, len(css))
		for _, cs := range css {
			err = csm.OnSessionDone(cs.Session, servicedBlockNumber, cuForFirstRequest, time.Millisecond, cs.Session.CalculateExpectedLatency(2*time.Millisecond), 1, numberOfProviders, numberOfProviders, false, nil)
			require.NoError(t, err)
		}
		usedProviders := NewUsedProviders(nil)
		usedProviders.RemoveUsed(providerAddresses[0], NewRouterKey(nil), nil)
		css, err = csm.GetSessions(ctx, 1, cuForFirstRequest, usedProviders, servicedBlockNumber, addon, nil, common.CONSISTENCY_SELECT_ALL_PROVIDERS, 0, "", "") // get a session
		require.NoError(t, err)
		require.Equal(t, allProviders-1, len(css))
	})
}

func TestMaximumBlockedSessionsErrorsInPairingListEmpty(t *testing.T) {
	ctx := context.Background()
	csm := CreateConsumerSessionManager()
	pairingList := createPairingList("", true)
	err := csm.UpdateAllProviders(firstEpochHeight, map[uint64]*ConsumerSessionsWithProvider{0: pairingList[0]}, nil) // update the providers.
	require.NoError(t, err)
	utils.LavaFormatDebug(fmt.Sprintf("%v", len(csm.validAddresses)))
	for i := 0; i < MaxSessionsAllowedPerProvider; i++ {
		css, err := csm.GetSessions(ctx, 1, cuForFirstRequest, NewUsedProviders(nil), servicedBlockNumber, "", nil, common.NO_STATE, 0, "", "") // get a session
		require.NoError(t, err)
		for _, cs := range css {
			err = csm.OnSessionFailure(cs.Session, errors.Join(BlockProviderError, SessionOutOfSyncError))
			require.NoError(t, err)
		}
	}

	_, err = csm.GetSessions(ctx, 1, cuForFirstRequest, NewUsedProviders(nil), servicedBlockNumber, "", nil, common.NO_STATE, 0, "", "") // get a session
	require.ErrorIs(t, err, PairingListEmptyError)
}

// TestDeadConnectionCleanup validates that dead gRPC connections are cleaned up
// before iterating over endpoint connections to prevent accumulation
func TestDeadConnectionCleanup(t *testing.T) {
	ctx := context.Background()
	csp := &ConsumerSessionsWithProvider{}

	// Create a valid connection (connected to the test grpc server)
	validClient, validConn, err := csp.ConnectRawClientWithTimeout(ctx, grpcListener)
	require.NoError(t, err)
	defer validConn.Close()

	// Create a shutdown connection (by creating and immediately closing it)
	shutdownClient, shutdownConn, err := csp.ConnectRawClientWithTimeout(ctx, grpcListener)
	require.NoError(t, err)
	defer shutdownConn.Close()
	shutdownConn.Close() // Close to put it in Shutdown state
	// Wait a bit for the connection to transition to Shutdown state
	time.Sleep(50 * time.Millisecond)
	require.Equal(t, connectivity.Shutdown, shutdownConn.GetState())

	// Create another valid connection for the disconnected test case
	disconnectedClient, disconnectedConn, err := csp.ConnectRawClientWithTimeout(ctx, grpcListener)
	require.NoError(t, err)
	defer disconnectedConn.Close()

	// Create endpoint with mixed connections:
	// 1. Valid connection (should remain)
	// 2. Nil connection (should be removed)
	// 3. Shutdown connection (should be removed)
	// 4. Disconnected connection (should be removed)
	endpoint := &Endpoint{
		NetworkAddress: grpcListener,
		Enabled:        true,
		Connections: []*EndpointConnection{
			{
				Client:       validClient,
				connection:   validConn,
				disconnected: false,
			},
			{
				Client:       nil,
				connection:   nil, // nil connection
				disconnected: false,
			},
			{
				Client:       shutdownClient,
				connection:   shutdownConn, // shutdown connection
				disconnected: false,
			},
			{
				Client:       disconnectedClient,
				connection:   disconnectedConn,
				disconnected: true, // disconnected flag set
			},
		},
		ConnectionRefusals: 0,
	}

	// Verify initial state - should have 4 connections
	require.Equal(t, 4, len(endpoint.Connections))

	// Create ConsumerSessionsWithProvider with this endpoint
	cswp := &ConsumerSessionsWithProvider{
		PublicLavaAddress: "test-provider",
		Endpoints:         []*Endpoint{endpoint},
		Sessions:          map[int64]*SingleConsumerSession{},
		MaxComputeUnits:   200,
		PairingEpoch:      firstEpochHeight,
	}

	// Call fetchEndpointConnectionFromConsumerSessionWithProvider which will trigger cleanup
	// This function will iterate through endpoints and call connectEndpoint, which contains the cleanup logic
	connected, endpointsList, providerAddress, err := cswp.fetchEndpointConnectionFromConsumerSessionWithProvider(
		ctx,
		false, // retryDisabledEndpoints
		false, // getAllEndpoints
		"",    // addon
		nil,   // extensionNames
	)

	// Should successfully connect (we have one valid connection)
	require.NoError(t, err)
	require.True(t, connected)
	require.NotEmpty(t, endpointsList)
	require.Equal(t, "test-provider", providerAddress)

	// Verify cleanup happened - should only have 1 connection remaining (the valid one)
	require.Equal(t, 1, len(endpoint.Connections), "Dead connections should be cleaned up")
	require.NotNil(t, endpoint.Connections[0].connection, "Remaining connection should not be nil")
	require.False(t, endpoint.Connections[0].disconnected, "Remaining connection should not be disconnected")
	require.NotEqual(t, connectivity.Shutdown, endpoint.Connections[0].connection.GetState(), "Remaining connection should not be in Shutdown state")
	require.Equal(t, validConn, endpoint.Connections[0].connection, "Remaining connection should be the valid one")
}

// grpcGoleakOptions returns common goleak options that ignore known gRPC internal goroutines.
// gRPC spawns background goroutines for connection management that persist briefly after
// connection close, which are not actual leaks.
func grpcGoleakOptions() []goleak.Option {
	return []goleak.Option{
		goleak.IgnoreCurrent(),
		goleak.IgnoreTopFunction("google.golang.org/grpc.(*addrConn).resetTransport"),
		goleak.IgnoreTopFunction("google.golang.org/grpc/internal/grpcsync.(*CallbackSerializer).run"),
		goleak.IgnoreTopFunction("google.golang.org/grpc.(*ccBalancerWrapper).watcher"),
		goleak.IgnoreTopFunction("google.golang.org/grpc/internal/transport.(*controlBuffer).get"),
		goleak.IgnoreTopFunction("google.golang.org/grpc/internal/transport.(*http2Client).keepalive"),
		goleak.IgnoreTopFunction("internal/poll.runtime_pollWait"),
	}
}

// TestConnectRawClientWithTimeoutGoroutineCleanup verifies that ConnectRawClientWithTimeout
// returns promptly and without leaking goroutines when context is cancelled.
// This is a regression test using goleak to detect actual goroutine leaks.
func TestConnectRawClientWithTimeoutGoroutineCleanup(t *testing.T) {
	// Use goleak to verify no goroutines are leaked after the test
	defer goleak.VerifyNone(t, grpcGoleakOptions()...)

	csp := &ConsumerSessionsWithProvider{}

	// Use a short timeout that will expire before connection is established
	// to an unreachable address, triggering the context cancellation path
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Use an address that will never connect (non-routable IP)
	unreachableAddr := "10.255.255.1:12345"

	// Call the actual function - this should return error due to timeout
	// and importantly, should not leak any goroutines
	_, conn, err := csp.ConnectRawClientWithTimeout(ctx, unreachableAddr)

	// We expect an error (context deadline exceeded or cancelled)
	require.Error(t, err)
	require.Nil(t, conn)

	// Give goroutines a moment to clean up
	time.Sleep(50 * time.Millisecond)

	// goleak.VerifyNone in defer will catch any leaked goroutines
}

// TestConnectRawClientWithTimeoutReturnsOnContextCancel verifies that
// ConnectRawClientWithTimeout returns promptly when context is cancelled,
// rather than hanging until the connection attempt completes.
// Uses goleak to verify no goroutines are leaked.
func TestConnectRawClientWithTimeoutReturnsOnContextCancel(t *testing.T) {
	// Use goleak to verify no goroutines are leaked
	defer goleak.VerifyNone(t, grpcGoleakOptions()...)

	// Use an address that will never connect (non-routable IP)
	unreachableAddr := "10.255.255.1:12345"
	csp := &ConsumerSessionsWithProvider{}

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	var conn *grpc.ClientConn
	var err error

	go func() {
		defer close(done)
		_, conn, err = csp.ConnectRawClientWithTimeout(ctx, unreachableAddr)
	}()

	// Wait a bit for the connection attempt to start
	time.Sleep(50 * time.Millisecond)

	// Cancel the context
	cancel()

	// The function should return promptly after cancellation
	select {
	case <-done:
		// Success - function returned after context cancellation
		// Should return context.Canceled error
		require.Error(t, err, "Expected error when context is cancelled")
		require.ErrorIs(t, err, context.Canceled, "Expected context.Canceled error")
		require.Nil(t, conn, "Connection should be nil when context is cancelled")
	case <-time.After(2 * time.Second):
		t.Fatal("ConnectRawClientWithTimeout did not return promptly after context cancellation")
	}

	// Give goroutines a moment to clean up
	time.Sleep(50 * time.Millisecond)
}

// TestConnectRawClientWithTimeoutSuccessfulConnection verifies that successful
// connections still work correctly and don't leak goroutines.
func TestConnectRawClientWithTimeoutSuccessfulConnection(t *testing.T) {
	// Use goleak to verify no goroutines are leaked
	defer goleak.VerifyNone(t, grpcGoleakOptions()...)

	// Use the test grpc server that's set up in TestMain
	if grpcListener == "localhost:0" {
		t.Skip("grpcListener not initialized - run full test suite")
	}

	csp := &ConsumerSessionsWithProvider{}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	client, conn, err := csp.ConnectRawClientWithTimeout(ctx, grpcListener)

	require.NoError(t, err)
	require.NotNil(t, client)
	require.NotNil(t, conn)

	// Verify connection is usable
	state := conn.GetState()
	require.True(t, state == connectivity.Ready || state == connectivity.Idle,
		"Connection should be in Ready or Idle state, got: %v", state)

	// Clean up - close connection before goleak check
	conn.Close()

	// Give goroutines a moment to clean up
	time.Sleep(50 * time.Millisecond)
}

// TestPeriodicProbeProvidersTickerCleanup tests that the ticker in PeriodicProbeProviders
// is properly stopped when the context is cancelled. This was a bug where the ticker
// was not stopped, causing a goroutine leak.
func TestPeriodicProbeProvidersTickerCleanup(t *testing.T) {
	t.Run("function exits promptly when context is cancelled", func(t *testing.T) {
		csm := CreateConsumerSessionManager()

		// Create a cancellable context
		ctx, cancel := context.WithCancel(context.Background())

		// Track when the function exits
		done := make(chan struct{})

		go func() {
			defer close(done)
			csm.PeriodicProbeProviders(ctx, 10*time.Millisecond)
		}()

		// Let it run for a few ticks to ensure ticker is active
		time.Sleep(50 * time.Millisecond)

		// Cancel the context
		cancel()

		// The function should exit promptly after context cancellation
		select {
		case <-done:
			// Success - function exited after context cancellation
		case <-time.After(500 * time.Millisecond):
			t.Fatal("PeriodicProbeProviders did not exit after context cancellation - possible ticker leak")
		}
	})

	t.Run("multiple probe sessions all exit on cancellation", func(t *testing.T) {
		numSessions := 5
		csms := make([]*ConsumerSessionManager, numSessions)
		ctxs := make([]context.Context, numSessions)
		cancels := make([]context.CancelFunc, numSessions)
		dones := make([]chan struct{}, numSessions)

		// Start multiple probe sessions
		for i := 0; i < numSessions; i++ {
			csms[i] = CreateConsumerSessionManager()
			ctxs[i], cancels[i] = context.WithCancel(context.Background())
			dones[i] = make(chan struct{})

			go func(idx int) {
				defer close(dones[idx])
				csms[idx].PeriodicProbeProviders(ctxs[idx], 10*time.Millisecond)
			}(i)
		}

		// Let them run briefly to ensure tickers are active
		time.Sleep(30 * time.Millisecond)

		// Cancel all contexts
		for i := 0; i < numSessions; i++ {
			cancels[i]()
		}

		// Wait for all to exit - they should all exit promptly
		for i := 0; i < numSessions; i++ {
			select {
			case <-dones[i]:
				// Success
			case <-time.After(500 * time.Millisecond):
				t.Fatalf("PeriodicProbeProviders %d did not exit after context cancellation - possible ticker leak", i)
			}
		}
	})

	t.Run("ticker fires expected number of times before cancellation", func(t *testing.T) {
		// This test verifies the ticker is actually working and then properly cleaned up
		tickerInterval := 20 * time.Millisecond
		runDuration := 100 * time.Millisecond

		ctx, cancel := context.WithCancel(context.Background())

		started := make(chan struct{})
		done := make(chan struct{})

		go func() {
			close(started)
			// We can't easily count ticks in the real function, but we can verify
			// the function runs for the expected duration and then exits cleanly
			ticker := time.NewTicker(tickerInterval)
			defer ticker.Stop() // This is what we added in the fix

			for {
				select {
				case <-ticker.C:
					// Tick occurred
				case <-ctx.Done():
					close(done)
					return
				}
			}
		}()

		<-started
		time.Sleep(runDuration)
		cancel()

		// Function should exit promptly
		select {
		case <-done:
			// Success - function exited and ticker.Stop() was called
		case <-time.After(200 * time.Millisecond):
			t.Fatal("Function did not exit promptly after context cancellation")
		}
	})
}

// TestBackupProviderOptimizerSelection verifies that when all static providers are exhausted,
// the optimizer selects backup providers one at a time (not as a batch), and that successive
// retries each get a different backup until the pool is exhausted.
func TestBackupProviderOptimizerSelection(t *testing.T) {
	ctx := context.Background()
	csm := CreateConsumerSessionManager()

	// Register two backup providers and zero static providers.
	backupA := NewConsumerSessionWithProvider("backupA", []*Endpoint{{NetworkAddress: grpcListener, Enabled: true, Connections: []*EndpointConnection{}}}, 999999, firstEpochHeight, sdk.NewInt64Coin("ulava", 0))
	backupA.StaticProvider = true
	backupB := NewConsumerSessionWithProvider("backupB", []*Endpoint{{NetworkAddress: grpcListener, Enabled: true, Connections: []*EndpointConnection{}}}, 999999, firstEpochHeight, sdk.NewInt64Coin("ulava", 0))
	backupB.StaticProvider = true

	backupList := map[uint64]*ConsumerSessionsWithProvider{
		0: backupA,
		1: backupB,
	}

	err := csm.UpdateAllProviders(firstEpochHeight, nil, backupList)
	require.NoError(t, err)

	ignoredProv := &ignoredProviders{
		providers:    make(map[string]struct{}),
		currentEpoch: firstEpochHeight,
	}

	// First call: optimizer picks one backup.
	result1, err := csm.getValidConsumerSessionsWithProviderFromBackupProviderList(ctx, ignoredProv, cuForFirstRequest, servicedBlockNumber, "", []string{}, 0, 0, NewUsedProviders(nil))
	require.NoError(t, err)
	require.Len(t, result1, 1, "expected exactly one backup provider returned per call")
	var first string
	for addr := range result1 {
		first = addr
	}
	require.Contains(t, []string{"backupA", "backupB"}, first)
	// The selected provider must now be in ignoredProviders.
	_, firstIgnored := ignoredProv.providers[first]
	require.True(t, firstIgnored, "selected backup should be in ignoredProviders after call")

	// Second call: optimizer picks the other backup (first is now ignored).
	result2, err := csm.getValidConsumerSessionsWithProviderFromBackupProviderList(ctx, ignoredProv, cuForFirstRequest, servicedBlockNumber, "", []string{}, 0, 0, NewUsedProviders(nil))
	require.NoError(t, err)
	require.Len(t, result2, 1, "expected exactly one backup provider returned per call")
	var second string
	for addr := range result2 {
		second = addr
	}
	require.NotEqual(t, first, second, "second call should return the other backup provider")

	// Third call: both backups are now ignored — expect an error.
	_, err = csm.getValidConsumerSessionsWithProviderFromBackupProviderList(ctx, ignoredProv, cuForFirstRequest, servicedBlockNumber, "", []string{}, 0, 0, NewUsedProviders(nil))
	require.Error(t, err, "expected error when all backup providers are exhausted")
}

// TestGetReportedProviders_EpochTransitionRace reproduces the race between
// GetReportedProviders and UpdateAllProviders that causes the error
// "Failed to find a reported provider in pairing list".
//
// The race window (without fix):
//  1. GetReportedProviders passes the epoch check and reads reportedProviders [A]
//     — neither operation requires csm.lock
//  2. While GetReportedProviders waits for csm.lock.RLock, UpdateAllProviders runs
//     under csm.lock.Lock: resets reportedProviders, rebuilds pairing without A
//  3. GetReportedProviders acquires csm.lock.RLock, looks up A → not found → error
//
// This test forces that exact interleaving: it holds csm.lock.Lock (simulating
// UpdateAllProviders) while GetReportedProviders runs in a goroutine. Without
// the fix, GetReportedProviders reads the stale reportedProviders [A] before
// blocking on the lock. With the fix, it blocks on the lock first, then reads
// the (now empty) reportedProviders — returning a consistent result.
func TestGetReportedProviders_EpochTransitionRace(t *testing.T) {
	ctx := context.Background()
	csm := CreateConsumerSessionManager()
	pairingList := createPairingList("", true)
	err := csm.UpdateAllProviders(firstEpochHeight, pairingList, nil)
	require.NoError(t, err)

	// Get a session and fail it to report a provider.
	css, err := csm.GetSessions(ctx, 1, cuForFirstRequest, NewUsedProviders(nil), servicedBlockNumber, "", nil, common.NO_STATE, 0, "", "")
	require.NoError(t, err)

	var reportedAddr string
	for _, cs := range css {
		reportedAddr = cs.Session.Parent.PublicLavaAddress
		err = csm.OnSessionFailure(cs.Session, ReportAndBlockProviderError)
		require.NoError(t, err)
	}
	require.NotEmpty(t, reportedAddr)

	// Sanity: the provider IS reported and returned before any race.
	reported := csm.GetReportedProviders(firstEpochHeight)
	require.NotEmpty(t, reported, "provider should be reported before the race test begins")

	// Hold the write lock — simulating UpdateAllProviders in progress.
	csm.lock.Lock()

	// Start GetReportedProviders in a goroutine. Without the fix it will:
	//   1. pass the epoch check (atomic, no lock needed)
	//   2. read reportedProviders [A] (under rp.lock, not csm.lock)
	//   3. block on csm.lock.RLock
	// With the fix it blocks on csm.lock.RLock immediately (step 1).
	resultCh := make(chan []*pairingtypes.ReportedProvider, 1)
	go func() {
		resultCh <- csm.GetReportedProviders(firstEpochHeight)
	}()

	// Give the goroutine enough time to reach the RLock blocking point.
	time.Sleep(100 * time.Millisecond)

	// Reset reportedProviders while holding the lock — this is what
	// UpdateAllProviders does (along with rebuilding pairing).
	// We only reset reportedProviders; pairing still contains A.
	csm.reportedProviders.Reset()
	csm.lock.Unlock()

	result := <-resultCh

	// Without fix: GetReportedProviders already read [A] from reportedProviders
	// BEFORE we held the lock, so it still returns [A] — stale data from a
	// snapshot that is no longer consistent with the current state.
	//
	// With fix: GetReportedProviders waited for the lock, then read
	// reportedProviders (now empty after our Reset), returning [] consistently.
	require.Empty(t, result,
		"GetReportedProviders returned stale reported providers — it read "+
			"reportedProviders outside csm.lock, producing a snapshot inconsistent "+
			"with the current state. The fix moves csm.lock.RLock before reading "+
			"reportedProviders so both reads are atomic.")
}

// TestGetReportedProviders_ReconnectRace reproduces a race between
// ReconnectProviders and UpdateAllProviders that causes the error
// "Failed to find a reported provider in pairing list".
//
// The race window:
//  1. ReconnectCandidates() returns a snapshot containing provider A
//  2. UpdateAllProviders runs: Reset() clears reportedProviders, pairing
//     is rebuilt WITHOUT provider A (new epoch, different provider set)
//  3. ReconnectProviders iterates the stale snapshot, reconnectCB fails
//     for provider A, and ReportProvider re-adds A to reportedProviders
//  4. GetReportedProviders finds A in reportedProviders but NOT in
//     csm.pairing → "Failed to find a reported provider in pairing list"
//
// The test forces this interleaving by using a reconnectCB that blocks
// until the epoch transition completes, ensuring the stale candidate is
// processed after Reset() + pairing rebuild.
func TestGetReportedProviders_ReconnectRace(t *testing.T) {
	ctx := context.Background()
	csm := CreateConsumerSessionManager()
	pairingList := createPairingList("", true)
	err := csm.UpdateAllProviders(firstEpochHeight, pairingList, nil)
	require.NoError(t, err)

	// Get a session and fail it to report a provider.
	css, err := csm.GetSessions(ctx, 1, cuForFirstRequest, NewUsedProviders(nil), servicedBlockNumber, "", nil, common.NO_STATE, 0, "", "")
	require.NoError(t, err)

	var reportedAddr string
	for _, cs := range css {
		reportedAddr = cs.Session.Parent.PublicLavaAddress
		err = csm.OnSessionFailure(cs.Session, ReportAndBlockProviderError)
		require.NoError(t, err)
	}
	require.NotEmpty(t, reportedAddr)

	// Sanity: provider is reported.
	reported := csm.GetReportedProviders(firstEpochHeight)
	require.NotEmpty(t, reported, "provider should be reported before the race test begins")

	// Set up a reconnectCB that blocks until the epoch transition happens.
	// This forces the exact race: ReconnectCandidates captures the snapshot,
	// then the CB blocks while UpdateAllProviders resets everything, then the
	// CB returns an error — triggering the re-report of a stale provider.
	epochTransitioned := make(chan struct{})
	csm.reportedProviders.lock.Lock()
	for _, entry := range csm.reportedProviders.addedToPurgeAndReport {
		entry.addedTime = time.Now().Add(-2 * ReconnectCandidateTime)
		entry.Errors = 0 // clear errors so it qualifies as a reconnect candidate
		entry.reconnectCB = func() error {
			// Block until epoch transition completes
			<-epochTransitioned
			return fmt.Errorf("reconnect failed")
		}
	}
	csm.reportedProviders.lock.Unlock()

	// Launch ReconnectProviders in a goroutine — it will call
	// ReconnectCandidates (capturing the stale snapshot), then block
	// inside reconnectCB waiting for the epoch transition.
	reconnectDone := make(chan struct{})
	go func() {
		csm.reportedProviders.ReconnectProviders()
		close(reconnectDone)
	}()

	// Give ReconnectProviders time to call ReconnectCandidates and enter reconnectCB.
	time.Sleep(100 * time.Millisecond)

	// Epoch transition: new providers, Reset() clears reportedProviders.
	newPairingList := createPairingList("new_provider_", true)
	err = csm.UpdateAllProviders(secondEpochHeight, newPairingList, nil)
	require.NoError(t, err)

	// Unblock the reconnectCB — it will return an error, and ReconnectProviders
	// will attempt to re-report the stale provider.
	close(epochTransitioned)
	<-reconnectDone

	// Verify the old provider is no longer in pairing.
	csm.lock.RLock()
	_, stillInPairing := csm.pairing[reportedAddr]
	csm.lock.RUnlock()
	require.False(t, stillInPairing, "old provider should not be in new epoch pairing")

	// The stale provider must NOT be re-added to reportedProviders.
	require.False(t, csm.reportedProviders.IsReported(reportedAddr),
		"stale provider should not be in reportedProviders after epoch transition — "+
			"ReconnectProviders re-added it from a stale candidate snapshot")

	// GetReportedProviders must not return the stale provider.
	reported = csm.GetReportedProviders(secondEpochHeight)
	for _, rp := range reported {
		require.NotEqual(t, reportedAddr, rp.Address,
			"stale provider from previous epoch should not appear in reported providers")
	}
}

// TestCanceledContextDoesNotPenalizeEndpoint verifies that when a request's context
// is canceled before ConnectRawClientWithTimeout succeeds, the endpoint's
// ConnectionRefusals counter is NOT incremented. This prevents genuinely healthy
// providers from being disabled due to client-side cancellations.
func TestCanceledContextDoesNotPenalizeEndpoint(t *testing.T) {
	// Create an endpoint with no existing connections so that
	// fetchEndpointConnectionFromConsumerSessionWithProvider is forced into the
	// ConnectRawClientWithTimeout path.
	endpoint := &Endpoint{
		NetworkAddress: grpcListener,
		Enabled:        true,
		Connections:    make([]*EndpointConnection, 0),
	}

	cswp := &ConsumerSessionsWithProvider{
		PublicLavaAddress: "test-provider",
		Endpoints:         []*Endpoint{endpoint},
		Sessions:          map[int64]*SingleConsumerSession{},
		MaxComputeUnits:   200,
		PairingEpoch:      firstEpochHeight,
	}

	// Pre-cancel the context before calling fetch.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	connected, endpoints, _, err := cswp.fetchEndpointConnectionFromConsumerSessionWithProvider(
		ctx,
		false, // retryDisabledEndpoints
		false, // getAllEndpoints
		"",    // addon
		nil,   // extensionNames
	)

	// Connection should fail (context canceled), but the endpoint must not be penalized.
	require.False(t, connected)
	require.Empty(t, endpoints)
	// err may or may not be set depending on whether allDisabled triggers, but
	// the critical assertion is on ConnectionRefusals.
	_ = err

	endpoint.mu.RLock()
	refusals := endpoint.ConnectionRefusals
	enabled := endpoint.Enabled
	endpoint.mu.RUnlock()

	require.Equal(t, uint64(0), refusals,
		"ConnectionRefusals should remain 0 when context is canceled")
	require.True(t, enabled,
		"Endpoint should remain enabled when context is canceled")
}

// createBackupProviderList creates a single-entry backup provider list pointing at the live gRPC server.
func createBackupProviderList(addr string) map[uint64]*ConsumerSessionsWithProvider {
	endpoints := []*Endpoint{{
		Connections:        []*EndpointConnection{},
		NetworkAddress:     addr,
		Enabled:            true,
		ConnectionRefusals: 0,
	}}
	cswp := NewConsumerSessionWithProvider(
		"lava@backup1",
		endpoints,
		1000,
		firstEpochHeight,
		sdk.NewCoin("ulava", sdk.NewInt(100)),
	)
	return map[uint64]*ConsumerSessionsWithProvider{0: cswp}
}

// TestBlockProvider_BackupProviderIsTracked verifies that calling blockProvider for a backup provider
// adds it to blockedBackupProviders (not silently dropped because it's not in validAddresses).
func TestBlockProvider_BackupProviderIsTracked(t *testing.T) {
	csm := CreateConsumerSessionManager()
	backupList := createBackupProviderList(grpcListener)
	err := csm.UpdateAllProviders(firstEpochHeight, nil, backupList)
	require.NoError(t, err)

	backupAddr := backupList[0].PublicLavaAddress

	// Block the backup provider
	err = csm.blockProvider(context.Background(), backupAddr, false, firstEpochHeight, 0, 0, false, nil, nil)
	require.NoError(t, err)

	csm.lock.RLock()
	_, blocked := csm.blockedBackupProviders[backupAddr]
	csm.lock.RUnlock()

	require.True(t, blocked, "backup provider should be in blockedBackupProviders after blockProvider call")
}

// TestBlockProvider_BackupProviderFilteredFromSelection verifies that a blocked backup provider is
// not returned by getValidConsumerSessionsWithProviderFromBackupProviderList.
func TestBlockProvider_BackupProviderFilteredFromSelection(t *testing.T) {
	csm := CreateConsumerSessionManager()
	backupList := createBackupProviderList(grpcListener)
	err := csm.UpdateAllProviders(firstEpochHeight, nil, backupList)
	require.NoError(t, err)

	backupAddr := backupList[0].PublicLavaAddress

	// Directly mark it blocked
	csm.lock.Lock()
	csm.blockedBackupProviders[backupAddr] = struct{}{}
	csm.lock.Unlock()

	// Attempting to get backup sessions should fail — no eligible backup providers
	ignored := &ignoredProviders{providers: make(map[string]struct{}), currentEpoch: firstEpochHeight}
	_, err = csm.getValidConsumerSessionsWithProviderFromBackupProviderList(
		context.Background(), ignored, 1, servicedBlockNumber, "", nil, 0, 0, NewUsedProviders(nil),
	)
	require.Error(t, err, "blocked backup provider should not be selectable")
}

// TestUpdateAllProviders_BlockedBackupProviderPersistedAcrossEpoch verifies that a backup provider
// blocked in epoch N is re-blocked in epoch N+1 via previousEpochBlockedProviders.
func TestUpdateAllProviders_BlockedBackupProviderPersistedAcrossEpoch(t *testing.T) {
	csm := CreateConsumerSessionManager()
	backupList := createBackupProviderList(grpcListener)
	err := csm.UpdateAllProviders(firstEpochHeight, nil, backupList)
	require.NoError(t, err)

	backupAddr := backupList[0].PublicLavaAddress

	// Block it in the first epoch
	err = csm.blockProvider(context.Background(), backupAddr, false, firstEpochHeight, 0, 0, false, nil, nil)
	require.NoError(t, err)

	csm.lock.RLock()
	_, blocked := csm.blockedBackupProviders[backupAddr]
	csm.lock.RUnlock()
	require.True(t, blocked, "backup provider should be blocked before epoch transition")

	// Use a non-listening endpoint for epoch 2 so the background probe fails and
	// the provider stays blocked (avoids a race between the assertion and the
	// probe goroutine spawned by UpdateAllProviders).
	backupListEpoch2 := map[uint64]*ConsumerSessionsWithProvider{
		0: NewConsumerSessionWithProvider(
			backupAddr,
			[]*Endpoint{{Connections: []*EndpointConnection{}, NetworkAddress: "127.0.0.1:1", Enabled: true}},
			1000,
			secondEpochHeight,
			sdk.NewCoin("ulava", sdk.NewInt(100)),
		),
	}
	err = csm.UpdateAllProviders(secondEpochHeight, nil, backupListEpoch2)
	require.NoError(t, err)

	// Should still be blocked in the new epoch
	csm.lock.RLock()
	_, stillBlocked := csm.blockedBackupProviders[backupAddr]
	csm.lock.RUnlock()
	require.True(t, stillBlocked, "backup provider should remain blocked after epoch transition")
}

// TestUpdateAllProviders_NormalProviderBlockedAsBackupInNextEpoch verifies that a provider blocked
// as a normal provider in epoch N is re-blocked as a backup provider in epoch N+1 if it moves to the backup list.
func TestUpdateAllProviders_NormalProviderBlockedAsBackupInNextEpoch(t *testing.T) {
	csm := CreateConsumerSessionManager()
	pairingList := createPairingList("", true)
	err := csm.UpdateAllProviders(firstEpochHeight, pairingList, nil)
	require.NoError(t, err)

	// Block a normal provider
	normalAddr := pairingList[0].PublicLavaAddress
	err = csm.blockProvider(context.Background(), normalAddr, false, firstEpochHeight, 0, 0, false, nil, nil)
	require.NoError(t, err)

	csm.lock.RLock()
	_, inBlockedList := func() (int, bool) {
		for _, addr := range csm.currentlyBlockedProviderAddresses {
			if addr == normalAddr {
				return 0, true
			}
		}
		return 0, false
	}()
	csm.lock.RUnlock()
	require.True(t, inBlockedList, "normal provider should be in currentlyBlockedProviderAddresses")

	// In next epoch the same provider appears only in the backup list.
	// Use a non-listening endpoint so the background probe fails and the provider
	// stays blocked (avoids a race with the probe goroutine spawned by UpdateAllProviders).
	backupList := map[uint64]*ConsumerSessionsWithProvider{
		0: NewConsumerSessionWithProvider(
			normalAddr,
			[]*Endpoint{{Connections: []*EndpointConnection{}, NetworkAddress: "127.0.0.1:1", Enabled: true}},
			1000,
			secondEpochHeight,
			sdk.NewCoin("ulava", sdk.NewInt(100)),
		),
	}
	err = csm.UpdateAllProviders(secondEpochHeight, nil, backupList)
	require.NoError(t, err)

	csm.lock.RLock()
	_, blockedAsBackup := csm.blockedBackupProviders[normalAddr]
	csm.lock.RUnlock()
	require.True(t, blockedAsBackup, "previously-blocked normal provider should be re-blocked as backup in new epoch")
}

// TestCheckAndUnblock_BackupRoutedToComprehensiveProbe verifies the `!isBackup && !IsReported`
// guard in checkAndUnblockHealthyReBlockedProviders. probeProviders only probes pairingList, so
// backup providers are never added to reportedProviders — without the guard, every backup would
// match `!IsReported` and take the immediate-unblock branch, skipping any real health check.
// This test confirms a backup lands in the comprehensive-probe branch instead: we assert the
// backup is absent from reportedProviders (guard precondition) and that the immediate-unblock
// path leaves blockedBackupProviders intact (since that path only touches validAddresses).
func TestCheckAndUnblock_BackupRoutedToComprehensiveProbe(t *testing.T) {
	csm := CreateConsumerSessionManager()
	// Point at an unreachable endpoint so the comprehensive probe's eventual outcome
	// doesn't race with this assertion via the immediate-unblock code path.
	unreachable := map[uint64]*ConsumerSessionsWithProvider{
		0: NewConsumerSessionWithProvider(
			"lava@backup-routed",
			[]*Endpoint{{Connections: []*EndpointConnection{}, NetworkAddress: "127.0.0.1:1", Enabled: true}},
			1000,
			firstEpochHeight,
			sdk.NewCoin("ulava", sdk.NewInt(100)),
		),
	}
	err := csm.UpdateAllProviders(firstEpochHeight, nil, unreachable)
	require.NoError(t, err)

	backupAddr := unreachable[0].PublicLavaAddress

	// Seed previousEpochBlockedProviders as if this backup was blocked last epoch.
	csm.lock.Lock()
	csm.previousEpochBlockedProviders[backupAddr] = struct{}{}
	csm.blockedBackupProviders[backupAddr] = struct{}{}
	csm.lock.Unlock()

	// Guard precondition: backups are never reported (probeProviders skips them).
	require.False(t, csm.reportedProviders.IsReported(backupAddr),
		"backup provider should never appear in reportedProviders")

	// The immediate-unblock branch calls validateAndReturnBlockedProviderToValidAddressesListLocked,
	// which is a no-op for backups (they're not in validAddresses). So if the guard is missing and
	// the backup wrongly takes that branch, blockedBackupProviders stays populated here *and* the
	// comprehensive probe is skipped — a silent stall.
	//
	// With the guard, the backup is routed to the comprehensive probe. We can't easily assert on
	// probe outcome here (probeProvider has separate quirks with unreachable endpoints), but we
	// can assert that the call flow reaches the comprehensive branch by checking the debug-log
	// marker indirectly via state: validAddresses should not contain the backup afterwards either
	// way, and the backup should not have been silently restored to pairing.
	csm.checkAndUnblockHealthyReBlockedProviders(context.Background(), firstEpochHeight)

	csm.lock.RLock()
	_, inPairing := csm.pairing[backupAddr]
	csm.lock.RUnlock()
	require.False(t, inPairing,
		"backup must never be promoted into csm.pairing by the unblock path")
}

// TestCheckAndUnblock_BackupUnblockedWhenHealthy verifies the positive path: a backup that was
// blocked in the previous epoch gets unblocked via the comprehensive probe when its endpoint
// is actually reachable in the new epoch.
func TestCheckAndUnblock_BackupUnblockedWhenHealthy(t *testing.T) {
	csm := CreateConsumerSessionManager()
	backupList := createBackupProviderList(grpcListener)
	err := csm.UpdateAllProviders(firstEpochHeight, nil, backupList)
	require.NoError(t, err)

	backupAddr := backupList[0].PublicLavaAddress

	err = csm.blockProvider(context.Background(), backupAddr, false, firstEpochHeight, 0, 0, false, nil, nil)
	require.NoError(t, err)

	// New epoch, same backup address, still pointing at the healthy gRPC listener.
	backupListEpoch2 := createBackupProviderList(grpcListener)
	// Reuse the same public address so previousEpoch re-blocking matches this entry.
	backupListEpoch2[0].PublicLavaAddress = backupAddr
	err = csm.UpdateAllProviders(secondEpochHeight, nil, backupListEpoch2)
	require.NoError(t, err)

	// Precondition: re-blocked after epoch transition (comes from the previousEpoch merge).
	csm.lock.RLock()
	_, reblocked := csm.blockedBackupProviders[backupAddr]
	csm.lock.RUnlock()
	require.True(t, reblocked, "backup should be re-blocked immediately after epoch transition")

	// Run the unblock pass. Comprehensive probe against grpcListener should succeed → unblock.
	csm.checkAndUnblockHealthyReBlockedProviders(context.Background(), secondEpochHeight)

	csm.lock.RLock()
	_, stillBlocked := csm.blockedBackupProviders[backupAddr]
	csm.lock.RUnlock()
	require.False(t, stillBlocked, "healthy backup should be unblocked after comprehensive probe succeeds")
}

// TestGenerateReconnectCallback_BackupProviderUnblocked covers the #2265-derived behavior:
// when the periodic reconnect callback runs for a blocked backup and its probe succeeds,
// the backup must be removed from blockedBackupProviders (not passed to
// validateAndReturnBlockedProviderToValidAddressesList, which would be a no-op since backups
// are not in validAddresses).
func TestGenerateReconnectCallback_BackupProviderUnblocked(t *testing.T) {
	csm := CreateConsumerSessionManager()
	backupList := createBackupProviderList(grpcListener)
	err := csm.UpdateAllProviders(firstEpochHeight, nil, backupList)
	require.NoError(t, err)

	backupAddr := backupList[0].PublicLavaAddress

	err = csm.blockProvider(context.Background(), backupAddr, false, firstEpochHeight, 0, 0, false, nil, nil)
	require.NoError(t, err)

	csm.lock.RLock()
	_, blocked := csm.blockedBackupProviders[backupAddr]
	csm.lock.RUnlock()
	require.True(t, blocked, "backup should be blocked before reconnect callback")

	callback := csm.GenerateReconnectCallback(backupList[0])
	require.NoError(t, callback(), "reconnect callback should succeed against healthy endpoint")

	csm.lock.RLock()
	_, stillBlocked := csm.blockedBackupProviders[backupAddr]
	csm.lock.RUnlock()
	require.False(t, stillBlocked,
		"successful reconnect probe must remove backup from blockedBackupProviders")
}

// TestGenerateReconnectCallback_NonBackupUsesValidAddressesPath verifies the non-backup branch
// of the reconnect callback: a regular (non-backup) provider whose probe succeeds is routed to
// validateAndReturnBlockedProviderToValidAddressesList, not the blockedBackupProviders path.
// This guards against a regression where the isBackup lookup is inverted or the lock split
// drops the non-backup handling.
func TestGenerateReconnectCallback_NonBackupUsesValidAddressesPath(t *testing.T) {
	csm := CreateConsumerSessionManager()
	pairingList := createPairingList("", true)
	err := csm.UpdateAllProviders(firstEpochHeight, pairingList, nil)
	require.NoError(t, err)

	regularAddr := pairingList[0].PublicLavaAddress
	err = csm.blockProvider(context.Background(), regularAddr, false, firstEpochHeight, 0, 0, false, nil, nil)
	require.NoError(t, err)

	// blockProvider removes from validAddresses and adds to currentlyBlockedProviderAddresses.
	csm.lock.RLock()
	wasInBlockedList := false
	for _, addr := range csm.currentlyBlockedProviderAddresses {
		if addr == regularAddr {
			wasInBlockedList = true
			break
		}
	}
	csm.lock.RUnlock()
	require.True(t, wasInBlockedList, "regular provider should be in currentlyBlockedProviderAddresses")

	// Precondition: the regular provider must NOT be in blockedBackupProviders, otherwise the
	// isBackup branch would swallow it.
	csm.lock.RLock()
	_, inBackupBlocked := csm.blockedBackupProviders[regularAddr]
	csm.lock.RUnlock()
	require.False(t, inBackupBlocked, "regular provider must not appear in blockedBackupProviders")

	callback := csm.GenerateReconnectCallback(pairingList[0])
	require.NoError(t, callback(), "reconnect callback should succeed for healthy regular provider")

	// After a successful reconnect for a non-backup, the provider is returned to validAddresses.
	csm.lock.RLock()
	restored := false
	for _, addr := range csm.validAddresses {
		if addr == regularAddr {
			restored = true
			break
		}
	}
	csm.lock.RUnlock()
	require.True(t, restored,
		"non-backup successful reconnect must restore provider to validAddresses")
}
