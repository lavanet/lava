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
	"log"
	"os"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/lavanet/lava/protocol/chainlib/chainproxy/rpcclient"
	"github.com/lavanet/lava/protocol/common"
	"github.com/lavanet/lava/utils"
	"github.com/lavanet/lava/utils/sigs"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	ParallelConnectionsFlag                    = "parallel-connections"
	GRPCUseTls                                 = "use-tls"
	GRPCAllowInsecureConnection                = "allow-insecure-connection"
	MaximumNumberOfParallelConnectionsAttempts = 10
	MaxCallRecvMsgSize                         = 1024 * 1024 * 32 // setting receive size to 32mb instead of 4mb default
)

var NumberOfParallelConnections uint = 10

type Connector struct {
	lock          sync.RWMutex
	freeClients   []*rpcclient.Client
	usedClients   int64
	nodeUrl       common.NodeUrl
	hashedNodeUrl string
}

func HashURL(url string) string {
	// Convert the URL string to bytes
	urlBytes := []byte(url)

	// Hash the URL using the HashMsg function
	hashedBytes := sigs.HashMsg(urlBytes)

	// Encode the hashed bytes to a hex string for easier sharing
	return hex.EncodeToString(hashedBytes)
}

func NewConnector(ctx context.Context, nConns uint, nodeUrl common.NodeUrl) (*Connector, error) {
	NumberOfParallelConnections = nConns // set number of parallel connections requested by user (or default.)
	connector := &Connector{
		freeClients:   make([]*rpcclient.Client, 0, nConns),
		nodeUrl:       nodeUrl,
		hashedNodeUrl: HashURL(nodeUrl.Url),
	}

	rpcClient, err := connector.createConnection(ctx, nodeUrl, connector.numberOfFreeClients())
	if err != nil {
		return nil, utils.LavaFormatError("Failed to create the first connection", err, utils.Attribute{Key: "address", Value: nodeUrl.UrlStr()})
	}

	connector.addClient(rpcClient)
	go addClientsAsynchronously(ctx, connector, nConns-1, nodeUrl)

	return connector, nil
}

func addClientsAsynchronously(ctx context.Context, connector *Connector, nConns uint, nodeUrl common.NodeUrl) {
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
	utils.LavaFormatInfo("Finished adding Clients Asynchronously", utils.Attribute{Key: "free clients", Value: len(connector.freeClients)}, utils.Attribute{Key: "url", Value: connector.nodeUrl.String()})
	go connector.connectorLoop(ctx)
}

func (connector *Connector) addClient(client *rpcclient.Client) {
	connector.lock.Lock()
	defer connector.lock.Unlock()
	connector.freeClients = append(connector.freeClients, client)
}

func (connector *Connector) numberOfFreeClients() int {
	connector.lock.RLock()
	defer connector.lock.RUnlock()
	return len(connector.freeClients)
}

func (connector *Connector) numberOfUsedClients() int {
	return int(atomic.LoadInt64(&connector.usedClients))
}

func (connector *Connector) createConnection(ctx context.Context, nodeUrl common.NodeUrl, currentNumberOfConnections int) (*rpcclient.Client, error) {
	var rpcClient *rpcclient.Client
	var err error
	numberOfConnectionAttempts := 0
	for {
		numberOfConnectionAttempts += 1
		if numberOfConnectionAttempts > MaximumNumberOfParallelConnectionsAttempts {
			err = utils.LavaFormatError("Reached maximum number of parallel connections attempts, consider decreasing number of connections",
				nil, utils.Attribute{Key: "Currently Connected", Value: currentNumberOfConnections})
			break
		}
		if ctx.Err() != nil {
			connector.Close()
			return nil, ctx.Err()
		}
		timeout := common.AverageWorldLatency * (1 + time.Duration(numberOfConnectionAttempts))
		nctx, cancel := nodeUrl.LowerContextTimeoutWithDuration(ctx, timeout)
		// add auth path
		authPathNodeUrl := nodeUrl.AuthConfig.AddAuthPath(nodeUrl.Url)
		rpcClient, err = rpcclient.DialContext(nctx, authPathNodeUrl)
		if err != nil {
			utils.LavaFormatWarning("Could not connect to the node, retrying", err, []utils.Attribute{
				{Key: "Current Number Of Connections", Value: currentNumberOfConnections},
				{Key: "Network Address", Value: authPathNodeUrl},
				{Key: "Number Of Attempts", Value: numberOfConnectionAttempts},
				{Key: "timeout", Value: timeout},
			}...)
			cancel()
			continue
		}
		cancel()
		nodeUrl.SetAuthHeaders(ctx, rpcClient.SetHeader)
		break
	}

	return rpcClient, err
}

