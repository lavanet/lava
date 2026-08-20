package rpcprovider

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/lavanet/lava/v5/protocol/lavasession"
	pairingtypes "github.com/lavanet/lava/v5/x/pairing/types"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/encoding/gzip"
	healthgrpc "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/stats"
)

const (
	testChainID      = "LAV1"
	testApiInterface = "tendermintrpc"
)

// testRelayReceiver is a RelayReceiver whose Relay reply is controlled by the test.
type testRelayReceiver struct {
	replyData []byte
}

func (r *testRelayReceiver) Relay(ctx context.Context, request *pairingtypes.RelayRequest) (*pairingtypes.RelayReply, error) {
	return &pairingtypes.RelayReply{Data: r.replyData}, nil
}

func (r *testRelayReceiver) RelaySubscribe(request *pairingtypes.RelayRequest, srv pairingtypes.Relayer_RelaySubscribeServer) error {
	return nil
}

func (r *testRelayReceiver) Probe(ctx context.Context, probeReq *pairingtypes.ProbeRequest) (*pairingtypes.ProbeReply, error) {
	return &pairingtypes.ProbeReply{Guid: probeReq.Guid}, nil
}

func freeAddress(t *testing.T) string {
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	addr := lis.Addr().String()
	require.NoError(t, lis.Close())
	return addr
}

// startTestListener brings up a ProviderListener with a single registered receiver and
// waits until it accepts connections.
func startTestListener(t *testing.T, disableTLS bool, replyData []byte) (*ProviderListener, string) {
	t.Helper()
	addr := freeAddress(t)
	pl := NewProviderListener(context.Background(), lavasession.NetworkAddressData{Address: addr, DisableTLS: disableTLS}, HealthCheckURLPathFlagDefault)
	err := pl.RegisterReceiver(&testRelayReceiver{replyData: replyData}, &lavasession.RPCProviderEndpoint{
		NetworkAddress: lavasession.NetworkAddressData{Address: addr, DisableTLS: disableTLS},
		ChainID:        testChainID,
		ApiInterface:   testApiInterface,
	})
	require.NoError(t, err)
	waitForListener(t, addr)
	t.Cleanup(func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		pl.Shutdown(shutdownCtx)
	})
	return pl, addr
}

func waitForListener(t *testing.T, addr string) {
	t.Helper()
	require.Eventually(t, func() bool {
		conn, err := net.DialTimeout("tcp", addr, 100*time.Millisecond)
		if err != nil {
			return false
		}
		conn.Close()
		return true
	}, 5*time.Second, 10*time.Millisecond, "listener did not come up on %s", addr)
}

func dialProvider(t *testing.T, addr string, disableTLS bool, extraOpts ...grpc.DialOption) *grpc.ClientConn {
	t.Helper()
	opts := []grpc.DialOption{grpc.WithBlock()}
	if disableTLS {
		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	} else {
		opts = append(opts, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{InsecureSkipVerify: true})))
	}
	opts = append(opts, extraOpts...)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	conn, err := grpc.DialContext(ctx, addr, opts...)
	require.NoError(t, err)
	t.Cleanup(func() { conn.Close() })
	return conn
}

func healthCheckURL(addr string, disableTLS bool) string {
	scheme := "https"
	if disableTLS {
		scheme = "http"
	}
	return fmt.Sprintf("%s://%s%s", scheme, addr, HealthCheckURLPathFlagDefault)
}

func httpProbeClient() *http.Client {
	return &http.Client{
		Timeout:   10 * time.Second,
		Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}},
	}
}

func relayRequest() *pairingtypes.RelayRequest {
	return &pairingtypes.RelayRequest{
		RelaySession: &pairingtypes.RelaySession{SpecId: testChainID},
		RelayData:    &pairingtypes.RelayPrivateData{ApiInterface: testApiInterface},
	}
}

// TestProviderListenerRelayAndHealthCheck asserts that native gRPC and the HTTP health
// probe are both served on the same port, with and without TLS. The TLS case is the
// regression guard for ALPN: a TLS listener that does not advertise "h2" is rejected by
// every native gRPC client.
func TestProviderListenerRelayAndHealthCheck(t *testing.T) {
	for _, disableTLS := range []bool{true, false} {
		name := "TLS"
		if disableTLS {
			name = "plaintext"
		}
		t.Run(name, func(t *testing.T) {
			_, addr := startTestListener(t, disableTLS, []byte("relay reply"))
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			conn := dialProvider(t, addr, disableTLS)
			relayer := pairingtypes.NewRelayerClient(conn)

			reply, err := relayer.Relay(ctx, relayRequest())
			require.NoError(t, err)
			require.Equal(t, []byte("relay reply"), reply.Data)

			probeReply, err := relayer.Probe(ctx, &pairingtypes.ProbeRequest{Guid: 42, SpecId: testChainID, ApiInterface: testApiInterface})
			require.NoError(t, err)
			require.Equal(t, uint64(42), probeReply.Guid)

			// the gRPC health checking protocol, what native clients and load balancers probe with
			healthReply, err := healthgrpc.NewHealthClient(conn).Check(ctx, &healthgrpc.HealthCheckRequest{Service: relayerServiceName})
			require.NoError(t, err)
			require.Equal(t, healthgrpc.HealthCheckResponse_SERVING, healthReply.Status)

			// the HTTP health endpoint, what kubernetes httpGet probes use
			resp, err := httpProbeClient().Get(healthCheckURL(addr, disableTLS))
			require.NoError(t, err)
			defer resp.Body.Close()
			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err)
			require.Equal(t, http.StatusOK, resp.StatusCode)
			require.Equal(t, "Healthy", string(body))
		})
	}
}

