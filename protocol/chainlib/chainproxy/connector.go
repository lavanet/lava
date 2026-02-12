package chainproxy

//
// Right now this is only for Ethereum
// TODO: make this into a proper connection pool that supports
// the chainproxy interface

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/lavanet/lava/v5/protocol/chainlib/chainproxy/rpcclient"
	"github.com/lavanet/lava/v5/protocol/common"
	"github.com/lavanet/lava/v5/utils"
	"github.com/lavanet/lava/v5/utils/sigs"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	ParallelConnectionsFlag                    = "parallel-connections"
	GRPCUseTls                                 = "use-tls"
	GRPCAllowInsecureConnection                = "allow-insecure-connection"
	MaximumNumberOfParallelConnectionsAttempts = 10
	MaxCallRecvMsgSize                         = 1024 * 1024 * 512 // setting receive size to 512mb for large debug responses
)

var NumberOfParallelConnections uint = 10

// Connector manages HTTP/JSON-RPC connections to blockchain nodes.
// For HTTP connections, a single shared rpcclient.Client is used since
// http.Client is goroutine-safe and handles connection pooling internally.
// This design eliminates lock contention and maximizes connection reuse.
type Connector struct {
	client        *rpcclient.Client // Single shared client - goroutine safe
	nodeUrl       common.NodeUrl
	hashedNodeUrl string
	closed        int32 // atomic flag to track if connector is closed
}

func HashURL(url string) string {
	// Convert the URL string to bytes
	urlBytes := []byte(url)

	// Hash the URL using the HashMsg function
	hashedBytes := sigs.HashMsg(urlBytes)

	// Encode the hashed bytes to a hex string for easier sharing
	return hex.EncodeToString(hashedBytes)
}

// NewConnector creates a new HTTP/JSON-RPC connector.
// Note: The nConns parameter is ignored for HTTP connections since a single
// shared client is used (http.Client is goroutine-safe and handles connection
// pooling internally). The parameter is kept for API compatibility with
// gRPC connector which still uses connection pooling.
func NewConnector(ctx context.Context, nConns uint, nodeUrl common.NodeUrl) (*Connector, error) {
	NumberOfParallelConnections = nConns // used by gRPC connector, ignored for HTTP
	connector := &Connector{
		nodeUrl:       nodeUrl,
		hashedNodeUrl: HashURL(nodeUrl.Url),
	}

	// Create a single shared client - it's goroutine-safe and handles
	// connection pooling via the shared http.Transport
	rpcClient, err := connector.createConnection(ctx, nodeUrl)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection: %w, url: %s", err, nodeUrl.UrlStr())
	}

	connector.client = rpcClient
	utils.LavaFormatInfo("Created HTTP connector with shared client",
		utils.Attribute{Key: "url", Value: connector.nodeUrl.String()})

	// Start the connector loop to handle graceful shutdown
	go connector.connectorLoop(ctx)

	return connector, nil
}

func (connector *Connector) getRpcClient(ctx context.Context, nodeUrl common.NodeUrl) (*rpcclient.Client, error) {
	authPathNodeUrl := nodeUrl.AuthConfig.AddAuthPath(nodeUrl.Url)
	// origin used for auth header in the websocket case
	authHeaders := nodeUrl.GetAuthHeaders()
	rpcClient, err := rpcclient.DialContext(ctx, authPathNodeUrl, authHeaders)
	if err != nil {
		return nil, err
	}
	nodeUrl.SetAuthHeaders(ctx, rpcClient.SetHeader)
	return rpcClient, nil
}

func (connector *Connector) createConnection(ctx context.Context, nodeUrl common.NodeUrl) (*rpcclient.Client, error) {
	var rpcClient *rpcclient.Client
	var err error

	for attempt := 1; attempt <= MaximumNumberOfParallelConnectionsAttempts; attempt++ {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}

		timeout := common.AverageWorldLatency * (1 + time.Duration(attempt))
		nctx, cancel := nodeUrl.LowerContextTimeoutWithDuration(ctx, timeout)
		rpcClient, err = connector.getRpcClient(nctx, nodeUrl)
		cancel()

		if err == nil {
			return rpcClient, nil
		}

		utils.LavaFormatDebug("Failed to create connection, retrying",
			utils.Attribute{Key: "attempt", Value: attempt},
			utils.Attribute{Key: "error", Value: err.Error()},
			utils.Attribute{Key: "url", Value: nodeUrl.UrlStr()})
	}

	return nil, utils.LavaFormatError("Failed to create connection after max attempts",
		err, utils.Attribute{Key: "url", Value: nodeUrl.UrlStr()})
}

