package chainproxy

//
// Right now this is only for Ethereum
// TODO: make this into a proper connection pool that supports
// the chainproxy interface

import (
	"context"
	"errors"
	"log"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/lavanet/lava/protocol/chainlib/chainproxy/rpcclient"
	"github.com/lavanet/lava/utils"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	DialTimeout                                = 500 * time.Millisecond
	ParallelConnectionsFlag                    = "parallel-connections"
	MaximumNumberOfParallelConnectionsAttempts = 10
)

var NumberOfParallelConnections uint = 10

type Connector struct {
	lock        sync.RWMutex
	freeClients []*rpcclient.Client
	usedClients int64
	addr        string
}

func NewConnector(ctx context.Context, nConns uint, addr string) *Connector {
	NumberOfParallelConnections = nConns // set number of parallel connections requested by user (or default.)
	connector := &Connector{
		freeClients: make([]*rpcclient.Client, 0, nConns),
		addr:        addr,
	}

	rpcClient, err := connector.createConnection(ctx, addr, connector.numberOfFreeClients())
	if err != nil {
		utils.LavaFormatFatal("Failed to create the first connection", err, &map[string]string{"address": addr})
	}
	connector.addClient(rpcClient)
	go addClientsAsynchronously(ctx, connector, nConns-1, addr)

	return connector
}

