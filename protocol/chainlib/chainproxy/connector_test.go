package chainproxy

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/rpc"
	"os"
	"testing"
	"time"

	"github.com/lavanet/lava/v2/protocol/chainlib/chainproxy/rpcclient"
	"github.com/lavanet/lava/v2/protocol/common"
	"github.com/lavanet/lava/v2/utils"
	pb_pkg "github.com/lavanet/lava/v2/x/spec/types"
	"github.com/stretchr/testify/require"
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

func TestConnector(t *testing.T) {
	ctx := context.Background()
	conn, err := NewConnector(ctx, numberOfClients, common.NodeUrl{Url: listenerAddressTcp})
	require.NoError(t, err)
	for { // wait for the routine to finish connecting
		if len(conn.freeClients) == numberOfClients {
			break
		}
	}
	require.Equal(t, len(conn.freeClients), numberOfClients)
	increasedClients := numberOfClients * 2 // increase to double the number of clients
	rpcList := make([]*rpcclient.Client, increasedClients)
	for i := 0; i < increasedClients; i++ {
		rpc, err := conn.GetRpc(ctx, true)
		require.NoError(t, err)
		rpcList[i] = rpc
	}
	require.Equal(t, conn.usedClients, int64(increasedClients)) // checking we have used clients
	for i := 0; i < increasedClients; i++ {
		conn.ReturnRpc(rpcList[i])
	}
	require.Equal(t, conn.usedClients, int64(0))              // checking we dont have clients used
	require.Equal(t, len(conn.freeClients), increasedClients) // checking we cleaned clients
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

func TestMain(m *testing.M) {
	listener := createRPCServer()
	for {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		_, err := rpcclient.DialContext(ctx, listenerAddressTcp)
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