func (connector *Connector) connectorLoop(ctx context.Context) {
	<-ctx.Done()
	utils.LavaFormatDebug("HTTP connector shutting down", utils.Attribute{Key: "url", Value: connector.nodeUrl.String()})
	connector.Close()
}

// Close closes the connector and its underlying client.
func (connector *Connector) Close() {
	// Use atomic to ensure we only close once
	if !atomic.CompareAndSwapInt32(&connector.closed, 0, 1) {
		return // Already closed
	}

	if connector.client != nil {
		connector.client.Close()
	}
}

// getting hashed url from connection. this is never changed. so its not locked.
func (connector *Connector) GetUrlHash() string {
	return connector.hashedNodeUrl
}

// GetRpc returns the shared RPC client.
// The client is goroutine-safe and handles connection pooling internally,
// so no locking or pool management is needed.
// The 'block' parameter is kept for API compatibility but is not used.
func (connector *Connector) GetRpc(ctx context.Context, block bool) (*rpcclient.Client, error) {
	if atomic.LoadInt32(&connector.closed) == 1 {
		return nil, errors.New("connector is closed")
	}
	return connector.client, nil
}

// ReturnRpc is a no-op for HTTP connections.
// The shared client is goroutine-safe and doesn't need to be "returned".
// This method exists for API compatibility.
func (connector *Connector) ReturnRpc(rpc *rpcclient.Client) {
	// No-op: HTTP clients are goroutine-safe and shared.
	// Connection pooling is handled by the underlying http.Transport.
}

type GRPCConnector struct {
	lock        sync.RWMutex
	freeClients []*grpc.ClientConn
	usedClients int64
	credentials credentials.TransportCredentials
	nodeUrl     common.NodeUrl
}

func NewGRPCConnector(ctx context.Context, nConns uint, nodeUrl common.NodeUrl) (*GRPCConnector, error) {
	NumberOfParallelConnections = nConns // set number of parallel connections requested by user (or default.)
	connector := &GRPCConnector{
		freeClients: make([]*grpc.ClientConn, 0, nConns),
		nodeUrl:     nodeUrl,
	}

	rpcClient, err := connector.createConnection(ctx, nodeUrl, connector.numberOfFreeClients())
	if err != nil {
		return nil, utils.LavaFormatError("grpc failed to create the first connection", err, utils.Attribute{Key: "address", Value: nodeUrl.UrlStr()})
	}
	connector.addClient(rpcClient)
	go addClientsAsynchronouslyGrpc(ctx, connector, nConns-1, nodeUrl)
	return connector, nil
}

func getTlsConf(nodeUrl common.NodeUrl) *tls.Config {
	var tlsConf tls.Config
	cacert := nodeUrl.AuthConfig.GetCaCertificateParams()
	if cacert != "" {
		utils.LavaFormatDebug("Loading ca certificate from local path", utils.Attribute{Key: "cacert", Value: cacert})
		caCert, err := os.ReadFile(cacert)
		if err == nil {
			caCertPool := x509.NewCertPool()
			caCertPool.AppendCertsFromPEM(caCert)
			tlsConf.RootCAs = caCertPool
			tlsConf.InsecureSkipVerify = true
		} else {
			utils.LavaFormatError("Failed loading CA certificate, continuing with a default certificate", err)
		}
	} else {
		keyPem, certPem := nodeUrl.AuthConfig.GetLoadingCertificateParams()
		if keyPem != "" && certPem != "" {
			utils.LavaFormatDebug("Loading certificate from local path", utils.Attribute{Key: "certPem", Value: certPem}, utils.Attribute{Key: "keyPem", Value: keyPem})
			cert, err := tls.LoadX509KeyPair(certPem, keyPem)
			if err != nil {
				utils.LavaFormatError("Failed setting up tls certificate from local path, continuing with dynamic certificates", err)
			} else {
				tlsConf.Certificates = []tls.Certificate{cert}
			}
		}
	}
	if nodeUrl.AuthConfig.AllowInsecure {
		tlsConf.InsecureSkipVerify = true
	}
	return &tlsConf
}

func (connector *GRPCConnector) setCredentials(credentials credentials.TransportCredentials) {
	connector.lock.Lock() // add connection to free list.
	defer connector.lock.Unlock()
	connector.credentials = credentials
}

