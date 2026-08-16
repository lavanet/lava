package rpcprovider

import (
	"context"
	"crypto/tls"
	"errors"
	"net"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gogo/status"
	"github.com/lavanet/lava/v5/protocol/chainlib"
	"github.com/lavanet/lava/v5/protocol/lavaprotocol/protocolerrors"
	"github.com/lavanet/lava/v5/protocol/lavasession"
	"github.com/lavanet/lava/v5/utils"
	pairingtypes "github.com/lavanet/lava/v5/x/pairing/types"
	"github.com/soheilhy/cmux"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	grpc "google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	_ "google.golang.org/grpc/encoding/gzip" // registers the gzip compressor so relays can be compressed by gRPC itself
	"google.golang.org/grpc/health"
	healthgrpc "google.golang.org/grpc/health/grpc_health_v1"
)

const (
	HealthCheckURLPathFlagName    = "health-check-url-path"
	HealthCheckURLPathFlagDefault = "/lava/health"

	// relayerServiceName is the fully qualified name of the relayer service, as reported
	// over the gRPC health checking protocol.
	relayerServiceName = "lavanet.lava.pairing.Relayer"

	// maxRelayMessageSize is deliberately large to accommodate big debug responses.
	maxRelayMessageSize = 1024 * 1024 * 512 // 512MB

	// connectionMatchTimeout bounds how long a freshly accepted connection may stay silent
	// while we decide whether it speaks gRPC or HTTP/1.1. Both send their first bytes
	// immediately after connecting, so a slower peer is not a client we can serve anyway.
	connectionMatchTimeout = 30 * time.Second
)

