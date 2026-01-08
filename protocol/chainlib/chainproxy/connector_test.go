package chainproxy

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/rpc"
	"os"
	"testing"
	"time"

	"github.com/lavanet/lava/v5/protocol/chainlib/chainproxy/rpcclient"
	"github.com/lavanet/lava/v5/protocol/common"
	"github.com/lavanet/lava/v5/utils"
	pb_pkg "github.com/lavanet/lava/v5/x/spec/types"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/websocket"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

var (
	listenerAddress     = "localhost:0"
	listenerAddressGrpc = "localhost:0"
	listenerAddressTcp  = ""
)

const numberOfClients = 5

type Args struct{}

type TimeServer int64

func (t *TimeServer) GiveServerTime(args *Args, reply *int64) error {
	// Set the value at the pointer got from the client
	*reply = time.Now().Unix()
	return nil
}

func createGRPCServer(t *testing.T) *grpc.Server {
	lis, err := net.Listen("tcp", listenerAddressGrpc)
	listenerAddressGrpc = lis.Addr().String()
	require.NoError(t, err)
	s := grpc.NewServer()
	go s.Serve(lis) // serve in a different thread
	return s
}

type implementedLavanetLavaSpec struct {
	pb_pkg.UnimplementedQueryServer
}

func (is *implementedLavanetLavaSpec) ShowChainInfo(ctx context.Context, req *pb_pkg.QueryShowChainInfoRequest) (*pb_pkg.QueryShowChainInfoResponse, error) {
	md := metadata.New(map[string]string{"content-type": "text/html"})
	grpc.SendHeader(ctx, md)

	result := &pb_pkg.QueryShowChainInfoResponse{ChainID: "Test"}
	return result, nil
}

func createGRPCServerWithRegisteredProto(t *testing.T) *grpc.Server {
	lis, err := net.Listen("tcp", listenerAddressGrpc)
	listenerAddressGrpc = lis.Addr().String()
	require.NoError(t, err)
	s := grpc.NewServer()
	lavanetlavaspec := &implementedLavanetLavaSpec{}
	pb_pkg.RegisterQueryServer(s, lavanetlavaspec)
	go s.Serve(lis) // serve in a different thread
	return s
}

func TestConnector(t *testing.T) {
	ctx := context.Background()
	conn, err := NewConnector(ctx, numberOfClients, common.NodeUrl{Url: listenerAddressTcp})
	require.NoError(t, err)
	defer conn.Close()

	// With shared client design, we always get the same client
	require.NotNil(t, conn.client)

	// Test that GetRpc always returns the same shared client
	client1, err := conn.GetRpc(ctx, true)
	require.NoError(t, err)
	require.NotNil(t, client1)

	client2, err := conn.GetRpc(ctx, true)
	require.NoError(t, err)
	require.NotNil(t, client2)

	// Both should be the same client (shared)
	require.Same(t, client1, client2, "GetRpc should return the same shared client")

	// ReturnRpc is a no-op for HTTP connections, but should not panic
	conn.ReturnRpc(client1)
	conn.ReturnRpc(client2)

	// After "returning", we can still get the client
	client3, err := conn.GetRpc(ctx, true)
	require.NoError(t, err)
	require.Same(t, client1, client3, "Client should still be available after ReturnRpc")
}

func TestConnectorConcurrentAccess(t *testing.T) {
	ctx := context.Background()
	conn, err := NewConnector(ctx, numberOfClients, common.NodeUrl{Url: listenerAddressTcp})
	require.NoError(t, err)
	defer conn.Close()

	// Test concurrent access to the shared client
	const numGoroutines = 50
	done := make(chan bool, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			client, err := conn.GetRpc(ctx, true)
			require.NoError(t, err)
			require.NotNil(t, client)
			conn.ReturnRpc(client) // no-op but should not panic
			done <- true
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		<-done
	}
}

func TestConnectorGrpc(t *testing.T) {
	server := createGRPCServer(t) // create a grpcServer so we can connect to its endpoint and validate everything works.
	defer server.Stop()
	ctx := context.Background()
	conn, err := NewGRPCConnector(ctx, numberOfClients, common.NodeUrl{Url: listenerAddressGrpc})
	require.NoError(t, err)
	for { // wait for the routine to finish connecting
		if len(conn.freeClients) == numberOfClients {
			break
		}
	}
	require.Equal(t, len(conn.freeClients), numberOfClients)
	increasedClients := numberOfClients * 2 // increase to double the number of clients
	rpcList := make([]*grpc.ClientConn, increasedClients)
	for i := 0; i < increasedClients; i++ {
		rpc, err := conn.GetRpc(ctx, true)
		require.NoError(t, err)
		rpcList[i] = rpc
	}
	require.Equal(t, increasedClients, int(conn.usedClients)) // checking we have used clients
	for i := 0; i < increasedClients; i++ {
		conn.ReturnRpc(rpcList[i])
	}
	require.Equal(t, int(conn.usedClients), 0)                // checking we dont have clients used
	require.Equal(t, increasedClients, len(conn.freeClients)) // checking we cleaned clients
}

