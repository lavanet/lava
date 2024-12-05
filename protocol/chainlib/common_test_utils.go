package chainlib

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/lavanet/lava/v4/utils"
	"github.com/lavanet/lava/v4/utils/rand"
	"github.com/lavanet/lava/v4/utils/sigs"

	"github.com/cometbft/cometbft/proto/tendermint/types"
	"github.com/cosmos/cosmos-sdk/client/grpc/tmservice"
	"github.com/cosmos/cosmos-sdk/server/grpc/gogoreflection"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/lavanet/lava/v4/protocol/chaintracker"
	"github.com/lavanet/lava/v4/protocol/common"
	"github.com/lavanet/lava/v4/protocol/lavasession"
	testcommon "github.com/lavanet/lava/v4/testutil/common"
	keepertest "github.com/lavanet/lava/v4/testutil/keeper"
	specutils "github.com/lavanet/lava/v4/utils/keeper"
	plantypes "github.com/lavanet/lava/v4/x/plans/types"
	spectypes "github.com/lavanet/lava/v4/x/spec/types"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

type mockResponseWriter struct {
	blockToReturn *int
}

func (mockResponseWriter) Header() http.Header {
	return http.Header{}
}

func (mockResponseWriter) Write(in []byte) (int, error) {
	return 0, nil
}

func (mrw mockResponseWriter) WriteHeader(statusCode int) {
	*mrw.blockToReturn = statusCode
}

type myServiceImplementation struct {
	*tmservice.UnimplementedServiceServer
	serverCallback http.HandlerFunc
}

func (bbb myServiceImplementation) GetLatestBlock(ctx context.Context, reqIn *tmservice.GetLatestBlockRequest) (*tmservice.GetLatestBlockResponse, error) {
	metadata, exists := metadata.FromIncomingContext(ctx)
	req := &http.Request{}
	if exists {
		headers := map[string][]string{}
		for key, val := range metadata {
			headers[key] = val
		}
		req = &http.Request{
			Header: headers,
		}
	}
	num := 5
	respWriter := mockResponseWriter{blockToReturn: &num}
	bbb.serverCallback(respWriter, req)
	return &tmservice.GetLatestBlockResponse{Block: &types.Block{Header: types.Header{Height: int64(num)}}}, nil
}

func generateCombinations(arr []string) [][]string {
	if len(arr) == 0 {
		return [][]string{{}}
	}

	first, rest := arr[0], arr[1:]
	combinationsWithoutFirst := generateCombinations(rest)
	var combinationsWithFirst [][]string

	for _, c := range combinationsWithoutFirst {
		combinationsWithFirst = append(combinationsWithFirst, append(c, first))
	}

	return append(combinationsWithoutFirst, combinationsWithFirst...)
}

func genericWebSocketHandler() http.HandlerFunc {
	upGrader := websocket.Upgrader{}

	// Create a simple websocket server that mocks the node
	return func(w http.ResponseWriter, r *http.Request) {
		conn, err := upGrader.Upgrade(w, r, nil)
		if err != nil {
			fmt.Println(err)
			panic("got error in upgrader")
		}
		defer conn.Close()

		for {
			// Read the request
			messageType, message, err := conn.ReadMessage()
			if err != nil {
				panic("got error in ReadMessage")
			}
			fmt.Println("got ws message", string(message), messageType)
			conn.WriteMessage(messageType, message)
			fmt.Println("writing ws message", string(message), messageType)
		}
	}
}