type ProviderListener struct {
	networkAddress string
	relayServer    *relayServer
	grpcServer     *grpc.Server
	httpServer     *http.Server
	healthServer   *health.Server
	listener       net.Listener
	connMux        cmux.CMux
	shuttingDown   atomic.Bool
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

// Shutdown stops serving new connections and drains the in flight ones. It reports
// NOT_SERVING to gRPC health clients first, so load balancers can drain the provider
// before the connections are torn down.
func (pl *ProviderListener) Shutdown(shutdownCtx context.Context) error {
	if !pl.shuttingDown.CompareAndSwap(false, true) {
		return nil
	}
	pl.healthServer.Shutdown()

	// stop accepting new connections and release the port, the connections that were
	// already matched to a server are drained below
	if err := pl.listener.Close(); err != nil && !errors.Is(err, net.ErrClosed) {
		utils.LavaFormatWarning("provider listener failed to close", err, utils.Attribute{Key: "address", Value: pl.networkAddress})
	}
	pl.connMux.Close()

	gracefulStopped := make(chan struct{})
	go func() {
		pl.grpcServer.GracefulStop()
		close(gracefulStopped)
	}()

	httpErr := pl.httpServer.Shutdown(shutdownCtx)

	select {
	case <-gracefulStopped:
	case <-shutdownCtx.Done():
		// the deadline passed while relays were still in flight, drop them
		pl.grpcServer.Stop()
		<-gracefulStopped
	}

	if httpErr != nil {
		return utils.LavaFormatError("provider failed to shutdown the health check server", httpErr, utils.Attribute{Key: "address", Value: pl.networkAddress})
	}
	utils.LavaFormatInfo("listener closed server", utils.Attribute{Key: "address", Value: pl.networkAddress})
	return nil
}

// serveError reports whether an error returned by one of the serve loops is worth
// crashing the process over, or is just the expected fallout of Shutdown.
func (pl *ProviderListener) serveError(err error) error {
	if err == nil || pl.shuttingDown.Load() {
		return nil
	}
	if errors.Is(err, http.ErrServerClosed) || errors.Is(err, cmux.ErrListenerClosed) || errors.Is(err, net.ErrClosed) || errors.Is(err, grpc.ErrServerStopped) {
		return nil
	}
	return err
}

func NewProviderListener(ctx context.Context, networkAddress lavasession.NetworkAddressData, healthCheckPath string) *ProviderListener {
	pl := &ProviderListener{networkAddress: networkAddress.Address}

	lis := chainlib.GetListenerWithRetryGrpc("tcp", networkAddress.Address)
	if networkAddress.DisableTLS {
		utils.LavaFormatInfo("Running with disabled TLS configuration")
	} else {
		tlsConfig := lavasession.GetTlsConfig(networkAddress)
		// gRPC clients only offer "h2" and refuse a connection that negotiated no protocol
		// at all, so it must be advertised. "http/1.1" is listed first because ALPN is
		// resolved in server preference order: clients that speak both (health probes,
		// browsers) get HTTP/1.1 and reach the health endpoint below, while gRPC clients
		// still get "h2" since it is the only protocol they offer.
		tlsConfig.NextProtos = []string{"http/1.1", "h2"}
		lis = tls.NewListener(lis, tlsConfig)
	}
	pl.listener = lis

	// gRPC and the HTTP health check share a single port, so connections are routed by
	// the protocol they open with. Matching on the HTTP/2 client preface is a cheap
	// prefix comparison and, unlike matching on the gRPC content-type, it resolves as
	// soon as the connection is established instead of waiting for the first request.
	connMux := cmux.New(lis)
	connMux.SetReadTimeout(connectionMatchTimeout)
	grpcListener := connMux.Match(cmux.HTTP2())
	httpListener := connMux.Match(cmux.Any())
	pl.connMux = connMux

	pl.grpcServer = grpc.NewServer(
		grpc.MaxRecvMsgSize(maxRelayMessageSize),
		grpc.MaxSendMsgSize(maxRelayMessageSize),
		grpc.StatsHandler(otelgrpc.NewServerHandler()),
	)

	relayServer := &relayServer{relayReceivers: map[string]*relayReceiverWrapper{}}
	pl.relayServer = relayServer
	pairingtypes.RegisterRelayerServer(pl.grpcServer, relayServer)

	// the gRPC health checking protocol is what native gRPC clients and load balancers
	// probe with, the HTTP endpoint below is kept for kubernetes style httpGet probes
	pl.healthServer = health.NewServer()
	healthgrpc.RegisterHealthServer(pl.grpcServer, pl.healthServer)
	pl.healthServer.SetServingStatus("", healthgrpc.HealthCheckResponse_SERVING)
	pl.healthServer.SetServingStatus(relayerServiceName, healthgrpc.HealthCheckResponse_SERVING)

	pl.httpServer = &http.Server{
		ReadHeaderTimeout: connectionMatchTimeout,
		Handler: http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
			if req.URL.Path != healthCheckPath {
				resp.WriteHeader(http.StatusNotFound)
				return
			}
			if req.Method != http.MethodGet {
				resp.WriteHeader(http.StatusMethodNotAllowed)
				return
			}
			resp.WriteHeader(http.StatusOK)
			resp.Write([]byte("Healthy"))
		}),
	}

	go func() {
		if err := pl.serveError(pl.grpcServer.Serve(grpcListener)); err != nil {
			utils.LavaFormatFatal("provider failed to serve grpc", err, utils.Attribute{Key: "Address", Value: pl.networkAddress})
		}
	}()

	go func() {
		if err := pl.serveError(pl.httpServer.Serve(httpListener)); err != nil {
			utils.LavaFormatFatal("provider failed to serve health checks", err, utils.Attribute{Key: "Address", Value: pl.networkAddress})
		}
	}()

	go func() {
		utils.LavaFormatInfo("New provider listener active", utils.Attribute{Key: "address", Value: networkAddress})
		if err := pl.serveError(connMux.Serve()); err != nil {
			utils.LavaFormatFatal("provider failed to serve", err, utils.Attribute{Key: "Address", Value: pl.networkAddress})
		}
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
		err := utils.LavaFormatError("got called with unhandled relay receiver", protocolerrors.UnhandledRelayReceiverError, utils.Attribute{Key: "requested_receiver", Value: endpoint.Key()}, utils.Attribute{Key: "handled_receivers", Value: strings.Join(keys, ",")})
		return nil, status.Error(codes.Code(protocolerrors.UnhandledRelayReceiverError.ABCICode()), err.Error())
	}
	if !relayReceiver.enabled {
		err := utils.LavaFormatError("relayReceiver is disabled", protocolerrors.DisabledRelayReceiverError, utils.Attribute{Key: "relayReceiver", Value: endpoint.Key()})
		return nil, status.Error(codes.Code(protocolerrors.DisabledRelayReceiverError.ABCICode()), err.Error())
	}
	return *relayReceiver.relayReceiver, nil
}
