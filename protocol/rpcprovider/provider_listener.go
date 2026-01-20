package rpcprovider

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"strings"
	"sync"

	"github.com/gogo/status"
	"github.com/lavanet/lava/v5/protocol/chainlib"
	"github.com/lavanet/lava/v5/protocol/lavaprotocol/protocolerrors"
	"github.com/lavanet/lava/v5/protocol/lavasession"
	"github.com/lavanet/lava/v5/utils"
	pairingtypes "github.com/lavanet/lava/v5/x/pairing/types"
	"github.com/soheilhy/cmux"
	grpc "google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/encoding/gzip" // Register gzip compressor
	"google.golang.org/grpc/health"
	healthgrpc "google.golang.org/grpc/health/grpc_health_v1"
)

// Ensure gzip compressor is registered
var _ = gzip.Name

const (
	HealthCheckURLPathFlagName    = "health-check-url-path"
	HealthCheckURLPathFlagDefault = "/lava/health"
)

// isShutdownError returns true if the error is expected during graceful shutdown
func isShutdownError(err error) bool {
	if err == nil {
		return false
	}
	if err == http.ErrServerClosed || err == cmux.ErrListenerClosed || err == net.ErrClosed {
		return true
	}
	// Check error message for common shutdown patterns
	errStr := err.Error()
	return strings.Contains(errStr, "use of closed network connection") ||
		strings.Contains(errStr, "server closed") ||
		strings.Contains(errStr, "mux: listener closed")
}

type ProviderListener struct {
	networkAddress string
	relayServer    *relayServer
	grpcServer     *grpc.Server
	httpServer     *http.Server
	healthServer   *health.Server
	cmux           cmux.CMux
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
	// Mark service as healthy when receiver is registered
	serviceName := endpoint.ChainID + "-" + endpoint.ApiInterface
	pl.healthServer.SetServingStatus(serviceName, healthgrpc.HealthCheckResponse_SERVING)
	utils.LavaFormatInfo("[++] Provider Listening on Address", utils.Attribute{Key: "chainID", Value: endpoint.ChainID}, utils.Attribute{Key: "apiInterface", Value: endpoint.ApiInterface}, utils.Attribute{Key: "Address", Value: endpoint.NetworkAddress})
	return nil
}

func (pl *ProviderListener) Shutdown(shutdownCtx context.Context) error {
	pl.healthServer.Shutdown()
	pl.httpServer.Shutdown(shutdownCtx)
	pl.grpcServer.GracefulStop()
	return nil
}

func NewProviderListener(ctx context.Context, networkAddress lavasession.NetworkAddressData, healthCheckPath string) *ProviderListener {
	pl := &ProviderListener{networkAddress: networkAddress.Address}

	lis := chainlib.GetListenerWithRetryGrpc("tcp", networkAddress.Address)

	// Wrap with TLS if enabled
	if !networkAddress.DisableTLS {
		tlsConfig := lavasession.GetTlsConfig(networkAddress)
		lis = tls.NewListener(lis, tlsConfig)
	} else {
		utils.LavaFormatInfo("Running with disabled TLS configuration")
	}

	// Create connection multiplexer to handle both HTTP and gRPC on same port
	mux := cmux.New(lis)
	// Match HTTP/1.1 first for health checks (fast prefix match)
	httpListener := mux.Match(cmux.HTTP1Fast())
	// Everything else goes to gRPC (avoids expensive header parsing)
	grpcListener := mux.Match(cmux.Any())
	pl.cmux = mux

	// Build gRPC server
	opts := []grpc.ServerOption{
		grpc.MaxRecvMsgSize(1024 * 1024 * 512), // 512MB for large debug responses
		grpc.MaxSendMsgSize(1024 * 1024 * 512), // 512MB for large debug responses
	}
	grpcServer := grpc.NewServer(opts...)
	pl.grpcServer = grpcServer

	// Register gRPC health checking service
	healthServer := health.NewServer()
	healthgrpc.RegisterHealthServer(grpcServer, healthServer)
	healthServer.SetServingStatus("", healthgrpc.HealthCheckResponse_SERVING)
	pl.healthServer = healthServer

	// Register relay server
	relayServer := &relayServer{relayReceivers: map[string]*relayReceiverWrapper{}}
	pl.relayServer = relayServer
	pairingtypes.RegisterRelayerServer(grpcServer, relayServer)

	// Create HTTP server for health checks
	httpMux := http.NewServeMux()
	httpMux.HandleFunc(healthCheckPath, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("Healthy"))
		} else {
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})
	httpServer := &http.Server{Handler: httpMux}
	pl.httpServer = httpServer

	// Start servers
	go func() {
		utils.LavaFormatInfo("New provider listener active", utils.Attribute{Key: "address", Value: networkAddress})
		if err := grpcServer.Serve(grpcListener); err != nil {
			// Ignore expected shutdown errors
			if isShutdownError(err) {
				return
			}
			utils.LavaFormatFatal("gRPC server failed", err, utils.Attribute{Key: "Address", Value: networkAddress.Address})
		}
	}()

	go func() {
		if err := httpServer.Serve(httpListener); err != nil {
			// Ignore expected shutdown errors
			if isShutdownError(err) {
				return
			}
			utils.LavaFormatFatal("HTTP health server failed", err, utils.Attribute{Key: "Address", Value: networkAddress.Address})
		}
	}()

	go func() {
		if err := mux.Serve(); err != nil {
			if !isShutdownError(err) {
				utils.LavaFormatError("cmux serve error", err)
			}
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
		err := utils.LavaFormatError("got called with unhandled relay receiver", protocolerrors.UnhandledRelayReceiverError, utils.Attribute{Key: "requested_receiver", Value: endpoint.Key()}, utils.Attribute{Key: "handled_receivers", Value: strings.Join(keys, ",")})
		return nil, status.Error(codes.Code(protocolerrors.UnhandledRelayReceiverError.ABCICode()), err.Error())
	}
	if !relayReceiver.enabled {
		err := utils.LavaFormatError("relayReceiver is disabled", protocolerrors.DisabledRelayReceiverError, utils.Attribute{Key: "relayReceiver", Value: endpoint.Key()})
		return nil, status.Error(codes.Code(protocolerrors.DisabledRelayReceiverError.ABCICode()), err.Error())
	}
	return *relayReceiver.relayReceiver, nil
}
