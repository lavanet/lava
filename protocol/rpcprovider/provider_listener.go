package rpcprovider

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"

	"github.com/gogo/status"
	"github.com/improbable-eng/grpc-web/go/grpcweb"
	"github.com/lavanet/lava/protocol/chainlib"
	"github.com/lavanet/lava/protocol/common"
	"github.com/lavanet/lava/protocol/lavaprotocol"
	"github.com/lavanet/lava/protocol/lavasession"
	"github.com/lavanet/lava/utils"
	pairingtypes "github.com/lavanet/lava/x/pairing/types"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	grpc "google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

const (
	HealthCheckURLPathFlagName    = "health-check-url-path"
	HealthCheckURLPathFlagDefault = "/lava/health"
)

type ProviderListener struct {
	networkAddress string
	relayServer    *relayServer
	httpServer     http.Server
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
		return utils.LavaFormatError("double_receiver_setup receiver already defined on this address with the same chainID and apiInterface", nil, utils.Attribute{Key: "chainID", Value: endpoint.ChainID}, utils.Attribute{Key: "apiInterface", Value: endpoint.ApiInterface})
	}
	pl.relayServer.relayReceivers[listen_endpoint.Key()] = &relayReceiverWrapper{relayReceiver: &existingReceiver, enabled: true}
	utils.LavaFormatInfo("[++] Provider Listening on Address", utils.Attribute{Key: "chainID", Value: endpoint.ChainID}, utils.Attribute{Key: "apiInterface", Value: endpoint.ApiInterface}, utils.Attribute{Key: "Address", Value: endpoint.NetworkAddress})
	return nil
}

func (pl *ProviderListener) Shutdown(shutdownCtx context.Context) error {
	if err := pl.httpServer.Shutdown(shutdownCtx); err != nil {
		utils.LavaFormatFatal("Provider failed to shutdown", err)
	}
	return nil
}

func NewProviderListener(ctx context.Context, networkAddress lavasession.NetworkAddressData, healthCheckPath string) *ProviderListener {
	pl := &ProviderListener{networkAddress: networkAddress.Address}

	// GRPC
	lis := chainlib.GetListenerWithRetryGrpc("tcp", networkAddress.Address)
	opts := []grpc.ServerOption{
		grpc.MaxRecvMsgSize(1024 * 1024 * 32), // setting receive size to 32mb instead of 4mb default
	}
	grpcServer := grpc.NewServer(opts...)

	wrappedServer := grpcweb.WrapServer(grpcServer)
	handler := func(resp http.ResponseWriter, req *http.Request) {
		// Set CORS headers
		resp.Header().Set("Access-Control-Allow-Origin", "*")
		resp.Header().Set("Access-Control-Allow-Headers", fmt.Sprintf("Content-Type, x-grpc-web, lava-sdk-relay-timeout, %s", common.LAVA_CONSUMER_PROCESS_GUID))

		if req.URL.Path == healthCheckPath && req.Method == http.MethodGet {
			resp.WriteHeader(http.StatusOK)
			resp.Write([]byte("Healthy"))
			return
		}

		wrappedServer.ServeHTTP(resp, req)
	}

	pl.httpServer = http.Server{
		Handler: h2c.NewHandler(http.HandlerFunc(handler), &http2.Server{}),
	}

	var serveExecutor func() error
	if networkAddress.DisableTLS {
		utils.LavaFormatInfo("Running with disabled TLS configuration")
		serveExecutor = func() error {
			return pl.httpServer.Serve(lis)
		}
	} else {
		pl.httpServer.TLSConfig = lavasession.GetTlsConfig(networkAddress)
		serveExecutor = func() error {
			return pl.httpServer.ServeTLS(lis, "", "")
		}
	}

	relayServer := &relayServer{relayReceivers: map[string]*relayReceiverWrapper{}}
	pl.relayServer = relayServer
	pairingtypes.RegisterRelayerServer(grpcServer, relayServer)
	go func() {
		utils.LavaFormatInfo("New provider listener active", utils.Attribute{Key: "address", Value: networkAddress})
		if err := serveExecutor(); !errors.Is(err, http.ErrServerClosed) {
			utils.LavaFormatFatal("provider failed to serve", err, utils.Attribute{Key: "Address", Value: lis.Addr().String()})
		}
		utils.LavaFormatInfo("listener closed server", utils.Attribute{Key: "address", Value: networkAddress})
	}()
	return pl
}