func TestConnectorGrpcAndInvoke(t *testing.T) {
	server := createGRPCServerWithRegisteredProto(t) // create a grpcServer so we can connect to its endpoint and validate everything works.
	defer server.Stop()
	ctx := context.Background()
	conn, err := NewGRPCConnector(ctx, numberOfClients, common.NodeUrl{Url: listenerAddressGrpc})
	require.NoError(t, err)
	for { // wait for the routine to finish connecting
		if len(conn.freeClients) == numberOfClients {
			break
		}
	}
	// require.Equal(t, len(conn.freeClients), numberOfClients)
	increasedClients := numberOfClients * 2 // increase to double the number of clients
	rpcList := make([]*grpc.ClientConn, increasedClients)
	for i := 0; i < increasedClients; i++ {
		rpc, err := conn.GetRpc(ctx, true)
		require.NoError(t, err)
		rpcList[i] = rpc
		response := &pb_pkg.QueryShowChainInfoResponse{}
		err = grpc.Invoke(ctx, "lavanet.lava.spec.Query/ShowChainInfo", &pb_pkg.QueryShowChainInfoRequest{}, response, rpc)
		require.Equal(t, "Test", response.ChainID)
		require.NoError(t, err)
	}
	require.Equal(t, increasedClients, int(conn.usedClients)) // checking we have used clients
	for i := 0; i < increasedClients; i++ {
		conn.ReturnRpc(rpcList[i])
	}
	require.Equal(t, int(conn.usedClients), 0) // checking we dont have clients used
}

func TestHashing(t *testing.T) {
	ctx := context.Background()
	conn, _ := NewConnector(ctx, numberOfClients, common.NodeUrl{Url: listenerAddressTcp})
	fmt.Println(conn.hashedNodeUrl)
	require.Equal(t, conn.hashedNodeUrl, HashURL(listenerAddressTcp))
}

func createRPCServer() net.Listener {
	timeserver := new(TimeServer)
	// Register the timeserver object upon which the GiveServerTime
	// function will be called from the RPC server (from the client)
	rpc.Register(timeserver)
	// Registers an HTTP handler for RPC messages
	rpc.HandleHTTP()
	// Start listening for the requests on port 1234
	listener, err := net.Listen("tcp", listenerAddress)
	if err != nil {
		log.Fatal("Listener error: ", err)
	}
	listenerAddress = listener.Addr().String()
	listenerAddressTcp = "http://" + listenerAddress
	// Serve accepts incoming HTTP connections on the listener l, creating
	// a new service goroutine for each. The service goroutines read requests
	// and then call handler to reply to them
	go http.Serve(listener, nil)

	return listener
}

func TestMain(m *testing.M) {
	listener := createRPCServer()
	for {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		_, err := rpcclient.DialContext(ctx, listenerAddressTcp, nil)
		if err != nil {
			utils.LavaFormatDebug("waiting for grpc server to launch")
			continue
		}
		cancel()
		break
	}

	// Start running tests.
	code := m.Run()
	listener.Close()
	os.Exit(code)
}

func TestConnectorWebsocket(t *testing.T) {
	// Set up auth headers we expect
	expectedAuthHeader := "Bearer test-token"

	// Create WebSocket server with auth check
	srv := &http.Server{
		Addr: "localhost:0", // random available port
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check auth header
			authHeader := r.Header.Get("Authorization")
			if authHeader != expectedAuthHeader {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
			fmt.Println("connection OK!")
			// Upgrade to websocket
			upgrader := websocket.Server{
				Handler: websocket.Handler(func(ws *websocket.Conn) {
					defer ws.Close()
					// Simple echo server
					for {
						var msg string
						err := websocket.Message.Receive(ws, &msg)
						if err != nil {
							break
						}
						websocket.Message.Send(ws, msg)
					}
				}),
			}
			upgrader.ServeHTTP(w, r)
		}),
	}

	// Start server
	listener, err := net.Listen("tcp", "localhost:0")
	require.NoError(t, err)
	defer listener.Close()

	go srv.Serve(listener)
	wsURL := "ws://" + listener.Addr().String()

	// Create connector with auth config
	ctx := context.Background()
	nodeUrl := common.NodeUrl{
		Url: wsURL,
		AuthConfig: common.AuthConfig{
			AuthHeaders: map[string]string{
				"Authorization": expectedAuthHeader,
			},
		},
	}

	// Create connector
	conn, err := NewConnector(ctx, numberOfClients, nodeUrl)
	require.NoError(t, err)
	defer conn.Close()

	// With shared client design, the client should be available immediately
	require.NotNil(t, conn.client)

	// Get a client and test the connection
	client, err := conn.GetRpc(ctx, true)
	require.NoError(t, err)
	require.NotNil(t, client)

	// Test sending a message using CallContext
	params := map[string]interface{}{
		"test": "value",
	}
	id := json.RawMessage(`1`)
	_, err = client.CallContext(ctx, id, "test_method", params, true, true)
	require.NoError(t, err)

	// Return the client (no-op for shared client)
	conn.ReturnRpc(client)

	// Client should still be the same
	client2, err := conn.GetRpc(ctx, true)
	require.NoError(t, err)
	require.Same(t, client, client2)
}