// generates a chain parser, a chain fetcher messages based on it
// apiInterface can either be an ApiInterface string as in spectypes.ApiInterfaceXXX or a number for an index in the apiCollections
func CreateChainLibMocks(
	ctx context.Context,
	specIndex string,
	apiInterface string,
	httpServerCallback http.HandlerFunc,
	wsServerCallback http.HandlerFunc,
	getToTopMostPath string,
	services []string,
) (cpar ChainParser, crout ChainRouter, cfetc chaintracker.ChainFetcher, closeServer func(), endpointRet *lavasession.RPCProviderEndpoint, errRet error) {
	utils.SetGlobalLoggingLevel("debug")
	closeServer = nil
	spec, err := specutils.GetASpec(specIndex, getToTopMostPath, nil, nil)
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}
	index, err := strconv.Atoi(apiInterface)
	if err == nil && index < len(spec.ApiCollections) {
		apiInterface = spec.ApiCollections[index].CollectionData.ApiInterface
	}
	chainParser, err := NewChainParser(apiInterface)
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}
	var chainRouter ChainRouter
	chainParser.SetSpec(spec)
	endpoint := &lavasession.RPCProviderEndpoint{
		NetworkAddress: lavasession.NetworkAddressData{},
		ChainID:        specIndex,
		ApiInterface:   apiInterface,
		Geolocation:    1,
		NodeUrls:       []common.NodeUrl{},
	}
	addons, extensions, err := chainParser.SeparateAddonsExtensions(services)
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}

	if httpServerCallback == nil {
		httpServerCallback = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
	}

	if wsServerCallback == nil {
		wsServerCallback = genericWebSocketHandler()
	}

	if apiInterface == spectypes.APIInterfaceGrpc {
		// Start a new gRPC server using the buffered connection
		grpcServer := grpc.NewServer()
		lis, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			return nil, nil, nil, closeServer, nil, err
		}
		endpoint.NodeUrls = append(endpoint.NodeUrls, common.NodeUrl{Url: lis.Addr().String(), Addons: addons})
		allCombinations := generateCombinations(extensions)
		for _, extensionsList := range allCombinations {
			endpoint.NodeUrls = append(endpoint.NodeUrls, common.NodeUrl{Url: lis.Addr().String(), Addons: append(addons, extensionsList...)})
		}
		go func() {
			service := myServiceImplementation{serverCallback: httpServerCallback}
			tmservice.RegisterServiceServer(grpcServer, service)
			gogoreflection.Register(grpcServer)
			// Serve requests on the buffered connection
			if err := grpcServer.Serve(lis); err != nil {
				return
			}
		}()
		time.Sleep(10 * time.Millisecond)
		chainRouter, err = GetChainRouter(ctx, 1, endpoint, chainParser)
		if err != nil {
			return nil, nil, nil, closeServer, nil, err
		}
	} else {
		var mockWebSocketServer *httptest.Server
		var wsUrl string
		mockWebSocketServer = httptest.NewServer(wsServerCallback)
		wsUrl = "ws" + strings.TrimPrefix(mockWebSocketServer.URL, "http")
		mockHttpServer := httptest.NewServer(httpServerCallback)
		closeServer = func() {
			mockHttpServer.Close()
			mockWebSocketServer.Close()
		}
		endpoint.NodeUrls = append(endpoint.NodeUrls, common.NodeUrl{Url: mockHttpServer.URL, Addons: addons})
		if len(extensions) > 0 {
			endpoint.NodeUrls = append(endpoint.NodeUrls, common.NodeUrl{Url: mockHttpServer.URL, Addons: extensions})
			endpoint.NodeUrls = append(endpoint.NodeUrls, common.NodeUrl{Url: wsUrl, Addons: extensions})
		}
		endpoint.NodeUrls = append(endpoint.NodeUrls, common.NodeUrl{Url: wsUrl, Addons: nil})
		chainRouter, err = GetChainRouter(ctx, 1, endpoint, chainParser)
		if err != nil {
			return nil, nil, nil, closeServer, nil, err
		}
	}
	chainFetcher := NewChainFetcher(ctx, &ChainFetcherOptions{chainRouter, chainParser, endpoint, nil})
	return chainParser, chainRouter, chainFetcher, closeServer, endpoint, err
}

type TestStruct struct {
	Ctx       context.Context
	Keepers   *keepertest.Keepers
	Servers   *keepertest.Servers
	Providers []sigs.Account
	Spec      spectypes.Spec
	Plan      plantypes.Plan
	Consumer  sigs.Account
	Validator sigs.Account
}

func (ts *TestStruct) BondDenom() string {
	return ts.Keepers.StakingKeeper.BondDenom(sdk.UnwrapSDKContext(ts.Ctx))
}

func SetupForTests(t *testing.T, numOfProviders int, specID string, getToTopMostPath string) TestStruct {
	rand.InitRandomSeed()
	ts := TestStruct{}
	ts.Servers, ts.Keepers, ts.Ctx = keepertest.InitAllKeepers(t)

	// init keepers state
	var balance int64 = 100000000000

	ts.Validator = testcommon.CreateNewAccount(ts.Ctx, *ts.Keepers, balance)
	msg, err := stakingtypes.NewMsgCreateValidator(
		sdk.ValAddress(ts.Validator.Addr),
		ts.Validator.PubKey,
		sdk.NewCoin(ts.BondDenom(), sdk.NewIntFromUint64(uint64(balance))),
		stakingtypes.Description{},
		stakingtypes.NewCommissionRates(sdk.NewDecWithPrec(1, 1), sdk.NewDecWithPrec(1, 1), sdk.NewDecWithPrec(1, 1)),
		sdk.ZeroInt(),
	)
	require.NoError(t, err)
	_, err = ts.Servers.StakingServer.CreateValidator(ts.Ctx, msg)
	require.NoError(t, err)
	// setup consumer
	ts.Consumer = testcommon.CreateNewAccount(ts.Ctx, *ts.Keepers, balance)

	// setup providers
	for i := 0; i < numOfProviders; i++ {
		ts.Providers = append(ts.Providers, testcommon.CreateNewAccount(ts.Ctx, *ts.Keepers, balance))
	}
	sdkContext := sdk.UnwrapSDKContext(ts.Ctx)
	spec, err := specutils.GetASpec(specID, getToTopMostPath, &sdkContext, &ts.Keepers.Spec)
	if err != nil {
		require.NoError(t, err)
	}
	ts.Spec = spec
	ts.Keepers.Spec.SetSpec(sdk.UnwrapSDKContext(ts.Ctx), ts.Spec)

	ts.Plan = testcommon.CreateMockPlan()
	ts.Keepers.Plans.AddPlan(sdk.UnwrapSDKContext(ts.Ctx), ts.Plan, false)

	var stake int64 = 50000000000

	// subscribe consumer
	testcommon.BuySubscription(ts.Ctx, *ts.Keepers, *ts.Servers, ts.Consumer, ts.Plan.Index)

	// stake providers
	for _, provider := range ts.Providers {
		testcommon.StakeAccount(t, ts.Ctx, *ts.Keepers, *ts.Servers, provider, ts.Spec, stake, ts.Validator)
	}

	// advance for the staking to be valid
	ts.Ctx = keepertest.AdvanceEpoch(ts.Ctx, ts.Keepers)
	return ts
}