func addClientsAsynchronously(ctx context.Context, connector *Connector, nConns uint, addr string) {
	for i := uint(0); i < nConns; i++ {
		rpcClient, err := connector.createConnection(ctx, addr, connector.numberOfFreeClients())
		if err != nil {
			break
		}
		connector.addClient(rpcClient)
	}
	if (connector.numberOfFreeClients() + connector.numberOfUsedClients()) == 0 {
		utils.LavaFormatFatal("Could not create any connections to the node check address", nil, &map[string]string{"address": addr})
	}
	utils.LavaFormatInfo("Finished adding Clients Asynchronously"+strconv.Itoa(len(connector.freeClients)), nil)
	utils.LavaFormatInfo("Number of parallel connections created: "+strconv.Itoa(len(connector.freeClients)), nil)
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

func (connector *Connector) createConnection(ctx context.Context, addr string, currentNumberOfConnections int) (*rpcclient.Client, error) {
	var rpcClient *rpcclient.Client
	var err error
	numberOfConnectionAttempts := 0
	for {
		numberOfConnectionAttempts += 1
		if numberOfConnectionAttempts > MaximumNumberOfParallelConnectionsAttempts {
			err = utils.LavaFormatError("Reached maximum number of parallel connections attempts, consider decreasing number of connections",
				nil, &map[string]string{"Currently Connected": strconv.FormatUint(uint64(currentNumberOfConnections), 10)},
			)
			break
		}
		if ctx.Err() != nil {
			connector.Close()
			return nil, ctx.Err()
		}
		nctx, cancel := context.WithTimeout(ctx, DialTimeout)
		rpcClient, err = rpcclient.DialContext(nctx, addr)
		if err != nil {
			utils.LavaFormatWarning("Could not connect to the node, retrying", err, &map[string]string{
				"Current Number Of Connections": strconv.FormatUint(uint64(currentNumberOfConnections), 10),
				"Number Of Attempts Remaining":  strconv.Itoa(numberOfConnectionAttempts),
			})
			cancel()
			continue
		}
		cancel()
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
	for {
		connector.lock.Lock()
		log.Println("Connector closing", len(connector.freeClients))
		for i := 0; i < len(connector.freeClients); i++ {
			connector.freeClients[i].Close()
		}
		connector.freeClients = []*rpcclient.Client{}

		if connector.usedClients > 0 {
			log.Println("Connector closing, waiting for in use clients", connector.usedClients)
			connector.lock.Unlock()
			time.Sleep(100 * time.Millisecond)
		} else {
			connector.lock.Unlock()
			break
		}
	}
}

func (connector *Connector) increaseNumberOfClients(ctx context.Context, numberOfFreeClients int) {
	utils.LavaFormatDebug("increasing number of clients", &map[string]string{"numberOfFreeClients": strconv.Itoa(numberOfFreeClients)})
	var rpcClient *rpcclient.Client
	var err error
	for connectionAttempt := 0; connectionAttempt < MaximumNumberOfParallelConnectionsAttempts; connectionAttempt++ {
		nctx, cancel := context.WithTimeout(ctx, DialTimeout)
		rpcClient, err = rpcclient.DialContext(nctx, connector.addr)
		if err != nil {
			utils.LavaFormatDebug(
				"increaseNumberOfClients, Could not connect to the node, retrying",
				&map[string]string{"err": err.Error(), "Number Of Attempts": strconv.Itoa(connectionAttempt)})
			cancel()
			continue
		}
		cancel()

		connector.lock.Lock() // add connection to free list.
		defer connector.lock.Unlock()
		connector.freeClients = append(connector.freeClients, rpcClient)
		return
	}
	utils.LavaFormatDebug("Failed increasing number of clients", nil)
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
	addr        string
}

func NewGRPCConnector(ctx context.Context, nConns uint, addr string) *GRPCConnector {
	NumberOfParallelConnections = nConns // set number of parallel connections requested by user (or default.)
	connector := &GRPCConnector{
		freeClients: make([]*grpc.ClientConn, 0, nConns),
		addr:        addr,
	}

	rpcClient, err := connector.createConnection(ctx, addr, connector.numberOfFreeClients())
	if err != nil {
		utils.LavaFormatFatal("Failed to create the first connection", err, &map[string]string{"address": addr})
	}
	connector.addClient(rpcClient)
	go addClientsAsynchronouslyGrpc(ctx, connector, nConns-1, addr)
	return connector
}

func (connector *GRPCConnector) increaseNumberOfClients(ctx context.Context, numberOfFreeClients int) {
	utils.LavaFormatDebug("increasing number of clients", &map[string]string{"numberOfFreeClients": strconv.Itoa(numberOfFreeClients)})
	var grpcClient *grpc.ClientConn
	var err error
	for connectionAttempt := 0; connectionAttempt < MaximumNumberOfParallelConnectionsAttempts; connectionAttempt++ {
		nctx, cancel := context.WithTimeout(ctx, DialTimeout)
		grpcClient, err = grpc.DialContext(nctx, connector.addr, grpc.WithBlock(), grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			utils.LavaFormatDebug("increaseNumberOfClients, Could not connect to the node, retrying", &map[string]string{"err": err.Error(), "Number Of Attempts": strconv.Itoa(connectionAttempt)})
			cancel()
			continue
		}
		cancel()

		connector.lock.Lock() // add connection to free list.
		defer connector.lock.Unlock()
		connector.freeClients = append(connector.freeClients, grpcClient)
		return
	}
	utils.LavaFormatDebug("increasing number of clients failed", nil)
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
	connector.freeClients = connector.freeClients[1:]
	connector.usedClients++

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
	for {
		connector.lock.Lock()
		log.Println("Connector closing", len(connector.freeClients))
		for i := 0; i < len(connector.freeClients); i++ {
			connector.freeClients[i].Close()
		}
		connector.freeClients = []*grpc.ClientConn{}

		if connector.usedClients > 0 {
			log.Println("Connector closing, waiting for in use clients", connector.usedClients)
			connector.lock.Unlock()
			time.Sleep(100 * time.Millisecond)
		} else {
			connector.lock.Unlock()
			break
		}
	}
}

func addClientsAsynchronouslyGrpc(ctx context.Context, connector *GRPCConnector, nConns uint, addr string) {
	for i := uint(0); i < nConns; i++ {
		rpcClient, err := connector.createConnection(ctx, addr, connector.numberOfFreeClients())
		if err != nil {
			break
		}
		connector.addClient(rpcClient)
	}
	if (connector.numberOfFreeClients() + connector.numberOfUsedClients()) == 0 {
		utils.LavaFormatFatal("Could not create any connections to the node check address", nil, &map[string]string{"address": addr})
	}
	utils.LavaFormatInfo("Finished adding Clients Asynchronously"+strconv.Itoa(len(connector.freeClients)), nil)
	utils.LavaFormatInfo("Number of parallel connections created: "+strconv.Itoa(len(connector.freeClients)), nil)
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

func (connector *GRPCConnector) createConnection(ctx context.Context, addr string, currentNumberOfConnections int) (*grpc.ClientConn, error) {
	var rpcClient *grpc.ClientConn
	var err error
	numberOfConnectionAttempts := 0
	for {
		numberOfConnectionAttempts += 1
		if numberOfConnectionAttempts > MaximumNumberOfParallelConnectionsAttempts {
			err = utils.LavaFormatError("Reached maximum number of parallel connections attempts, consider decreasing number of connections",
				nil, &map[string]string{"Currently Connected": strconv.FormatUint(uint64(currentNumberOfConnections), 10)},
			)
			break
		}
		if ctx.Err() != nil {
			connector.Close()
			return nil, ctx.Err()
		}
		nctx, cancel := context.WithTimeout(ctx, DialTimeout)
		rpcClient, err = grpc.DialContext(nctx, addr, grpc.WithBlock(), grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			utils.LavaFormatWarning("Could not connect to the node, retrying", err, &map[string]string{
				"Current Number Of Connections": strconv.FormatUint(uint64(currentNumberOfConnections), 10),
				"Number Of Attempts Remaining":  strconv.Itoa(numberOfConnectionAttempts),
			})
			cancel()
			continue
		}
		cancel()
		break
	}
	return rpcClient, err
}