type relayReceiverWrapper struct {
	relayReceiver *RelayReceiver
	enabled       bool
}

type relayServer struct {
	pairingtypes.UnimplementedRelayerServer
	relayReceivers map[string]*relayReceiverWrapper
	lock           sync.RWMutex
}

type RelayReceiver interface {
	Relay(ctx context.Context, request *pairingtypes.RelayRequest) (*pairingtypes.RelayReply, error)
	RelaySubscribe(request *pairingtypes.RelayRequest, srv pairingtypes.Relayer_RelaySubscribeServer) error
	Probe(ctx context.Context, probeReq *pairingtypes.ProbeRequest) (*pairingtypes.ProbeReply, error)
}

func (rs *relayServer) Relay(ctx context.Context, request *pairingtypes.RelayRequest) (*pairingtypes.RelayReply, error) {
	if request.RelayData == nil || request.RelaySession == nil {
		return nil, utils.LavaFormatError("invalid relay request, internal fields are nil", nil)
	}
	relayReceiver, err := rs.findReceiver(request.RelayData.ApiInterface, request.RelaySession.SpecId)
	if err != nil {
		return nil, err
	}
	return relayReceiver.Relay(ctx, request)
}

func (rs *relayServer) Probe(ctx context.Context, probeReq *pairingtypes.ProbeRequest) (*pairingtypes.ProbeReply, error) {
	relayReceiver, err := rs.findReceiver(probeReq.ApiInterface, probeReq.SpecId)
	if err != nil {
		return nil, err
	}
	return relayReceiver.Probe(ctx, probeReq)
}

func (rs *relayServer) RelaySubscribe(request *pairingtypes.RelayRequest, srv pairingtypes.Relayer_RelaySubscribeServer) error {
	relayReceiver, err := rs.findReceiver(request.RelayData.ApiInterface, request.RelaySession.SpecId)
	if err != nil {
		return err
	}
	return relayReceiver.RelaySubscribe(request, srv)
}

func (rs *relayServer) findReceiver(apiInterface string, specID string) (RelayReceiver, error) {
	endpoint := lavasession.RPCEndpoint{ChainID: specID, ApiInterface: apiInterface}
	rs.lock.RLock()
	defer rs.lock.RUnlock()
	relayReceiver, ok := rs.relayReceivers[endpoint.Key()]
	if !ok {
		keys := make([]string, 0, len(rs.relayReceivers))
		for k := range rs.relayReceivers {
			keys = append(keys, k)
		}
		err := utils.LavaFormatError("got called with unhandled relay receiver", lavaprotocol.UnhandledRelayReceiverError, utils.Attribute{Key: "requested_receiver", Value: endpoint.Key()}, utils.Attribute{Key: "handled_receivers", Value: strings.Join(keys, ",")})
		return nil, status.Error(codes.Code(lavaprotocol.UnhandledRelayReceiverError.ABCICode()), err.Error())
	}
	if !relayReceiver.enabled {
		err := utils.LavaFormatError("relayReceiver is disabled", lavaprotocol.DisabledRelayReceiverError, utils.Attribute{Key: "relayReceiver", Value: endpoint.Key()})
		return nil, status.Error(codes.Code(lavaprotocol.DisabledRelayReceiverError.ABCICode()), err.Error())
	}
	return *relayReceiver.relayReceiver, nil
}
