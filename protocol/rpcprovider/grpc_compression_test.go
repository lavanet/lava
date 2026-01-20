package rpcprovider

import (
	"context"
	"fmt"
	"net"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/lavanet/lava/v5/protocol/lavasession"
	pairingtypes "github.com/lavanet/lava/v5/x/pairing/types"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/encoding/gzip"
	"google.golang.org/grpc/stats"
)

// createTestEndpoint creates an RPCProviderEndpoint for testing
func createTestEndpoint(addr string) *lavasession.RPCProviderEndpoint {
	return &lavasession.RPCProviderEndpoint{
		ChainID:      "ETH1",
		ApiInterface: "jsonrpc",
		NetworkAddress: lavasession.NetworkAddressData{
			Address: addr,
		},
	}
}

// compressionStatsHandler tracks gRPC compression statistics
type compressionStatsHandler struct {
	outPayloadCompressedBytes   atomic.Int64
	outPayloadUncompressedBytes atomic.Int64
	inPayloadCompressedBytes    atomic.Int64
	inPayloadUncompressedBytes  atomic.Int64
}

func (h *compressionStatsHandler) TagRPC(ctx context.Context, info *stats.RPCTagInfo) context.Context {
	return ctx
}

func (h *compressionStatsHandler) HandleRPC(ctx context.Context, s stats.RPCStats) {
	switch st := s.(type) {
	case *stats.OutPayload:
		h.outPayloadCompressedBytes.Add(int64(st.CompressedLength))
		h.outPayloadUncompressedBytes.Add(int64(st.Length))
	case *stats.InPayload:
		h.inPayloadCompressedBytes.Add(int64(st.CompressedLength))
		h.inPayloadUncompressedBytes.Add(int64(st.Length))
	}
}

func (h *compressionStatsHandler) TagConn(ctx context.Context, info *stats.ConnTagInfo) context.Context {
	return ctx
}

func (h *compressionStatsHandler) HandleConn(ctx context.Context, s stats.ConnStats) {}

func (h *compressionStatsHandler) GetCompressionRatio() float64 {
	uncompressed := h.inPayloadUncompressedBytes.Load()
	compressed := h.inPayloadCompressedBytes.Load()
	if uncompressed == 0 {
		return 1.0
	}
	return float64(compressed) / float64(uncompressed)
}

func (h *compressionStatsHandler) Reset() {
	h.outPayloadCompressedBytes.Store(0)
	h.outPayloadUncompressedBytes.Store(0)
	h.inPayloadCompressedBytes.Store(0)
	h.inPayloadUncompressedBytes.Store(0)
}

