package chainproxy

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"log"
	"math/big"
	"net"
	"net/http"
	"net/rpc"
	"testing"
	"time"

	"github.com/lavanet/lava/relayer/chainproxy/rpcclient"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

const (
	listenerAddress    = "localhost:1234"
	listenerAddressTcp = "http://localhost:1234"
	numberOfClients    = 5
)

type Args struct{}

type TimeServer int64

func (t *TimeServer) GiveServerTime(args *Args, reply *int64) error {
	// Set the value at the pointer got from the client
	*reply = time.Now().Unix()
	return nil
}

func createGRPCServer(t *testing.T) *grpc.Server {
	lis, err := net.Listen("tcp", listenerAddress)
	require.Nil(t, err)
	s := grpc.NewServer()
	go s.Serve(lis) // serve in a different thread
	return s
}

func createRPCServer(t *testing.T) net.Listener {
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
	// Serve accepts incoming HTTP connections on the listener l, creating
	// a new service goroutine for each. The service goroutines read requests
	// and then call handler to reply to them
	go http.Serve(listener, nil)

	return listener
}

func TestConnector(t *testing.T) {
	listener := createRPCServer(t) // create a grpcServer so we can connect to its endpoint and validate everything works.
	defer listener.Close()
	ctx := context.Background()
	conn := NewConnector(ctx, numberOfClients, listenerAddressTcp)
	require.Equal(t, len(conn.freeClients), numberOfClients)
	increasedClients := numberOfClients * 2 // increase to double the number of clients
	rpcList := make([]*rpcclient.Client, increasedClients)
	for i := 0; i < increasedClients; i++ {
		rpc, err := conn.GetRpc(ctx, true)
		require.Nil(t, err)
		rpcList[i] = rpc
	}
	require.Equal(t, conn.usedClients, increasedClients) // checking we have used clients
	for i := 0; i < increasedClients; i++ {
		conn.ReturnRpc(rpcList[i])
	}
	require.Equal(t, conn.usedClients, 0)                     // checking we dont have clients used
	require.Equal(t, len(conn.freeClients), increasedClients) // checking we cleaned clients
}

func TestConnectorGrpc(t *testing.T) {
	server := createGRPCServer(t) // create a grpcServer so we can connect to its endpoint and validate everything works.
	defer server.Stop()
	ctx := context.Background()
	conn := NewGRPCConnector(ctx, numberOfClients, listenerAddress)
	require.Equal(t, len(conn.freeClients), numberOfClients)
	increasedClients := numberOfClients * 2 // increase to double the number of clients
	rpcList := make([]*grpc.ClientConn, increasedClients)
	for i := 0; i < increasedClients; i++ {
		rpc, err := conn.GetRpc(ctx, true)
		require.Nil(t, err)
		rpcList[i] = rpc
	}
	require.Equal(t, conn.usedClients, increasedClients) // checking we have used clients
	for i := 0; i < increasedClients; i++ {
		conn.ReturnRpc(rpcList[i])
	}
	require.Equal(t, conn.usedClients, 0)                     // checking we dont have clients used
	require.Equal(t, len(conn.freeClients), increasedClients) // checking we cleaned clients
}

func TestConnectorGrpcTest(t *testing.T) {
	ctx := context.Background()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		log.Fatalf("failed to generate private key: %v", err)
	}

	// Define a self-signed certificate.
	template := x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "example.com"},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(time.Hour * 24 * 365),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		DNSNames:              []string{"example.com", "localhost"},
		IPAddresses:           []net.IP{net.ParseIP("127.0.0.1")},
	}
	cert, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	if err != nil {
		log.Fatalf("failed to create self-signed certificate: %v", err)
	}

	creds := credentials.NewTLS(&tls.Config{
		Certificates: []tls.Certificate{{
			Certificate: [][]byte{cert},
			PrivateKey:  key,
		}},
		InsecureSkipVerify: true,
	})

	nctx, cancel := context.WithTimeout(ctx, DialTimeout*3)
	defer cancel()
	grpcClient, err := grpc.DialContext(nctx, "secured-connection-ip", grpc.WithBlock(), grpc.WithInsecure())
	require.Nil(t, err)
	grpcClient.Close()
	grpcClient, err = grpc.DialContext(nctx,
		"secured-connection-ip",
		grpc.WithBlock(),
		grpc.WithTransportCredentials(creds))
	require.Nil(t, err)

	grpcClient.Close()

}
