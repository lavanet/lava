package chainlib

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/cosmos/cosmos-sdk/client/grpc/tmservice"
	"github.com/cosmos/cosmos-sdk/server/grpc/gogoreflection"
	"github.com/lavanet/lava/protocol/chaintracker"
	"github.com/lavanet/lava/protocol/common"
	"github.com/lavanet/lava/protocol/lavasession"
	keepertest "github.com/lavanet/lava/testutil/keeper"
	spectypes "github.com/lavanet/lava/x/spec/types"
	"github.com/tendermint/tendermint/proto/tendermint/types"
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

// generates a chain parser, a chain fetcher messages based on it
func CreateChainLibMocks(ctx context.Context, specIndex string, apiInterface string, serverCallback http.HandlerFunc, getToTopMostPath string) (cpar ChainParser, cprox ChainProxy, cfetc chaintracker.ChainFetcher, closeServer func(), errRet error) {
	closeServer = nil
	lavaSpec, err := keepertest.GetASpec(specIndex, getToTopMostPath, nil, nil)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	chainParser, err := NewChainParser(apiInterface)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	var chainProxy ChainProxy
	chainParser.SetSpec(lavaSpec)
	endpoint := &lavasession.RPCProviderEndpoint{
		NetworkAddress: lavasession.NetworkAddressData{},
		ChainID:        specIndex,
		ApiInterface:   apiInterface,
		Geolocation:    1,
		NodeUrls:       []common.NodeUrl{},
	}
	if apiInterface == spectypes.APIInterfaceGrpc {
		// Start a new gRPC server using the buffered connection
		grpcServer := grpc.NewServer()
		lis, err := net.Listen("tcp", "localhost:0")
		if err != nil {
			return nil, nil, nil, closeServer, err
		}
		endpoint.NodeUrls = append(endpoint.NodeUrls, common.NodeUrl{Url: lis.Addr().String()})
		go func() {
			service := myServiceImplementation{serverCallback: serverCallback}
			tmservice.RegisterServiceServer(grpcServer, service)
			gogoreflection.Register(grpcServer)
			// Serve requests on the buffered connection
			if err := grpcServer.Serve(lis); err != nil {
				return
			}
		}()
		time.Sleep(10 * time.Millisecond)
		chainProxy, err = GetChainProxy(ctx, 1, endpoint, chainParser)
		if err != nil {
			return nil, nil, nil, closeServer, err
		}
	} else {
		mockServer := httptest.NewServer(serverCallback)
		closeServer = mockServer.Close
		endpoint.NodeUrls = append(endpoint.NodeUrls, common.NodeUrl{Url: mockServer.URL})
		chainProxy, err = GetChainProxy(ctx, 1, endpoint, chainParser)
		if err != nil {
			return nil, nil, nil, closeServer, err
		}
	}

	chainFetcher := NewChainFetcher(ctx, chainProxy, chainParser, endpoint)

	return chainParser, chainProxy, chainFetcher, closeServer, err
}