// TestGRPCCompressionEnabled verifies that gRPC compression actually compresses data
func TestGRPCCompressionEnabled(t *testing.T) {
	addr := getAvailablePort(t)
	networkAddr := lavasession.NetworkAddressData{
		Address:    addr,
		DisableTLS: true,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create provider listener
	pl := NewProviderListener(ctx, networkAddr, "/lava/health")
	require.NotNil(t, pl)

	// Large, highly compressible response data (JSON-like repeated pattern)
	largeResponseData := []byte(strings.Repeat(`{"blockNumber":"0x123456","result":"success","data":"`, 1000) +
		strings.Repeat("a]", 5000) + `"}`)

	// Register mock receiver that returns large compressible data
	receiver := &mockRelayReceiver{
		relayFunc: func(ctx context.Context, request *pairingtypes.RelayRequest) (*pairingtypes.RelayReply, error) {
			return &pairingtypes.RelayReply{Data: largeResponseData}, nil
		},
	}
	err := pl.RegisterReceiver(receiver, createTestEndpoint(addr))
	require.NoError(t, err)

	time.Sleep(100 * time.Millisecond)

	// Create stats handler to track compression
	statsHandler := &compressionStatsHandler{}

	// Connect WITH compression enabled
	conn, err := grpc.DialContext(ctx, addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
		grpc.WithDefaultCallOptions(grpc.UseCompressor(gzip.Name)),
		grpc.WithStatsHandler(statsHandler),
	)
	require.NoError(t, err)
	defer conn.Close()

	client := pairingtypes.NewRelayerClient(conn)

	// Make relay request
	reply, err := client.Relay(ctx, &pairingtypes.RelayRequest{
		RelayData: &pairingtypes.RelayPrivateData{
			Data:         []byte("test request"),
			ApiInterface: "jsonrpc",
		},
		RelaySession: &pairingtypes.RelaySession{
			SpecId: "ETH1",
		},
	})
	require.NoError(t, err)
	require.NotNil(t, reply)
	require.Equal(t, largeResponseData, reply.Data)

	// Verify compression occurred - compressed should be smaller than uncompressed
	inCompressed := statsHandler.inPayloadCompressedBytes.Load()
	inUncompressed := statsHandler.inPayloadUncompressedBytes.Load()

	t.Logf("Response size - Compressed: %d bytes, Uncompressed: %d bytes, Ratio: %.2f%%",
		inCompressed, inUncompressed, float64(inCompressed)/float64(inUncompressed)*100)

	// For highly compressible data, we expect significant compression (at least 50% reduction)
	require.Greater(t, inUncompressed, inCompressed,
		"Compressed payload should be smaller than uncompressed")
	require.Less(t, float64(inCompressed)/float64(inUncompressed), 0.5,
		"Expected at least 50%% compression for repetitive JSON data")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	pl.Shutdown(shutdownCtx)
}

// TestGRPCCompressionDisabled verifies that without compression flag, data is not compressed
func TestGRPCCompressionDisabled(t *testing.T) {
	addr := getAvailablePort(t)
	networkAddr := lavasession.NetworkAddressData{
		Address:    addr,
		DisableTLS: true,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pl := NewProviderListener(ctx, networkAddr, "/lava/health")
	require.NotNil(t, pl)

	// Same large response data
	largeResponseData := []byte(strings.Repeat(`{"blockNumber":"0x123456","result":"success"}`, 1000))

	receiver := &mockRelayReceiver{
		relayFunc: func(ctx context.Context, request *pairingtypes.RelayRequest) (*pairingtypes.RelayReply, error) {
			return &pairingtypes.RelayReply{Data: largeResponseData}, nil
		},
	}
	err := pl.RegisterReceiver(receiver, createTestEndpoint(addr))
	require.NoError(t, err)

	time.Sleep(100 * time.Millisecond)

	statsHandler := &compressionStatsHandler{}

	// Connect WITHOUT compression (no grpc.UseCompressor)
	conn, err := grpc.DialContext(ctx, addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
		grpc.WithStatsHandler(statsHandler),
	)
	require.NoError(t, err)
	defer conn.Close()

	client := pairingtypes.NewRelayerClient(conn)

	reply, err := client.Relay(ctx, &pairingtypes.RelayRequest{
		RelayData: &pairingtypes.RelayPrivateData{
			Data:         []byte("test request"),
			ApiInterface: "jsonrpc",
		},
		RelaySession: &pairingtypes.RelaySession{
			SpecId: "ETH1",
		},
	})
	require.NoError(t, err)
	require.NotNil(t, reply)

	// Without compression, compressed == uncompressed (or very close due to protobuf overhead)
	inCompressed := statsHandler.inPayloadCompressedBytes.Load()
	inUncompressed := statsHandler.inPayloadUncompressedBytes.Load()

	t.Logf("Response size (no compression) - Wire: %d bytes, Logical: %d bytes",
		inCompressed, inUncompressed)

	// Without compression enabled, sizes should be approximately equal
	require.InDelta(t, inCompressed, inUncompressed, float64(inUncompressed)*0.05,
		"Without compression, wire size should equal logical size (within 5%%)")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	pl.Shutdown(shutdownCtx)
}

// TestGRPCCompressionBidirectional verifies compression works in both directions
func TestGRPCCompressionBidirectional(t *testing.T) {
	addr := getAvailablePort(t)
	networkAddr := lavasession.NetworkAddressData{
		Address:    addr,
		DisableTLS: true,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pl := NewProviderListener(ctx, networkAddr, "/lava/health")
	require.NotNil(t, pl)

	// Large response
	largeResponseData := []byte(strings.Repeat("response_data_", 5000))

	var receivedRequestSize int
	receiver := &mockRelayReceiver{
		relayFunc: func(ctx context.Context, request *pairingtypes.RelayRequest) (*pairingtypes.RelayReply, error) {
			receivedRequestSize = len(request.RelayData.Data)
			return &pairingtypes.RelayReply{Data: largeResponseData}, nil
		},
	}
	err := pl.RegisterReceiver(receiver, createTestEndpoint(addr))
	require.NoError(t, err)

	time.Sleep(100 * time.Millisecond)

	statsHandler := &compressionStatsHandler{}

	conn, err := grpc.DialContext(ctx, addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
		grpc.WithDefaultCallOptions(grpc.UseCompressor(gzip.Name)),
		grpc.WithStatsHandler(statsHandler),
	)
	require.NoError(t, err)
	defer conn.Close()

	client := pairingtypes.NewRelayerClient(conn)

	// Large, compressible request
	largeRequestData := []byte(strings.Repeat("request_data_", 5000))

	reply, err := client.Relay(ctx, &pairingtypes.RelayRequest{
		RelayData: &pairingtypes.RelayPrivateData{
			Data:         largeRequestData,
			ApiInterface: "jsonrpc",
		},
		RelaySession: &pairingtypes.RelaySession{
			SpecId: "ETH1",
		},
	})
	require.NoError(t, err)
	require.NotNil(t, reply)

	// Verify provider received the full uncompressed request
	require.Equal(t, len(largeRequestData), receivedRequestSize,
		"Provider should receive uncompressed request data")

	// Verify response was received correctly
	require.Equal(t, largeResponseData, reply.Data)

	// Check outgoing (request) compression
	outCompressed := statsHandler.outPayloadCompressedBytes.Load()
	outUncompressed := statsHandler.outPayloadUncompressedBytes.Load()

	t.Logf("Request - Compressed: %d bytes, Uncompressed: %d bytes, Ratio: %.2f%%",
		outCompressed, outUncompressed, float64(outCompressed)/float64(outUncompressed)*100)

	require.Greater(t, outUncompressed, outCompressed,
		"Outgoing request should be compressed")

	// Check incoming (response) compression
	inCompressed := statsHandler.inPayloadCompressedBytes.Load()
	inUncompressed := statsHandler.inPayloadUncompressedBytes.Load()

	t.Logf("Response - Compressed: %d bytes, Uncompressed: %d bytes, Ratio: %.2f%%",
		inCompressed, inUncompressed, float64(inCompressed)/float64(inUncompressed)*100)

	require.Greater(t, inUncompressed, inCompressed,
		"Incoming response should be compressed")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	pl.Shutdown(shutdownCtx)
}

// TestGRPCCompressionWithConnectGRPCClient tests compression via the actual ConnectGRPCClient function
func TestGRPCCompressionWithConnectGRPCClient(t *testing.T) {
	addr := getAvailablePort(t)
	networkAddr := lavasession.NetworkAddressData{
		Address:    addr,
		DisableTLS: true,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pl := NewProviderListener(ctx, networkAddr, "/lava/health")
	require.NotNil(t, pl)

	largeResponseData := []byte(strings.Repeat(`{"result":"ok","data":"test"}`, 2000))

	receiver := &mockRelayReceiver{
		relayFunc: func(ctx context.Context, request *pairingtypes.RelayRequest) (*pairingtypes.RelayReply, error) {
			return &pairingtypes.RelayReply{Data: largeResponseData}, nil
		},
	}
	err := pl.RegisterReceiver(receiver, createTestEndpoint(addr))
	require.NoError(t, err)

	time.Sleep(100 * time.Millisecond)

	// Test with compression enabled using the actual lavasession.ConnectGRPCClient
	connCtx, connCancel := context.WithTimeout(ctx, 5*time.Second)
	defer connCancel()

	// allowInsecure=true, skipTLS=true, allowCompression=true
	conn, err := lavasession.ConnectGRPCClient(connCtx, addr, true, true, true)
	require.NoError(t, err)
	defer conn.Close()

	client := pairingtypes.NewRelayerClient(conn)

	reply, err := client.Relay(ctx, &pairingtypes.RelayRequest{
		RelayData: &pairingtypes.RelayPrivateData{
			Data:         []byte("test"),
			ApiInterface: "jsonrpc",
		},
		RelaySession: &pairingtypes.RelaySession{
			SpecId: "ETH1",
		},
	})
	require.NoError(t, err)
	require.NotNil(t, reply)
	require.Equal(t, largeResponseData, reply.Data,
		"Data should be correctly transmitted with compression enabled via ConnectGRPCClient")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	pl.Shutdown(shutdownCtx)
}

// TestGRPCCompressionSmallPayload verifies behavior with small payloads
func TestGRPCCompressionSmallPayload(t *testing.T) {
	addr := getAvailablePort(t)
	networkAddr := lavasession.NetworkAddressData{
		Address:    addr,
		DisableTLS: true,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pl := NewProviderListener(ctx, networkAddr, "/lava/health")
	require.NotNil(t, pl)

	// Small response - compression may not help much
	smallResponseData := []byte(`{"ok":true}`)

	receiver := &mockRelayReceiver{
		relayFunc: func(ctx context.Context, request *pairingtypes.RelayRequest) (*pairingtypes.RelayReply, error) {
			return &pairingtypes.RelayReply{Data: smallResponseData}, nil
		},
	}
	err := pl.RegisterReceiver(receiver, createTestEndpoint(addr))
	require.NoError(t, err)

	time.Sleep(100 * time.Millisecond)

	statsHandler := &compressionStatsHandler{}

	conn, err := grpc.DialContext(ctx, addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
		grpc.WithDefaultCallOptions(grpc.UseCompressor(gzip.Name)),
		grpc.WithStatsHandler(statsHandler),
	)
	require.NoError(t, err)
	defer conn.Close()

	client := pairingtypes.NewRelayerClient(conn)

	reply, err := client.Relay(ctx, &pairingtypes.RelayRequest{
		RelayData: &pairingtypes.RelayPrivateData{
			Data:         []byte("x"),
			ApiInterface: "jsonrpc",
		},
		RelaySession: &pairingtypes.RelaySession{
			SpecId: "ETH1",
		},
	})
	require.NoError(t, err)
	require.NotNil(t, reply)
	require.Equal(t, smallResponseData, reply.Data)

	// For small payloads, compression still works but may have overhead
	// Just verify data integrity - compression ratio may not be favorable
	t.Logf("Small payload - Compressed: %d bytes, Uncompressed: %d bytes",
		statsHandler.inPayloadCompressedBytes.Load(),
		statsHandler.inPayloadUncompressedBytes.Load())

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	pl.Shutdown(shutdownCtx)
}

// BenchmarkGRPCWithCompression benchmarks relay with compression
func BenchmarkGRPCWithCompression(b *testing.B) {
	addr := getAvailablePortForBenchmark(b)
	networkAddr := lavasession.NetworkAddressData{
		Address:    addr,
		DisableTLS: true,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pl := NewProviderListener(ctx, networkAddr, "/lava/health")

	largeResponseData := []byte(strings.Repeat(`{"result":"benchmark_data"}`, 1000))

	receiver := &mockRelayReceiver{
		relayFunc: func(ctx context.Context, request *pairingtypes.RelayRequest) (*pairingtypes.RelayReply, error) {
			return &pairingtypes.RelayReply{Data: largeResponseData}, nil
		},
	}
	_ = pl.RegisterReceiver(receiver, createTestEndpoint(addr))

	time.Sleep(100 * time.Millisecond)

	conn, _ := grpc.DialContext(ctx, addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
		grpc.WithDefaultCallOptions(grpc.UseCompressor(gzip.Name)),
	)
	defer conn.Close()

	client := pairingtypes.NewRelayerClient(conn)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = client.Relay(ctx, &pairingtypes.RelayRequest{
			RelayData: &pairingtypes.RelayPrivateData{
				Data:         []byte("bench"),
				ApiInterface: "jsonrpc",
			},
			RelaySession: &pairingtypes.RelaySession{
				SpecId: "ETH1",
			},
		})
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	pl.Shutdown(shutdownCtx)
}

// BenchmarkGRPCWithoutCompression benchmarks relay without compression
func BenchmarkGRPCWithoutCompression(b *testing.B) {
	addr := getAvailablePortForBenchmark(b)
	networkAddr := lavasession.NetworkAddressData{
		Address:    addr,
		DisableTLS: true,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pl := NewProviderListener(ctx, networkAddr, "/lava/health")

	largeResponseData := []byte(strings.Repeat(`{"result":"benchmark_data"}`, 1000))

	receiver := &mockRelayReceiver{
		relayFunc: func(ctx context.Context, request *pairingtypes.RelayRequest) (*pairingtypes.RelayReply, error) {
			return &pairingtypes.RelayReply{Data: largeResponseData}, nil
		},
	}
	_ = pl.RegisterReceiver(receiver, createTestEndpoint(addr))

	time.Sleep(100 * time.Millisecond)

	conn, _ := grpc.DialContext(ctx, addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
		// No compression
	)
	defer conn.Close()

	client := pairingtypes.NewRelayerClient(conn)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = client.Relay(ctx, &pairingtypes.RelayRequest{
			RelayData: &pairingtypes.RelayPrivateData{
				Data:         []byte("bench"),
				ApiInterface: "jsonrpc",
			},
			RelaySession: &pairingtypes.RelaySession{
				SpecId: "ETH1",
			},
		})
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	pl.Shutdown(shutdownCtx)
}

func getAvailablePortForBenchmark(b *testing.B) string {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		b.Fatal(err)
	}
	tcpAddr, ok := listener.Addr().(*net.TCPAddr)
	if !ok {
		b.Fatal("expected TCP address")
	}
	port := tcpAddr.Port
	listener.Close()
	return fmt.Sprintf("127.0.0.1:%d", port)
}