func (connector *Connector) connectorLoop(ctx context.Context) {
	<-ctx.Done()
	log.Println("connectorLoop ctx.Done")
	connector.Close()
}

func (connector *Connector) Close() {
	for i := 0; ; i++ {
		connector.lock.Lock()
		for i := 0; i < len(connector.freeClients); i++ {
			connector.freeClients[i].Close()
		}
		connector.freeClients = []*rpcclient.Client{}

		if connector.usedClients > 0 {
			if i > 10 {
				utils.LavaFormatError("stuck while closing connector", nil, utils.LogAttr("freeClients", connector.freeClients), utils.LogAttr("usedClients", connector.usedClients))
			}
			connector.lock.Unlock()
			time.Sleep(100 * time.Millisecond)
		} else {
			connector.lock.Unlock()
			break
		}
	}
}

func (connector *Connector) increaseNumberOfClients(ctx context.Context, numberOfFreeClients int) {
	utils.LavaFormatDebug("increasing number of clients", utils.Attribute{Key: "numberOfFreeClients", Value: numberOfFreeClients}, utils.Attribute{Key: "url", Value: connector.nodeUrl.UrlStr()})
	var rpcClient *rpcclient.Client
	var err error
	for connectionAttempt := 0; connectionAttempt < MaximumNumberOfParallelConnectionsAttempts; connectionAttempt++ {
		nctx, cancel := connector.nodeUrl.LowerContextTimeoutWithDuration(ctx, common.AverageWorldLatency*2)
		rpcClient, err = rpcclient.DialContext(nctx, connector.nodeUrl.Url)
		if err != nil {
			utils.LavaFormatDebug(
				"could no increase number of connections to the node jsonrpc connector, retrying",
				[]utils.Attribute{{Key: "err", Value: err.Error()}, {Key: "Number Of Attempts", Value: connectionAttempt}}...)
			cancel()
			continue
		}
		cancel()

		connector.lock.Lock() // add connection to free list.
		defer connector.lock.Unlock()
		connector.freeClients = append(connector.freeClients, rpcClient)
		return
	}
	utils.LavaFormatDebug("Failed increasing number of clients")
}

// getting hashed url from connection. this is never changed. so its not locked.
func (connector *Connector) GetUrlHash() string {
	return connector.hashedNodeUrl
}

func (connector *Connector) GetRpc(ctx context.Context, block bool) (*rpcclient.Client, error) {
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
	connector.freeClients = connector.freeClients[1:]
	connector.usedClients++

	return ret, nil
}

func (connector *Connector) ReturnRpc(rpc *rpcclient.Client) {
	connector.lock.Lock()
	defer connector.lock.Unlock()

	connector.usedClients--
	if len(connector.freeClients) > (int(connector.usedClients) + int(NumberOfParallelConnections) /* the number we started with */) {
		rpc.Close() // close connection
		return      // return without appending back to decrease idle connections
	}
	connector.freeClients = append(connector.freeClients, rpc)
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
	utils.LavaFormatInfo("Finished adding Clients Asynchronously" + strconv.Itoa(len(connector.freeClients)))
	utils.LavaFormatInfo("Number of parallel connections created: " + strconv.Itoa(len(connector.freeClients)))
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
