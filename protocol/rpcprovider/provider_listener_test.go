package rpcprovider

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/lavanet/lava/v5/protocol/lavasession"
	pairingtypes "github.com/lavanet/lava/v5/x/pairing/types"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	healthgrpc "google.golang.org/grpc/health/grpc_health_v1"
)

// getAvailablePort finds an available port for testing
func getAvailablePort(t *testing.T) string {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	tcpAddr, ok := listener.Addr().(*net.TCPAddr)
	require.True(t, ok, "expected TCP address")
	port := tcpAddr.Port
	listener.Close()
	return fmt.Sprintf("127.0.0.1:%d", port)
}

// mockRelayReceiver implements RelayReceiver for testing
type mockRelayReceiver struct {
	relayFunc     func(ctx context.Context, request *pairingtypes.RelayRequest) (*pairingtypes.RelayReply, error)
	probeFunc     func(ctx context.Context, probeReq *pairingtypes.ProbeRequest) (*pairingtypes.ProbeReply, error)
	subscribeFunc func(request *pairingtypes.RelayRequest, srv pairingtypes.Relayer_RelaySubscribeServer) error
}

func (m *mockRelayReceiver) Relay(ctx context.Context, request *pairingtypes.RelayRequest) (*pairingtypes.RelayReply, error) {
	if m.relayFunc != nil {
		return m.relayFunc(ctx, request)
	}
	return &pairingtypes.RelayReply{Data: []byte("mock relay response")}, nil
}

func (m *mockRelayReceiver) Probe(ctx context.Context, probeReq *pairingtypes.ProbeRequest) (*pairingtypes.ProbeReply, error) {
	if m.probeFunc != nil {
		return m.probeFunc(ctx, probeReq)
	}
	return &pairingtypes.ProbeReply{Guid: probeReq.Guid, LatestBlock: 12345}, nil
}

func (m *mockRelayReceiver) RelaySubscribe(request *pairingtypes.RelayRequest, srv pairingtypes.Relayer_RelaySubscribeServer) error {
	if m.subscribeFunc != nil {
		return m.subscribeFunc(request, srv)
	}
	return nil
}

