package rpcprovider

import (
	"context"
	"os"
	"os/signal"
	"strings"
	"sync"

	"errors"
	"net"
	"net/http"
	"time"

	"github.com/lavanet/lava/protocol/lavasession"

	"github.com/improbable-eng/grpc-web/go/grpcweb"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	"github.com/lavanet/lava/utils"
	pairingtypes "github.com/lavanet/lava/x/pairing/types"
	grpc "google.golang.org/grpc"
)

type ProviderListener struct {
	networkAddress string
	relayServer    *relayServer
}

func (pl *ProviderListener) Key() string {
	return pl.networkAddress
}

func (pl *ProviderListener) RegisterReceiver(existingReceiver RelayReceiver, endpoint *lavasession.RPCProviderEndpoint) error {
	listen_endpoint := lavasession.RPCEndpoint{ChainID: endpoint.ChainID, ApiInterface: endpoint.ApiInterface}
	pl.relayServer.lock.Lock()
	defer pl.relayServer.lock.Unlock()
	_, ok := pl.relayServer.relayReceivers[listen_endpoint.Key()]
	if ok {
		// there was already a receiver defined
		return utils.LavaFormatError("double_receiver_setup receiver already defined on this address with the same chainID and apiInterface", nil, &map[string]string{"chainID": endpoint.ChainID, "apiInterface": endpoint.ApiInterface})
	}
	pl.relayServer.relayReceivers[listen_endpoint.Key()] = existingReceiver
	utils.LavaFormatInfo("Provider Listening on Address", &map[string]string{"chainID": endpoint.ChainID, "apiInterface": endpoint.ApiInterface, "Address": endpoint.NetworkAddress})
	return nil
}

func NewProviderListener(ctx context.Context, networkAddress string) *ProviderListener {
	pl := &ProviderListener{networkAddress: networkAddress}
	ctx, cancel := context.WithCancel(ctx)
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)
	defer func() {
		signal.Stop(signalChan)
		cancel()
	}()
	// GRPC
	lis, err := net.Listen("tcp", networkAddress)
	if err != nil {
		utils.LavaFormatFatal("provider failure setting up listener", err, &map[string]string{"listenAddr": networkAddress})
	}
	grpcServer := grpc.NewServer()

	wrappedServer := grpcweb.WrapServer(grpcServer)
	handler := func(resp http.ResponseWriter, req *http.Request) {
		// Set CORS headers
		resp.Header().Set("Access-Control-Allow-Origin", "*")
		resp.Header().Set("Access-Control-Allow-Headers", "Content-Type,x-grpc-web")

		wrappedServer.ServeHTTP(resp, req)
	}

	httpServer := http.Server{
		Handler: h2c.NewHandler(http.HandlerFunc(handler), &http2.Server{}),
	}

	go func() {
		select {
		case <-ctx.Done():
			utils.LavaFormatInfo("Provider Server ctx.Done", nil)
		case <-signalChan:
			utils.LavaFormatInfo("Provider Server signalChan", nil)
		}

		shutdownCtx, shutdownRelease := context.WithTimeout(context.Background(), 10*time.Second)
		defer shutdownRelease()

		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			utils.LavaFormatFatal("Provider failed to shutdown", err, &map[string]string{})
		}
	}()

	relayServer := &relayServer{relayReceivers: map[string]RelayReceiver{}}
	pl.relayServer = relayServer
	go func() {
		pairingtypes.RegisterRelayerServer(grpcServer, relayServer)

		if err := httpServer.Serve(lis); !errors.Is(err, http.ErrServerClosed) {
			utils.LavaFormatFatal("provider failed to serve", err, &map[string]string{"Address": lis.Addr().String()})
		}

	}()

	return pl
}

type relayServer struct {
	pairingtypes.UnimplementedRelayerServer
	relayReceivers map[string]RelayReceiver
	lock           sync.RWMutex
}

type RelayReceiver interface {
	Relay(ctx context.Context, request *pairingtypes.RelayRequest) (*pairingtypes.RelayReply, error)
	RelaySubscribe(request *pairingtypes.RelayRequest, srv pairingtypes.Relayer_RelaySubscribeServer) error
}

func (rs *relayServer) Relay(ctx context.Context, request *pairingtypes.RelayRequest) (*pairingtypes.RelayReply, error) {
	relayReceiver, err := rs.findReceiver(request)
	if err != nil {
		return nil, err
	}
	return relayReceiver.Relay(ctx, request)
}

func (rs *relayServer) RelaySubscribe(request *pairingtypes.RelayRequest, srv pairingtypes.Relayer_RelaySubscribeServer) error {
	relayReceiver, err := rs.findReceiver(request)
	if err != nil {
		return err
	}
	return relayReceiver.RelaySubscribe(request, srv)
}

func (rs *relayServer) findReceiver(request *pairingtypes.RelayRequest) (RelayReceiver, error) {
	apiInterface := request.ApiInterface
	chainID := request.ChainID
	endpoint := lavasession.RPCEndpoint{ChainID: chainID, ApiInterface: apiInterface}
	rs.lock.RLock()
	defer rs.lock.RUnlock()
	relayReceiver, ok := rs.relayReceivers[endpoint.Key()]
	if !ok {
		keys := make([]string, 0, len(rs.relayReceivers))
		for k := range rs.relayReceivers {
			keys = append(keys, k)
		}
		return nil, utils.LavaFormatError("got called with unhandled relay receiver", nil, &map[string]string{"requested_receiver": endpoint.Key(), "handled_receivers": strings.Join(keys, ",")})
	}
	return relayReceiver, nil
}
