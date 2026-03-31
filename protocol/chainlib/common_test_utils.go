package chainlib

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/reflection"

	"github.com/lavanet/lava/v5/protocol/chaintracker"
	"github.com/lavanet/lava/v5/protocol/common"
	"github.com/lavanet/lava/v5/protocol/lavasession"
	specutils "github.com/lavanet/lava/v5/utils/keeper"
	spectypes "github.com/lavanet/lava/v5/x/spec/types"
	"github.com/lavanet/lava/v5/utils"
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

// mockTMServiceServer provides a minimal mock of the Tendermint service for gRPC testing.
// This replaces the cosmos-sdk tmservice dependency.
type mockTMServiceServer struct {
	serverCallback http.HandlerFunc
}

// mockGetLatestBlockRequest/Response are minimal types for the mock gRPC service.
type mockGetLatestBlockRequest struct{}
type mockGetLatestBlockResponse struct {
	Height int64
}

func (s mockTMServiceServer) GetLatestBlock(ctx context.Context, _ *mockGetLatestBlockRequest) (*mockGetLatestBlockResponse, error) {
	md, exists := metadata.FromIncomingContext(ctx)
	req := &http.Request{}
	if exists {
		headers := map[string][]string{}
		for key, val := range md {
			headers[key] = val
		}
		req = &http.Request{Header: headers}
	}
	num := 5
	respWriter := mockResponseWriter{blockToReturn: &num}
	s.serverCallback(respWriter, req)
	return &mockGetLatestBlockResponse{Height: int64(num)}, nil
}

// mockTMServiceDesc is the gRPC service descriptor for the mock TM service.
var mockTMServiceDesc = grpc.ServiceDesc{
	ServiceName: "cosmos.base.tendermint.v1beta1.Service",
	HandlerType: (*mockTMServiceInterface)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "GetLatestBlock",
			Handler: func(srv any, ctx context.Context, dec func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) {
				in := new(mockGetLatestBlockRequest)
				if err := dec(in); err != nil {
					return nil, err
				}
				return srv.(mockTMServiceInterface).GetLatestBlock(ctx, in)
			},
		},
	},
	Streams: []grpc.StreamDesc{},
}

type mockTMServiceInterface interface {
	GetLatestBlock(context.Context, *mockGetLatestBlockRequest) (*mockGetLatestBlockResponse, error)
}

type myServiceImplementation struct {
	serverCallback http.HandlerFunc
}

func (s myServiceImplementation) GetLatestBlock(ctx context.Context, req *mockGetLatestBlockRequest) (*mockGetLatestBlockResponse, error) {
	return mockTMServiceServer{serverCallback: s.serverCallback}.GetLatestBlock(ctx, req)
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
				// Don't panic here, just return to close the connection gracefully
				return
			}
			fmt.Println("got ws message", string(message), messageType)
			conn.WriteMessage(messageType, message)
			fmt.Println("writing ws message", string(message), messageType)
		}
	}
}

// CreateMockSpec returns a minimal spectypes.Spec suitable for unit tests.
func CreateMockSpec() spectypes.Spec {
	specName := "mockspec"
	spec := spectypes.Spec{}
	spec.Name = specName
	spec.Index = specName
	spec.Enabled = true
	spec.ReliabilityThreshold = 4294967295
	spec.BlockDistanceForFinalizedData = 0
	spec.DataReliabilityEnabled = true
	spec.ApiCollections = []*spectypes.ApiCollection{
		{
			Enabled: true,
			CollectionData: spectypes.CollectionData{
				ApiInterface: "stub",
				Type:         "GET",
			},
			Apis: []*spectypes.Api{
				{
					Name:         specName + "API",
					ComputeUnits: 100,
					Enabled:      true,
				},
			},
		},
	}
	spec.Shares = 1
	return spec
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
	// Create a cancellable context for the connector to prevent leaks
	connectorCtx, cancelConnector := context.WithCancel(ctx)

	closeServer = func() {
		cancelConnector()
	}

	// Load specs from all directories so cross-directory imports (e.g. LAV1 -> COSMOSSDK) resolve.
	spec, err := specutils.GetSpecFromLocalDirs([]string{
		getToTopMostPath + "specs/mainnet-1/specs/",
		getToTopMostPath + "specs/testnet-2/specs/",
	}, specIndex)
	if err != nil {
		cancelConnector()
		return nil, nil, nil, nil, nil, err
	}
	index, err := strconv.Atoi(apiInterface)
	if err == nil && index < len(spec.ApiCollections) {
		apiInterface = spec.ApiCollections[index].CollectionData.ApiInterface
	}
	chainParser, err := NewChainParser(apiInterface)
	if err != nil {
		cancelConnector()
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
	addons, extensions, err := chainParser.SeparateAddonsExtensions(context.Background(), services)
	if err != nil {
		cancelConnector()
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
			cancelConnector()
			return nil, nil, nil, closeServer, nil, err
		}
		endpoint.NodeUrls = append(endpoint.NodeUrls, common.NodeUrl{Url: lis.Addr().String(), Addons: addons})
		allCombinations := generateCombinations(extensions)
		for _, extensionsList := range allCombinations {
			endpoint.NodeUrls = append(endpoint.NodeUrls, common.NodeUrl{Url: lis.Addr().String(), Addons: append(addons, extensionsList...)})
		}
		go func() {
			service := myServiceImplementation{serverCallback: httpServerCallback}
			grpcServer.RegisterService(&mockTMServiceDesc, service)
			reflection.Register(grpcServer)
			// Serve requests on the buffered connection
			if err := grpcServer.Serve(lis); err != nil {
				return
			}
		}()
		time.Sleep(10 * time.Millisecond)
		chainRouter, err = GetChainRouter(connectorCtx, 1, endpoint, chainParser)
		if err != nil {
			grpcServer.Stop()
			cancelConnector()
			return nil, nil, nil, closeServer, nil, err
		}
		closeServer = func() {
			grpcServer.Stop()
			cancelConnector()
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
			cancelConnector()
		}
		endpoint.NodeUrls = append(endpoint.NodeUrls, common.NodeUrl{Url: mockHttpServer.URL, Addons: addons})
		// Only add WebSocket URLs for API interfaces that support WebSocket (JsonRPC and TendermintRPC)
		// REST API only supports HTTP/HTTPS, not WebSocket
		supportsWebSocket := apiInterface == spectypes.APIInterfaceJsonRPC || apiInterface == spectypes.APIInterfaceTendermintRPC
		if len(extensions) > 0 {
			endpoint.NodeUrls = append(endpoint.NodeUrls, common.NodeUrl{Url: mockHttpServer.URL, Addons: extensions})
			if supportsWebSocket {
				endpoint.NodeUrls = append(endpoint.NodeUrls, common.NodeUrl{Url: wsUrl, Addons: extensions})
			}
		}
		if supportsWebSocket {
			endpoint.NodeUrls = append(endpoint.NodeUrls, common.NodeUrl{Url: wsUrl, Addons: nil})
		}
		chainRouter, err = GetChainRouter(connectorCtx, 1, endpoint, chainParser)
		if err != nil {
			mockHttpServer.Close()
			mockWebSocketServer.Close()
			cancelConnector()
			return nil, nil, nil, closeServer, nil, err
		}
	}
	chainFetcher := NewChainFetcher(ctx, &ChainFetcherOptions{chainRouter, chainParser, endpoint, nil})
	return chainParser, chainRouter, chainFetcher, closeServer, endpoint, err
}