func (connector *GRPCConnector) getCredentials() credentials.TransportCredentials {
	connector.lock.RLock() // add connection to free list.
	defer connector.lock.RUnlock()
	return connector.credentials
}

func (connector *GRPCConnector) getTransportCredentials() grpc.DialOption {
	creds := connector.getCredentials()
	if creds != nil {
		return grpc.WithTransportCredentials(creds)
	}
	return grpc.WithTransportCredentials(insecure.NewCredentials())
}

func (connector *GRPCConnector) increaseNumberOfClients(ctx context.Context, numberOfFreeClients int) {
	utils.LavaFormatDebug("increasing number of clients", utils.Attribute{Key: "numberOfFreeClients", Value: numberOfFreeClients},
		utils.Attribute{Key: "url", Value: connector.nodeUrl.Url})
	var grpcClient *grpc.ClientConn
	var err error
	for connectionAttempt := 0; connectionAttempt < MaximumNumberOfParallelConnectionsAttempts; connectionAttempt++ {
		nctx, cancel := connector.nodeUrl.LowerContextTimeoutWithDuration(ctx, common.AverageWorldLatency*2)
		grpcClient, err = grpc.DialContext(nctx, connector.nodeUrl.Url, grpc.WithBlock(), connector.getTransportCredentials(), grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(MaxCallRecvMsgSize)))
		if err != nil {
			utils.LavaFormatDebug("increaseNumberOfClients, Could not connect to the node, retrying", []utils.Attribute{{Key: "err", Value: err.Error()}, {Key: "Number Of Attempts", Value: connectionAttempt}, {Key: "nodeUrl", Value: connector.nodeUrl.UrlStr()}}...)
			cancel()
			continue
		}
		cancel()

		connector.lock.Lock() // add connection to free list.
		defer connector.lock.Unlock()
		connector.freeClients = append(connector.freeClients, grpcClient)
		return
	}
	utils.LavaFormatDebug("increasing number of clients failed")
}

func (connector *GRPCConnector) GetRpc(ctx context.Context, block bool) (*grpc.ClientConn, error) {
	connector.lock.Lock()
	defer connector.lock.Unlock()

	numberOfFreeClients := len(connector.freeClients)
	if numberOfFreeClients <= int(connector.usedClients) { // if we reached half of the free clients start creating new connections
		go connector.increaseNumberOfClients(ctx, numberOfFreeClients) // increase asynchronously the free list.
	}

	if numberOfFreeClients == 0 {
		if !block {
			return nil, errors.New("out of clients")
		} else {
			for {
				connector.lock.Unlock()
				// if we reached 0 connections we need to create more connections
				// before sleeping, increase asynchronously the free list.
				go connector.increaseNumberOfClients(ctx, numberOfFreeClients)
				time.Sleep(50 * time.Millisecond)
				connector.lock.Lock()
				numberOfFreeClients = len(connector.freeClients)
				if numberOfFreeClients != 0 {
					break
				}
			}
		}
	}

	ret := connector.freeClients[0]
	connector.usedClients++
	connector.freeClients = connector.freeClients[1:]

	return ret, nil
}

func (connector *GRPCConnector) ReturnRpc(rpc *grpc.ClientConn) {
	connector.lock.Lock()
	defer connector.lock.Unlock()

	connector.usedClients--
	if len(connector.freeClients) > (int(connector.usedClients) + int(NumberOfParallelConnections) /* the number we started with */) {
		rpc.Close() // close connection
		return      // return without appending back to decrease idle connections
	}
	connector.freeClients = append(connector.freeClients, rpc)
}

func (connector *GRPCConnector) connectorLoop(ctx context.Context) {
	<-ctx.Done()
	log.Println("connectorLoop ctx.Done")
	connector.Close()
}

func (connector *GRPCConnector) Close() {
	for i := 0; ; i++ {
		connector.lock.Lock()
		for i := 0; i < len(connector.freeClients); i++ {
			connector.freeClients[i].Close()
		}
		connector.freeClients = []*grpc.ClientConn{}

		if connector.usedClients > 0 {
			if i > 10 {
				utils.LavaFormatError("stuck while closing grpc connector", nil, utils.LogAttr("freeClients", connector.freeClients), utils.LogAttr("usedClients", connector.usedClients))
			}
			connector.lock.Unlock()
			time.Sleep(100 * time.Millisecond)
		} else {
			connector.lock.Unlock()
			break
		}
	}
}