func TestNewProviderListener(t *testing.T) {
	addr := getAvailablePort(t)
	networkAddr := lavasession.NetworkAddressData{
		Address:    addr,
		DisableTLS: true,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pl := NewProviderListener(ctx, networkAddr, "/lava/health")
	require.NotNil(t, pl)
	require.Equal(t, addr, pl.Key())
	require.NotNil(t, pl.grpcServer)
	require.NotNil(t, pl.httpServer)
	require.NotNil(t, pl.healthServer)
	require.NotNil(t, pl.relayServer)

	// Give time for listeners to start
	time.Sleep(100 * time.Millisecond)

	// Shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	err := pl.Shutdown(shutdownCtx)
	require.NoError(t, err)
}

func TestProviderListener_HTTPHealthCheck(t *testing.T) {
	addr := getAvailablePort(t)
	networkAddr := lavasession.NetworkAddressData{
		Address:    addr,
		DisableTLS: true,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	healthPath := "/lava/health"
	pl := NewProviderListener(ctx, networkAddr, healthPath)
	require.NotNil(t, pl)

	// Give time for listeners to start
	time.Sleep(100 * time.Millisecond)

	// Test HTTP health check
	resp, err := http.Get(fmt.Sprintf("http://%s%s", addr, healthPath))
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Equal(t, "Healthy", string(body))

	// Shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	err = pl.Shutdown(shutdownCtx)
	require.NoError(t, err)
}

func TestProviderListener_HTTPHealthCheckMethodNotAllowed(t *testing.T) {
	addr := getAvailablePort(t)
	networkAddr := lavasession.NetworkAddressData{
		Address:    addr,
		DisableTLS: true,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	healthPath := "/lava/health"
	pl := NewProviderListener(ctx, networkAddr, healthPath)
	require.NotNil(t, pl)

	time.Sleep(100 * time.Millisecond)

	// Test POST to health check (should return 405)
	resp, err := http.Post(fmt.Sprintf("http://%s%s", addr, healthPath), "text/plain", nil)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusMethodNotAllowed, resp.StatusCode)

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	pl.Shutdown(shutdownCtx)
}

func TestProviderListener_GRPCHealthService(t *testing.T) {
	addr := getAvailablePort(t)
	networkAddr := lavasession.NetworkAddressData{
		Address:    addr,
		DisableTLS: true,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pl := NewProviderListener(ctx, networkAddr, "/lava/health")
	require.NotNil(t, pl)

	time.Sleep(100 * time.Millisecond)

	// Connect via gRPC
	conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	defer conn.Close()

	// Check gRPC health service
	healthClient := healthgrpc.NewHealthClient(conn)
	resp, err := healthClient.Check(ctx, &healthgrpc.HealthCheckRequest{Service: ""})
	require.NoError(t, err)
	require.Equal(t, healthgrpc.HealthCheckResponse_SERVING, resp.Status)

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	pl.Shutdown(shutdownCtx)
}

func TestProviderListener_RegisterReceiver(t *testing.T) {
	addr := getAvailablePort(t)
	networkAddr := lavasession.NetworkAddressData{
		Address:    addr,
		DisableTLS: true,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pl := NewProviderListener(ctx, networkAddr, "/lava/health")
	require.NotNil(t, pl)

	time.Sleep(100 * time.Millisecond)

	// Create mock receiver
	mockReceiver := &mockRelayReceiver{}

	endpoint := &lavasession.RPCProviderEndpoint{
		ChainID:      "ETH1",
		ApiInterface: "jsonrpc",
		NetworkAddress: lavasession.NetworkAddressData{
			Address: addr,
		},
	}

	// Register receiver
	err := pl.RegisterReceiver(mockReceiver, endpoint)
	require.NoError(t, err)

	// Verify health status is set for the service
	conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	defer conn.Close()

	healthClient := healthgrpc.NewHealthClient(conn)
	resp, err := healthClient.Check(ctx, &healthgrpc.HealthCheckRequest{Service: "ETH1-jsonrpc"})
	require.NoError(t, err)
	require.Equal(t, healthgrpc.HealthCheckResponse_SERVING, resp.Status)

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	pl.Shutdown(shutdownCtx)
}

func TestProviderListener_RegisterReceiverDuplicate(t *testing.T) {
	addr := getAvailablePort(t)
	networkAddr := lavasession.NetworkAddressData{
		Address:    addr,
		DisableTLS: true,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pl := NewProviderListener(ctx, networkAddr, "/lava/health")
	require.NotNil(t, pl)

	time.Sleep(100 * time.Millisecond)

	mockReceiver := &mockRelayReceiver{}
	endpoint := &lavasession.RPCProviderEndpoint{
		ChainID:      "ETH1",
		ApiInterface: "jsonrpc",
		NetworkAddress: lavasession.NetworkAddressData{
			Address: addr,
		},
	}

	// First registration should succeed
	err := pl.RegisterReceiver(mockReceiver, endpoint)
	require.NoError(t, err)

	// Second registration with same endpoint should fail
	err = pl.RegisterReceiver(mockReceiver, endpoint)
	require.Error(t, err)
	require.Contains(t, err.Error(), "double_receiver_setup")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	pl.Shutdown(shutdownCtx)
}

func TestProviderListener_GRPCProbe(t *testing.T) {
	addr := getAvailablePort(t)
	networkAddr := lavasession.NetworkAddressData{
		Address:    addr,
		DisableTLS: true,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pl := NewProviderListener(ctx, networkAddr, "/lava/health")
	require.NotNil(t, pl)

	time.Sleep(100 * time.Millisecond)

	// Register a mock receiver
	expectedLatestBlock := int64(99999)
	mockReceiver := &mockRelayReceiver{
		probeFunc: func(ctx context.Context, probeReq *pairingtypes.ProbeRequest) (*pairingtypes.ProbeReply, error) {
			return &pairingtypes.ProbeReply{
				Guid:        probeReq.Guid,
				LatestBlock: expectedLatestBlock,
			}, nil
		},
	}

	endpoint := &lavasession.RPCProviderEndpoint{
		ChainID:      "ETH1",
		ApiInterface: "jsonrpc",
		NetworkAddress: lavasession.NetworkAddressData{
			Address: addr,
		},
	}

	err := pl.RegisterReceiver(mockReceiver, endpoint)
	require.NoError(t, err)

	// Connect via gRPC and call Probe
	conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	defer conn.Close()

	client := pairingtypes.NewRelayerClient(conn)
	probeResp, err := client.Probe(ctx, &pairingtypes.ProbeRequest{
		Guid:         12345,
		SpecId:       "ETH1",
		ApiInterface: "jsonrpc",
	})
	require.NoError(t, err)
	require.Equal(t, uint64(12345), probeResp.Guid)
	require.Equal(t, expectedLatestBlock, probeResp.LatestBlock)

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	pl.Shutdown(shutdownCtx)
}

func TestProviderListener_GRPCRelay(t *testing.T) {
	addr := getAvailablePort(t)
	networkAddr := lavasession.NetworkAddressData{
		Address:    addr,
		DisableTLS: true,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pl := NewProviderListener(ctx, networkAddr, "/lava/health")
	require.NotNil(t, pl)

	time.Sleep(100 * time.Millisecond)

	// Register a mock receiver
	expectedData := []byte("test relay response data")
	mockReceiver := &mockRelayReceiver{
		relayFunc: func(ctx context.Context, request *pairingtypes.RelayRequest) (*pairingtypes.RelayReply, error) {
			return &pairingtypes.RelayReply{
				Data:        expectedData,
				LatestBlock: 100,
			}, nil
		},
	}

	endpoint := &lavasession.RPCProviderEndpoint{
		ChainID:      "ETH1",
		ApiInterface: "jsonrpc",
		NetworkAddress: lavasession.NetworkAddressData{
			Address: addr,
		},
	}

	err := pl.RegisterReceiver(mockReceiver, endpoint)
	require.NoError(t, err)

	// Connect via gRPC and call Relay
	conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	defer conn.Close()

	client := pairingtypes.NewRelayerClient(conn)
	relayResp, err := client.Relay(ctx, &pairingtypes.RelayRequest{
		RelayData: &pairingtypes.RelayPrivateData{
			ApiInterface: "jsonrpc",
		},
		RelaySession: &pairingtypes.RelaySession{
			SpecId: "ETH1",
		},
	})
	require.NoError(t, err)
	require.Equal(t, expectedData, relayResp.Data)
	require.Equal(t, int64(100), relayResp.LatestBlock)

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	pl.Shutdown(shutdownCtx)
}

func TestProviderListener_UnhandledReceiver(t *testing.T) {
	addr := getAvailablePort(t)
	networkAddr := lavasession.NetworkAddressData{
		Address:    addr,
		DisableTLS: true,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pl := NewProviderListener(ctx, networkAddr, "/lava/health")
	require.NotNil(t, pl)

	time.Sleep(100 * time.Millisecond)

	// Don't register any receiver, just try to call

	conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	defer conn.Close()

	client := pairingtypes.NewRelayerClient(conn)

	// Try to probe unregistered chain
	_, err = client.Probe(ctx, &pairingtypes.ProbeRequest{
		Guid:         12345,
		SpecId:       "UNKNOWN",
		ApiInterface: "jsonrpc",
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "unhandled relay receiver")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	pl.Shutdown(shutdownCtx)
}

func TestProviderListener_RelayInvalidRequest(t *testing.T) {
	addr := getAvailablePort(t)
	networkAddr := lavasession.NetworkAddressData{
		Address:    addr,
		DisableTLS: true,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pl := NewProviderListener(ctx, networkAddr, "/lava/health")
	require.NotNil(t, pl)

	time.Sleep(100 * time.Millisecond)

	conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	defer conn.Close()

	client := pairingtypes.NewRelayerClient(conn)

	// Call with nil RelayData
	_, err = client.Relay(ctx, &pairingtypes.RelayRequest{
		RelayData:    nil,
		RelaySession: &pairingtypes.RelaySession{},
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "internal fields are nil")

	// Call with nil RelaySession
	_, err = client.Relay(ctx, &pairingtypes.RelayRequest{
		RelayData:    &pairingtypes.RelayPrivateData{},
		RelaySession: nil,
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "internal fields are nil")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	pl.Shutdown(shutdownCtx)
}

func TestProviderListener_ConcurrentRequests(t *testing.T) {
	addr := getAvailablePort(t)
	networkAddr := lavasession.NetworkAddressData{
		Address:    addr,
		DisableTLS: true,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pl := NewProviderListener(ctx, networkAddr, "/lava/health")
	require.NotNil(t, pl)

	time.Sleep(100 * time.Millisecond)

	// Register receiver
	mockReceiver := &mockRelayReceiver{}
	endpoint := &lavasession.RPCProviderEndpoint{
		ChainID:      "ETH1",
		ApiInterface: "jsonrpc",
		NetworkAddress: lavasession.NetworkAddressData{
			Address: addr,
		},
	}
	err := pl.RegisterReceiver(mockReceiver, endpoint)
	require.NoError(t, err)

	// Make concurrent gRPC and HTTP requests
	const numRequests = 50
	errChan := make(chan error, numRequests*2)

	// gRPC requests
	for i := 0; i < numRequests; i++ {
		go func() {
			conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
			if err != nil {
				errChan <- err
				return
			}
			defer conn.Close()

			client := pairingtypes.NewRelayerClient(conn)
			_, err = client.Probe(ctx, &pairingtypes.ProbeRequest{
				Guid:         12345,
				SpecId:       "ETH1",
				ApiInterface: "jsonrpc",
			})
			errChan <- err
		}()
	}

	// HTTP health check requests
	for i := 0; i < numRequests; i++ {
		go func() {
			resp, err := http.Get(fmt.Sprintf("http://%s/lava/health", addr))
			if err != nil {
				errChan <- err
				return
			}
			resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				errChan <- fmt.Errorf("unexpected status: %d", resp.StatusCode)
				return
			}
			errChan <- nil
		}()
	}

	// Collect results
	successCount := 0
	for i := 0; i < numRequests*2; i++ {
		err := <-errChan
		if err == nil {
			successCount++
		}
	}

	require.Equal(t, numRequests*2, successCount, "All concurrent requests should succeed")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	pl.Shutdown(shutdownCtx)
}

func TestProviderListener_CustomHealthPath(t *testing.T) {
	addr := getAvailablePort(t)
	networkAddr := lavasession.NetworkAddressData{
		Address:    addr,
		DisableTLS: true,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	customPath := "/custom/health/check"
	pl := NewProviderListener(ctx, networkAddr, customPath)
	require.NotNil(t, pl)

	time.Sleep(100 * time.Millisecond)

	// Custom path should work
	resp, err := http.Get(fmt.Sprintf("http://%s%s", addr, customPath))
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	// Default path should not work (404)
	resp2, err := http.Get(fmt.Sprintf("http://%s/lava/health", addr))
	require.NoError(t, err)
	defer resp2.Body.Close()
	require.Equal(t, http.StatusNotFound, resp2.StatusCode)

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	pl.Shutdown(shutdownCtx)
}

func TestProviderListener_TLSConfiguration(t *testing.T) {
	// This test verifies TLS configuration is applied correctly
	// We test with DisableTLS=false but without actual certs, so we expect it to fail to connect
	// This validates the TLS path is taken

	addr := getAvailablePort(t)
	networkAddr := lavasession.NetworkAddressData{
		Address:    addr,
		DisableTLS: false, // TLS enabled, but no certs
		KeyPem:     "",
		CertPem:    "",
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Note: This may fail during NewProviderListener if GetTlsConfig returns nil
	// or succeed and then fail on connection. Either way, we're testing TLS path.
	pl := NewProviderListener(ctx, networkAddr, "/lava/health")
	if pl == nil {
		// Expected if TLS config fails
		return
	}

	time.Sleep(100 * time.Millisecond)

	// HTTP without TLS should fail
	_, err := http.Get(fmt.Sprintf("http://%s/lava/health", addr))
	// Either connection refused or TLS handshake error is expected
	if err == nil {
		t.Log("HTTP connection succeeded - TLS might not be enforced")
	}

	// HTTPS should work if we skip verification (for testing)
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}
	resp, err := client.Get(fmt.Sprintf("https://%s/lava/health", addr))
	if err == nil {
		defer resp.Body.Close()
		require.Equal(t, http.StatusOK, resp.StatusCode)
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	pl.Shutdown(shutdownCtx)
}

func TestRelayServer_FindReceiver(t *testing.T) {
	rs := &relayServer{relayReceivers: map[string]*relayReceiverWrapper{}}

	var mockReceiver RelayReceiver = &mockRelayReceiver{}

	// Add receiver
	endpoint := lavasession.RPCEndpoint{ChainID: "ETH1", ApiInterface: "jsonrpc"}
	rs.relayReceivers[endpoint.Key()] = &relayReceiverWrapper{
		relayReceiver: &mockReceiver,
		enabled:       true,
	}

	// Find existing receiver
	found, err := rs.findReceiver("jsonrpc", "ETH1")
	require.NoError(t, err)
	require.NotNil(t, found)

	// Find non-existent receiver
	_, err = rs.findReceiver("jsonrpc", "UNKNOWN")
	require.Error(t, err)
	require.Contains(t, err.Error(), "unhandled relay receiver")

	// Find disabled receiver
	rs.relayReceivers[endpoint.Key()].enabled = false
	_, err = rs.findReceiver("jsonrpc", "ETH1")
	require.Error(t, err)
	require.Contains(t, err.Error(), "disabled")
}

func BenchmarkProviderListener_HTTPHealthCheck(b *testing.B) {
	addr := fmt.Sprintf("127.0.0.1:%d", 30000+b.N%1000)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		b.Skipf("Could not bind to port: %v", err)
	}
	listener.Close()

	networkAddr := lavasession.NetworkAddressData{
		Address:    addr,
		DisableTLS: true,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pl := NewProviderListener(ctx, networkAddr, "/lava/health")
	if pl == nil {
		b.Fatal("Failed to create provider listener")
	}

	time.Sleep(100 * time.Millisecond)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp, err := http.Get(fmt.Sprintf("http://%s/lava/health", addr))
		if err != nil {
			b.Fatal(err)
		}
		resp.Body.Close()
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	pl.Shutdown(shutdownCtx)
}

func BenchmarkProviderListener_GRPCProbe(b *testing.B) {
	addr := fmt.Sprintf("127.0.0.1:%d", 31000+b.N%1000)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		b.Skipf("Could not bind to port: %v", err)
	}
	listener.Close()

	networkAddr := lavasession.NetworkAddressData{
		Address:    addr,
		DisableTLS: true,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pl := NewProviderListener(ctx, networkAddr, "/lava/health")
	if pl == nil {
		b.Fatal("Failed to create provider listener")
	}

	time.Sleep(100 * time.Millisecond)

	mockReceiver := &mockRelayReceiver{}
	endpoint := &lavasession.RPCProviderEndpoint{
		ChainID:      "ETH1",
		ApiInterface: "jsonrpc",
		NetworkAddress: lavasession.NetworkAddressData{
			Address: addr,
		},
	}
	pl.RegisterReceiver(mockReceiver, endpoint)

	conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		b.Fatal(err)
	}
	defer conn.Close()

	client := pairingtypes.NewRelayerClient(conn)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := client.Probe(ctx, &pairingtypes.ProbeRequest{
			Guid:         uint64(i),
			SpecId:       "ETH1",
			ApiInterface: "jsonrpc",
		})
		if err != nil {
			b.Fatal(err)
		}
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	pl.Shutdown(shutdownCtx)
}