// TestProviderListenerHealthCheckPaths verifies the health handler only answers GETs on
// the configured path.
func TestProviderListenerHealthCheckPaths(t *testing.T) {
	_, addr := startTestListener(t, true, []byte("relay reply"))
	client := httpProbeClient()

	resp, err := client.Get(fmt.Sprintf("http://%s/not-the-health-path", addr))
	require.NoError(t, err)
	resp.Body.Close()
	require.Equal(t, http.StatusNotFound, resp.StatusCode)

	resp, err = client.Post(healthCheckURL(addr, true), "text/plain", strings.NewReader(""))
	require.NoError(t, err)
	resp.Body.Close()
	require.Equal(t, http.StatusMethodNotAllowed, resp.StatusCode)
}

// TestProviderListenerUnregisteredReceiver asserts relays for endpoints this listener does
// not serve are rejected rather than silently handled.
func TestProviderListenerUnregisteredReceiver(t *testing.T) {
	_, addr := startTestListener(t, true, []byte("relay reply"))
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	relayer := pairingtypes.NewRelayerClient(dialProvider(t, addr, true))
	request := relayRequest()
	request.RelaySession.SpecId = "NOTREGISTERED"
	_, err := relayer.Relay(ctx, request)
	require.Error(t, err)
}

func TestProviderListenerDoubleReceiverRegistration(t *testing.T) {
	pl, addr := startTestListener(t, true, []byte("relay reply"))
	err := pl.RegisterReceiver(&testRelayReceiver{}, &lavasession.RPCProviderEndpoint{
		NetworkAddress: lavasession.NetworkAddressData{Address: addr, DisableTLS: true},
		ChainID:        testChainID,
		ApiInterface:   testApiInterface,
	})
	require.Error(t, err)
}

// TestProviderListenerShutdown asserts a shut down listener reports NOT_SERVING to gRPC
// health clients and frees the port, so a restarting provider can bind it again.
func TestProviderListenerShutdown(t *testing.T) {
	pl, addr := startTestListener(t, true, []byte("relay reply"))
	conn := dialProvider(t, addr, true)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_, err := pairingtypes.NewRelayerClient(conn).Relay(ctx, relayRequest())
	require.NoError(t, err)

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	require.NoError(t, pl.Shutdown(shutdownCtx))

	// in flight aware clients are told to stop using this provider
	healthReply, err := healthgrpc.NewHealthClient(conn).Check(ctx, &healthgrpc.HealthCheckRequest{})
	if err == nil {
		require.Equal(t, healthgrpc.HealthCheckResponse_NOT_SERVING, healthReply.Status)
	}

	// the port is released
	require.Eventually(t, func() bool {
		lis, err := net.Listen("tcp", addr)
		if err != nil {
			return false
		}
		lis.Close()
		return true
	}, 5*time.Second, 20*time.Millisecond, "port %s was not released by Shutdown", addr)

	// a second Shutdown is a no-op rather than a panic on a closed listener
	require.NoError(t, pl.Shutdown(shutdownCtx))
}

// byteCountingStatsHandler records the payload sizes gRPC actually put on the wire.
type byteCountingStatsHandler struct {
	stats.Handler
	mu           sync.Mutex
	wireBytesIn  int
	uncompressed int
}

func (h *byteCountingStatsHandler) TagRPC(ctx context.Context, info *stats.RPCTagInfo) context.Context {
	return ctx
}

func (h *byteCountingStatsHandler) HandleRPC(ctx context.Context, rpcStats stats.RPCStats) {
	inPayload, ok := rpcStats.(*stats.InPayload)
	if !ok {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	h.wireBytesIn += inPayload.WireLength
	h.uncompressed += inPayload.Length
}

func (h *byteCountingStatsHandler) TagConn(ctx context.Context, info *stats.ConnTagInfo) context.Context {
	return ctx
}
func (h *byteCountingStatsHandler) HandleConn(ctx context.Context, connStats stats.ConnStats) {}

func (h *byteCountingStatsHandler) totals() (wire, uncompressed int) {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.wireBytesIn, h.uncompressed
}

// TestProviderListenerGRPCCompression asserts the provider serves relays compressed by
// gRPC itself when the consumer dials with a compressor, which is what the
// enable-grpc-compression flag turns on, and uncompressed otherwise.
func TestProviderListenerGRPCCompression(t *testing.T) {
	// a highly compressible reply, well over a single gRPC message frame
	replyData := []byte(strings.Repeat(`{"jsonrpc":"2.0","id":1,"result":"0x1234567890abcdef"},`, 5000))
	_, addr := startTestListener(t, true, replyData)

	for _, compressed := range []bool{false, true} {
		name := "uncompressed"
		opts := []grpc.DialOption{}
		if compressed {
			name = "compressed"
			opts = append(opts, grpc.WithDefaultCallOptions(grpc.UseCompressor(gzip.Name)))
		}

		t.Run(name, func(t *testing.T) {
			statsHandler := &byteCountingStatsHandler{}
			opts := append(opts, grpc.WithStatsHandler(statsHandler))
			conn := dialProvider(t, addr, true, opts...)

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			reply, err := pairingtypes.NewRelayerClient(conn).Relay(ctx, relayRequest())
			require.NoError(t, err)
			// the payload the caller sees is always the original one
			require.Equal(t, replyData, reply.Data)

			wire, uncompressed := statsHandler.totals()
			require.NotZero(t, uncompressed)
			if compressed {
				// the provider echoed the consumer's compressor back on the response
				require.Less(t, wire, uncompressed/2, "response should have been gzip compressed on the wire")
			} else {
				// the wire length is the payload plus the 5 byte gRPC message header
				require.GreaterOrEqual(t, wire, uncompressed, "response should not have been compressed")
			}
		})
	}
}
