package lavasession

import (
	"context"
	"errors"
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
	"github.com/lavanet/lava/utils/lavaslices"
	"github.com/lavanet/lava/utils/rand"
	pairingtypes "github.com/lavanet/lava/x/pairing/types"
	spectypes "github.com/lavanet/lava/x/spec/types"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
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
	err := csm.UpdateAllProviders(firstEpochHeight, pairingList) // update the providers.
	require.NoError(t, err)
	css, err := csm.GetSessions(ctx, cuForFirstRequest, NewUsedProviders(nil), servicedBlockNumber, "", nil, common.NO_STATE, 0) // get a session
	require.NoError(t, err)

	for _, cs := range css {
		require.NotNil(t, cs)
		require.Equal(t, cs.Epoch, csm.currentEpoch)
		require.Equal(t, cs.Session.LatestRelayCu, cuForFirstRequest)
		err = csm.OnSessionDone(cs.Session, servicedBlockNumber, cuForFirstRequest, time.Millisecond, cs.Session.CalculateExpectedLatency(2*time.Millisecond), (servicedBlockNumber - 1), numberOfProviders, numberOfProviders, false)
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
	err = csm.UpdateAllProviders(firstEpochHeight, pairingList) // update the providers.
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
	baseLatency := common.AverageWorldLatency / 2 // we want performance to be half our timeout or better
	return NewConsumerSessionManager(&RPCEndpoint{"stub", "stub", "stub", false, "/", 0}, provideroptimizer.NewProviderOptimizer(provideroptimizer.STRATEGY_BALANCED, 0, baseLatency, 1), nil, nil, "lava@test", NewActiveSubscriptionProvidersStorage())
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
	err := csm.UpdateAllProviders(firstEpochHeight, pairingList) // update the providers.
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
	css, err := csm.GetSessions(ctx, cuForFirstRequest, usedProviders, servicedBlockNumber, "", nil, common.NO_STATE, 0) // get a session
	require.NoError(t, err)
	_, expectedProviderAddress := css[csm.validAddresses[0]]
	require.True(t, expectedProviderAddress)

	css2, err := csm.GetSessions(ctx, cuForFirstRequest, usedProviders, servicedBlockNumber, "", nil, common.NO_STATE, 0) // get a session
	require.NoError(t, err)
	_, expectedProviderAddress2 := css2[highestProviderCu]
	require.True(t, expectedProviderAddress2)

	runOnSessionDoneForConsumerSessionMap(t, css, csm)
	runOnSessionDoneForConsumerSessionMap(t, css2, csm)
	time.Sleep(time.Second)
	require.Equal(t, len(csm.validAddresses), 2)

	css3, err := csm.GetSessions(ctx, cuForFirstRequest, usedProviders, servicedBlockNumber, "", nil, common.NO_STATE, 0) // get a session
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
	err := csm.UpdateAllProviders(firstEpochHeight, map[uint64]*ConsumerSessionsWithProvider{0: pairingList[0], 1: pairingList[1]}) // create two providers
	require.NoError(t, err)
	timeLimit := time.Second * 30
	loopStartTime := time.Now()
	for {
		usedProviders := NewUsedProviders(map[string]string{"lava-providers-block": pairingList[1].PublicLavaAddress})
		css, err := csm.GetSessions(ctx, cuForFirstRequest, usedProviders, servicedBlockNumber, "", nil, common.NO_STATE, 0) // get a session
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
	css, err := csm.GetSessions(ctx, cuForFirstRequest, usedProviders, servicedBlockNumber, "", nil, common.NO_STATE, 0) // get a session
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
		usedProviders := NewUsedProviders(map[string]string{"lava-providers-block": pairingList[1].PublicLavaAddress})
		require.Equal(t, BlockedProviderSessionUnusedStatus, csm.pairing[pairingList[0].PublicLavaAddress].blockedAndUsedWithChanceForRecoveryStatus)
		css, err := csm.GetSessions(ctx, cuForFirstRequest, usedProviders, servicedBlockNumber, "", nil, common.NO_STATE, 0) // get a session
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
		err := csm.OnSessionDone(cs.Session, servicedBlockNumber, cuForFirstRequest, time.Millisecond, cs.Session.CalculateExpectedLatency(2*time.Millisecond), (servicedBlockNumber - 1), numberOfProviders, numberOfProviders, false)
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
	err := csm.UpdateAllProviders(firstEpochHeight, pairingList) // update the providers.
	require.NoError(t, err)
	css, err := csm.GetSessions(ctx, maxCuForVirtualEpoch*(virtualEpoch+1), NewUsedProviders(nil), servicedBlockNumber, "", nil, common.NO_STATE, virtualEpoch) // get a session
	require.NoError(t, err)

	for _, cs := range css {
		require.NotNil(t, cs)
		require.Equal(t, cs.Epoch, csm.currentEpoch)
		require.Equal(t, cs.Session.LatestRelayCu, maxCuForVirtualEpoch*(virtualEpoch+1))
		err = csm.OnSessionDone(cs.Session, servicedBlockNumber, maxCuForVirtualEpoch*(virtualEpoch+1), time.Millisecond, cs.Session.CalculateExpectedLatency(2*time.Millisecond), (servicedBlockNumber - 1), numberOfProviders, numberOfProviders, false)
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
	err := csm.UpdateAllProviders(firstEpochHeight, pairingList) // update the providers.
	require.NoError(t, err)

	_, err = csm.GetSessions(ctx, maxCuForVirtualEpoch*(virtualEpoch+1)+10, NewUsedProviders(nil), servicedBlockNumber, "", nil, common.NO_STATE, virtualEpoch) // get a session
	require.Error(t, err)
}

func TestPairingReset(t *testing.T) {
	ctx := context.Background()
	csm := CreateConsumerSessionManager()
	pairingList := createPairingList("", true)
	err := csm.UpdateAllProviders(firstEpochHeight, pairingList) // update the providers.
	require.NoError(t, err)
	csm.validAddresses = []string{}                                                                                              // set valid addresses to zero
	css, err := csm.GetSessions(ctx, cuForFirstRequest, NewUsedProviders(nil), servicedBlockNumber, "", nil, common.NO_STATE, 0) // get a session
	require.NoError(t, err)
	require.Equal(t, len(csm.validAddresses), len(csm.pairingAddresses))

	for _, cs := range css {
		require.NotNil(t, cs)
		require.Equal(t, cs.Epoch, csm.currentEpoch)
		require.Equal(t, cs.Session.LatestRelayCu, cuForFirstRequest)
		err = csm.OnSessionDone(cs.Session, servicedBlockNumber, cuForFirstRequest, time.Millisecond, cs.Session.CalculateExpectedLatency(2*time.Millisecond), (servicedBlockNumber - 1), numberOfProviders, numberOfProviders, false)
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
	err := csm.UpdateAllProviders(firstEpochHeight, pairingList) // update the providers.
	require.NoError(t, err)
	for {
		utils.LavaFormatDebug(fmt.Sprintf("%v", len(csm.validAddresses)))
		if len(csm.validAddresses) == 0 { // wait for all pairings to be blocked.
			break
		}
		css, err := csm.GetSessions(ctx, cuForFirstRequest, NewUsedProviders(nil), servicedBlockNumber, "", nil, common.NO_STATE, 0) // get a session
		require.NoError(t, err)

		for _, cs := range css {
			err = csm.OnSessionFailure(cs.Session, nil)
			require.NoError(t, err) // fail test.
		}
	}
	require.Equal(t, len(csm.validAddresses), 0)
	css, err := csm.GetSessions(ctx, cuForFirstRequest, NewUsedProviders(nil), servicedBlockNumber, "", nil, common.NO_STATE, 0) // get a session
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
	err := csm.UpdateAllProviders(firstEpochHeight, pairingList) // update the providers.
	require.NoError(t, err)

	for numberOfResets := 0; numberOfResets < numberOfResetsToTest; numberOfResets++ {
		for {
			utils.LavaFormatDebug(fmt.Sprintf("%v", len(csm.validAddresses)))
			if len(csm.validAddresses) == 0 { // wait for all pairings to be blocked.
				break
			}
			css, err := csm.GetSessions(ctx, cuForFirstRequest, NewUsedProviders(nil), servicedBlockNumber, "", nil, common.NO_STATE, 0) // get a session
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
		css, err := csm.GetSessions(ctx, cuForFirstRequest, NewUsedProviders(nil), servicedBlockNumber, "", nil, common.NO_STATE, 0) // get a session
		require.NoError(t, err)
		require.Equal(t, len(csm.validAddresses), len(csm.pairingAddresses))

		for _, cs := range css {
			require.NotNil(t, cs)
			require.Equal(t, cs.Epoch, csm.currentEpoch)
			require.Equal(t, cs.Session.LatestRelayCu, cuForFirstRequest)
			require.Equal(t, csm.numberOfResets, uint64(numberOfResets+1)) // verify we had one reset only
		}
	}

	css, err := csm.GetSessions(ctx, cuForFirstRequest, NewUsedProviders(nil), servicedBlockNumber, "", nil, common.NO_STATE, 0) // get a session
	require.NoError(t, err)

	for _, cs := range css {
		require.NotNil(t, cs)
		require.Equal(t, cs.Epoch, csm.currentEpoch)
		require.Equal(t, cs.Session.LatestRelayCu, cuForFirstRequest)
		err = csm.OnSessionDone(cs.Session, servicedBlockNumber, cuForFirstRequest, time.Millisecond, cs.Session.CalculateExpectedLatency(2*time.Millisecond), (servicedBlockNumber - 1), numberOfProviders, numberOfProviders, false)
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
	err := csm.UpdateAllProviders(firstEpochHeight, pairingList) // update the providers.
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
		css, err := csm.GetSessions(ctx, cuForFirstRequest, NewUsedProviders(nil), servicedBlockNumber, "", nil, common.NO_STATE, 0) // get a session
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
			err = csm.OnSessionDone(cs, servicedBlockNumber, cuForFirstRequest, time.Millisecond, cs.CalculateExpectedLatency(2*time.Millisecond), (servicedBlockNumber - 1), numberOfProviders, numberOfProviders, false)
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
		css, err := csm.GetSessions(ctx, cuForFirstRequest, NewUsedProviders(nil), servicedBlockNumber, "", nil, common.NO_STATE, 0) // get a session
		require.NoError(t, err)

		for _, cs := range css { // get a session
			require.NotNil(t, cs)
			require.Equal(t, cs.Epoch, csm.currentEpoch)
			require.Equal(t, cs.Session.LatestRelayCu, cuForFirstRequest)
		}
	}

	err = csm.UpdateAllProviders(secondEpochHeight, createPairingList("test2", true)) // update the providers. with half of them
	require.NoError(t, err)

	for j := numberOfAllowedSessionsPerConsumer / 2; j < numberOfAllowedSessionsPerConsumer; j++ {
		cs := sessionList[j].cs
		if rand.Intn(2) > 0 {
			err = csm.OnSessionDone(cs, servicedBlockNumber, cuForFirstRequest, time.Millisecond, cs.CalculateExpectedLatency(2*time.Millisecond), (servicedBlockNumber - 1), numberOfProviders, numberOfProviders, false)
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
	css, err := csm.GetSessions(ctx, cuForFirstRequest, NewUsedProviders(nil), servicedBlockNumber, "", nil, common.NO_STATE, 0) // get a session
	require.NoError(t, err)

	for _, cs := range css {
		require.NotNil(t, cs)
		time.Sleep(time.Duration((rand.Intn(500) + 1)) * time.Millisecond)
		err = csm.OnSessionDone(cs.Session, servicedBlockNumber, cuForFirstRequest, time.Millisecond, cs.Session.CalculateExpectedLatency(2*time.Millisecond), (servicedBlockNumber - 1), numberOfProviders, numberOfProviders, false)
		require.NoError(t, err)
		ch <- p
	}
}

func failedSession(ctx context.Context, csm *ConsumerSessionManager, t *testing.T, p int, ch chan int) {
	css, err := csm.GetSessions(ctx, cuForFirstRequest, NewUsedProviders(nil), servicedBlockNumber, "", nil, common.NO_STATE, 0) // get a session
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
	err := csm.UpdateAllProviders(firstEpochHeight, pairingList) // update the providers.
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
	err := csm.UpdateAllProviders(firstEpochHeight, pairingList) // update the providers.
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
				err := csm.UpdateAllProviders(secondEpochHeight, createPairingList("test2", true)) // update the providers. with half of them
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
	err := csm.UpdateAllProviders(firstEpochHeight, pairingList) // update the providers.
	require.NoError(t, err)
	css, err := csm.GetSessions(ctx, cuForFirstRequest, NewUsedProviders(nil), servicedBlockNumber, "", nil, common.NO_STATE, 0) // get a session
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
	err := csm.UpdateAllProviders(firstEpochHeight, pairingList) // update the providers.
	require.NoError(t, err)
	css, err := csm.GetSessions(ctx, cuForFirstRequest, NewUsedProviders(nil), servicedBlockNumber, "", nil, common.NO_STATE, 0) // get a session
	require.NoError(t, err)

	for _, cs := range css {
		require.NotNil(t, cs)
		require.Equal(t, cs.Epoch, csm.currentEpoch)
		require.Equal(t, cs.Session.LatestRelayCu, cuForFirstRequest)

		err = csm.UpdateAllProviders(secondEpochHeight, pairingList) // update the providers again.
		require.NoError(t, err)
		err = csm.OnSessionFailure(cs.Session, ReportAndBlockProviderError)
		require.NoError(t, err)
	}
}

func TestAllProvidersEndpointsDisabled(t *testing.T) {
	ctx := context.Background()
	csm := CreateConsumerSessionManager()
	pairingList := createPairingList("", false)
	err := csm.UpdateAllProviders(firstEpochHeight, pairingList) // update the providers.
	require.NoError(t, err)
	cs, err := csm.GetSessions(ctx, cuForFirstRequest, NewUsedProviders(nil), servicedBlockNumber, "", nil, common.NO_STATE, 0) // get a session
	require.Nil(t, cs)
	require.Error(t, err)
}

func TestUpdateAllProviders(t *testing.T) {
	csm := CreateConsumerSessionManager()
	pairingList := createPairingList("", true)
	err := csm.UpdateAllProviders(firstEpochHeight, pairingList)
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
	err := csm.UpdateAllProviders(firstEpochHeight, pairingList)
	require.NoError(t, err)
	err = csm.UpdateAllProviders(firstEpochHeight, pairingList)
	require.Error(t, err)
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
	err := csm.UpdateAllProviders(firstEpochHeight, pairingList)
	require.NoError(t, err)
	css, err := csm.GetSessions(ctx, cuForFirstRequest, NewUsedProviders(nil), servicedBlockNumber, "", nil, common.NO_STATE, 0)
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
			err := csm.UpdateAllProviders(firstEpochHeight, pairingList) // update the providers.
			require.NoError(t, err)
			time.Sleep(5 * time.Millisecond) // let probes finish
			utils.LavaFormatDebug("valid providers::::", utils.Attribute{Key: "length", Value: len(csm.getValidAddresses(addon, nil))}, utils.Attribute{Key: "valid addresses", Value: csm.getValidAddresses(addon, nil)}, utils.Attribute{Key: "addon", Value: addon})
			require.NotEqual(t, 0, len(csm.getValidAddresses(addon, nil)), "valid addresses: %#v addonAddresses %#v", csm.getValidAddresses(addon, nil), csm.addonAddresses)
			// block all providers
			initialProvidersLen := len(csm.getValidAddresses(addon, nil))
			for i := 0; i < initialProvidersLen; i++ {
				css, err := csm.GetSessions(ctx, cuForFirstRequest, NewUsedProviders(nil), servicedBlockNumber, addon, nil, common.NO_STATE, 0) // get a session
				require.NoError(t, err, i)
				for _, cs := range css {
					err = csm.OnSessionFailure(cs.Session, ReportAndBlockProviderError)
					require.NoError(t, err)
				}
				utils.LavaFormatDebug("length!", utils.Attribute{Key: "length", Value: len(csm.getValidAddresses(addon, nil))}, utils.Attribute{Key: "valid addresses", Value: csm.getValidAddresses(addon, nil)})
			}
			require.Equal(t, 0, len(csm.getValidAddresses(addon, nil)), csm.validAddresses)
			if addon != "" {
				require.NotEqual(t, csm.getValidAddresses(addon, nil), csm.getValidAddresses("", nil))
			}
			css, err := csm.GetSessions(ctx, cuForFirstRequest, NewUsedProviders(nil), servicedBlockNumber, addon, nil, common.NO_STATE, 0) // get a session
			require.NoError(t, err)
			for _, cs := range css {
				err = csm.OnSessionDone(cs.Session, servicedBlockNumber, cuForFirstRequest, time.Millisecond, cs.Session.CalculateExpectedLatency(2*time.Millisecond), (servicedBlockNumber - 1), numberOfProviders, numberOfProviders, false)
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
			err := csm.UpdateAllProviders(firstEpochHeight, pairingList) // update the providers.
			require.NoError(t, err)
			time.Sleep(5 * time.Millisecond) // let probes finish
			utils.LavaFormatDebug("valid providers::::", utils.Attribute{Key: "length", Value: len(csm.getValidAddresses(extensionOpt.addon, extensionOpt.extensions))}, utils.Attribute{Key: "valid addresses", Value: csm.getValidAddresses(extensionOpt.addon, extensionOpt.extensions)}, utils.Attribute{Key: "extensions", Value: extensionOpt.extensions}, utils.Attribute{Key: "addon", Value: extensionOpt.addon})
			require.NotEqual(t, 0, len(csm.getValidAddresses(extensionOpt.addon, extensionOpt.extensions)), "valid addresses: %#v addonAddresses %#v", csm.getValidAddresses(extensionOpt.addon, extensionOpt.extensions), csm.addonAddresses)
			// block all providers
			extensionsList := []*spectypes.Extension{}
			for _, extension := range extensionOpt.extensions {
				ext := &spectypes.Extension{
					Name: extension,
				}
				extensionsList = append(extensionsList, ext)
			}
			initialProvidersLen := len(csm.getValidAddresses(extensionOpt.addon, extensionOpt.extensions))
			for i := 0; i < initialProvidersLen; i++ {
				css, err := csm.GetSessions(ctx, cuForFirstRequest, NewUsedProviders(nil), servicedBlockNumber, extensionOpt.addon, extensionsList, common.NO_STATE, 0) // get a session
				require.NoError(t, err, i)
				for _, cs := range css {
					err = csm.OnSessionFailure(cs.Session, ReportAndBlockProviderError)
					require.NoError(t, err)
				}
				utils.LavaFormatDebug("length!", utils.Attribute{Key: "length", Value: len(csm.getValidAddresses(extensionOpt.addon, extensionOpt.extensions))}, utils.Attribute{Key: "valid addresses", Value: csm.getValidAddresses(extensionOpt.addon, extensionOpt.extensions)})
			}
			require.Equal(t, 0, len(csm.getValidAddresses(extensionOpt.addon, extensionOpt.extensions)), csm.validAddresses)
			if len(extensionOpt.extensions) > 0 || extensionOpt.addon != "" {
				require.NotEqual(t, csm.getValidAddresses(extensionOpt.addon, extensionOpt.extensions), csm.getValidAddresses("", nil))
			}
			css, err := csm.GetSessions(ctx, cuForFirstRequest, NewUsedProviders(nil), servicedBlockNumber, extensionOpt.addon, extensionsList, common.NO_STATE, 0) // get a session
			require.NoError(t, err)
			for _, cs := range css {
				err = csm.OnSessionDone(cs.Session, servicedBlockNumber, cuForFirstRequest, time.Millisecond, cs.Session.CalculateExpectedLatency(2*time.Millisecond), (servicedBlockNumber - 1), numberOfProviders, numberOfProviders, false)
				require.NoError(t, err)
			}
		})
	}
}

func TestNoPairingsError(t *testing.T) {
	csm := CreateConsumerSessionManager()
	pairingList := createPairingList("", true)
	err := csm.UpdateAllProviders(firstEpochHeight, pairingList) // update the providers.
	require.NoError(t, err)
	time.Sleep(5 * time.Millisecond) // let probes finish
	_, err = csm.getValidProviderAddresses(map[string]struct{}{}, 10, 100, "invalid", nil, common.NO_STATE)
	require.Error(t, err)
	require.True(t, PairingListEmptyError.Is(err))
}

func TestPairingWithStateful(t *testing.T) {
	ctx := context.Background()
	t.Run("stateful", func(t *testing.T) {
		csm := CreateConsumerSessionManager()                        // crates an optimizer returning always 1 provider
		pairingList := createPairingList("", true)                   // creates 10 providers
		err := csm.UpdateAllProviders(firstEpochHeight, pairingList) // update the providers.
		require.NoError(t, err)
		addon := ""
		time.Sleep(5 * time.Millisecond) // let probes finish
		utils.LavaFormatDebug("valid providers::::", utils.Attribute{Key: "length", Value: len(csm.getValidAddresses(addon, nil))}, utils.Attribute{Key: "valid addresses", Value: csm.getValidAddresses(addon, nil)}, utils.Attribute{Key: "addon", Value: addon})
		require.NotEqual(t, 0, len(csm.getValidAddresses(addon, nil)), "valid addresses: %#v addonAddresses %#v", csm.getValidAddresses(addon, nil), csm.addonAddresses)
		providerAddresses := csm.getValidAddresses(addon, nil)
		allProviders := len(providerAddresses)
		require.Equal(t, 10, allProviders)
		css, err := csm.GetSessions(ctx, cuForFirstRequest, NewUsedProviders(nil), servicedBlockNumber, addon, nil, common.CONSISTENCY_SELECT_ALL_PROVIDERS, 0) // get a session
		require.NoError(t, err)
		require.Equal(t, allProviders, len(css))
		for _, cs := range css {
			err = csm.OnSessionDone(cs.Session, servicedBlockNumber, cuForFirstRequest, time.Millisecond, cs.Session.CalculateExpectedLatency(2*time.Millisecond), (servicedBlockNumber - 1), numberOfProviders, numberOfProviders, false)
			require.NoError(t, err)
		}
		usedProviders := NewUsedProviders(nil)
		usedProviders.RemoveUsed(providerAddresses[0], nil)
		css, err = csm.GetSessions(ctx, cuForFirstRequest, usedProviders, servicedBlockNumber, addon, nil, common.CONSISTENCY_SELECT_ALL_PROVIDERS, 0) // get a session
		require.NoError(t, err)
		require.Equal(t, allProviders-1, len(css))
	})
}

func TestMaximumBlockedSessionsErrorsInPairingListEmpty(t *testing.T) {
	ctx := context.Background()
	csm := CreateConsumerSessionManager()
	pairingList := createPairingList("", true)
	err := csm.UpdateAllProviders(firstEpochHeight, map[uint64]*ConsumerSessionsWithProvider{0: pairingList[0]}) // update the providers.
	require.NoError(t, err)
	utils.LavaFormatDebug(fmt.Sprintf("%v", len(csm.validAddresses)))
	for i := 0; i < MaxSessionsAllowedPerProvider; i++ {
		css, err := csm.GetSessions(ctx, cuForFirstRequest, NewUsedProviders(nil), servicedBlockNumber, "", nil, common.NO_STATE, 0) // get a session
		require.NoError(t, err)
		for _, cs := range css {
			err = csm.OnSessionFailure(cs.Session, errors.Join(BlockProviderError, SessionOutOfSyncError))
			require.NoError(t, err)
		}
	}

	_, err = csm.GetSessions(ctx, cuForFirstRequest, NewUsedProviders(nil), servicedBlockNumber, "", nil, common.NO_STATE, 0) // get a session
	require.ErrorIs(t, err, PairingListEmptyError)
}