func addClientsAsynchronouslyGrpc(ctx context.Context, connector *GRPCConnector, nConns uint, nodeUrl common.NodeUrl) {
	for i := uint(0); i < nConns; i++ {
		rpcClient, err := connector.createConnection(ctx, nodeUrl, connector.numberOfFreeClients())
		if err != nil {
			break
		}
		connector.addClient(rpcClient)
	}
	if (connector.numberOfFreeClients() + connector.numberOfUsedClients()) == 0 {
		utils.LavaFormatFatal("Could not create any connections to the node check address", nil, utils.Attribute{Key: "address", Value: nodeUrl.UrlStr()})
	}
	utils.LavaFormatInfo("Finished adding clients asynchronously", utils.LogAttr("count", len(connector.freeClients)))
	go connector.connectorLoop(ctx)
}

func (connector *GRPCConnector) addClient(client *grpc.ClientConn) {
	connector.lock.Lock()
	defer connector.lock.Unlock()
	connector.freeClients = append(connector.freeClients, client)
}

func (connector *GRPCConnector) numberOfFreeClients() int {
	connector.lock.RLock()
	defer connector.lock.RUnlock()
	return len(connector.freeClients)
}

func (connector *GRPCConnector) numberOfUsedClients() int {
	return int(atomic.LoadInt64(&connector.usedClients))
}

func (connector *GRPCConnector) createConnection(ctx context.Context, nodeUrl common.NodeUrl, currentNumberOfConnections int) (*grpc.ClientConn, error) {
	addr := nodeUrl.Url
	var rpcClient *grpc.ClientConn
	var err error
	numberOfConnectionAttempts := 0

	var credentialsToConnect credentials.TransportCredentials = nil
	// in the case the grpc server needs to connect using tls, but we haven't set credentials yet, should only happen once
	if nodeUrl.AuthConfig.GetUseTls() && connector.getCredentials() == nil {
		// this will allow us to use self signed certificates in development.
		tlsConf := getTlsConf(nodeUrl)
		connector.setCredentials(credentials.NewTLS(tlsConf))
	}

	for {
		numberOfConnectionAttempts += 1
		if numberOfConnectionAttempts > MaximumNumberOfParallelConnectionsAttempts {
			err = utils.LavaFormatError("Reached maximum number of parallel connections attempts, consider decreasing number of connections",
				nil, utils.Attribute{Key: "Currently Connected", Value: currentNumberOfConnections})
			return nil, err
		}
		if ctx.Err() != nil {
			connector.Close()
			return nil, ctx.Err()
		}
		nctx, cancel := connector.nodeUrl.LowerContextTimeoutWithDuration(ctx, common.AverageWorldLatency*2)
		rpcClient, err = grpc.DialContext(nctx, addr, grpc.WithBlock(), connector.getTransportCredentials(), grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(MaxCallRecvMsgSize)))
		cancel()
		if err == nil {
			return rpcClient, nil
		}

		// in case the provider didn't set TLS config and there are no active connections will do a retry with secure connection in case the endpoint is secure
		if connector.getCredentials() == nil && connector.numberOfFreeClients()+connector.numberOfUsedClients() == 0 {
			if credentialsToConnect == nil {
				tlsConf := getTlsConf(nodeUrl)
				credentialsToConnect = credentials.NewTLS(tlsConf)
			}
			nctx, cancel := connector.nodeUrl.LowerContextTimeoutWithDuration(ctx, common.AverageWorldLatency*2)
			var errNew error
			rpcClient, errNew = grpc.DialContext(nctx, addr, grpc.WithBlock(), grpc.WithTransportCredentials(credentialsToConnect), grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(MaxCallRecvMsgSize)))
			cancel()
			if errNew == nil {
				// this means our endpoint is TLS, and we support upgrading even if the config didn't explicitly say it
				utils.LavaFormatDebug("upgraded TLS connection for grpc instead of insecure", utils.LogAttr("address", nodeUrl.String()))
				connector.setCredentials(credentialsToConnect)
				return rpcClient, nil
			}
		}
		utils.LavaFormatWarning("grpc could not connect to the node, retrying", err, []utils.Attribute{{
			Key: "Current Number Of Connections", Value: currentNumberOfConnections,
		}, {Key: "Number Of Attempts Remaining", Value: numberOfConnectionAttempts}, {Key: "nodeUrl", Value: connector.nodeUrl.UrlStr()}}...)
	}
}
