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
	"time"

	"github.com/lavanet/lava/relayer/chainproxy/rpcclient"
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
	lock        utils.LavaMutex
	freeClients []*rpcclient.Client
	usedClients int
	addr        string
}

func NewConnector(ctx context.Context, nConns uint, addr string) *Connector {
	NumberOfParallelConnections = nConns // set number of parallel connections requested by user (or default.)
	connector := &Connector{
		freeClients: make([]*rpcclient.Client, 0, nConns),
		addr:        addr,
	}
	reachedClientLimit := false

	for i := uint(0); i < nConns; i++ {
		if reachedClientLimit {
			break
		}
		var rpcClient *rpcclient.Client
		var err error
		numberOfConnectionAttempts := 0
		for {
			numberOfConnectionAttempts += 1
			if numberOfConnectionAttempts > MaximumNumberOfParallelConnectionsAttempts {
				utils.LavaFormatError("Reached maximum number of parallel connections attempts, consider decreasing number of connections",
					nil,
					&map[string]string{"Number of parallel connections": strconv.FormatUint(uint64(nConns), 10),
						"Currently Connected": strconv.FormatUint(uint64(len(connector.freeClients)), 10),
					},
				)
				reachedClientLimit = true
				break
			}
			if ctx.Err() != nil {
				connector.Close()
				return nil
			}
			nctx, cancel := context.WithTimeout(ctx, DialTimeout)
			rpcClient, err = rpcclient.DialContext(nctx, addr)
			if err != nil {
				utils.LavaFormatWarning("Could not connect to the node, retrying",
					err,
					&map[string]string{
						"Current Number Of Connections": strconv.FormatUint(uint64(i), 10),
						"Number Of Attempts Remaining":  strconv.Itoa(numberOfConnectionAttempts),
					})
				cancel()
				continue
			}
			cancel()
			break
		}
		connector.freeClients = append(connector.freeClients, rpcClient)
	}
	utils.LavaFormatInfo("Number of parallel connections created: "+strconv.Itoa(len(connector.freeClients)), nil)
	go connector.connectorLoop(ctx)
	return connector
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

func (connector *Connector) increaseNumberOfClients(ctx context.Context) {
	utils.LavaFormatInfo("No clients are free, increasing number of clients", nil)
	var rpcClient *rpcclient.Client
	var err error
	for connectionAttempt := 0; connectionAttempt < MaximumNumberOfParallelConnectionsAttempts; connectionAttempt++ {
		nctx, cancel := context.WithTimeout(ctx, DialTimeout)
		rpcClient, err = rpcclient.DialContext(nctx, connector.addr)
		if err != nil {
			utils.LavaFormatWarning("increaseNumberOfClients, Could not connect to the node, retrying",
				err,
				&map[string]string{
					"Number Of Attempts": strconv.Itoa(connectionAttempt),
				})
			cancel()
			continue
		}
		cancel()

		connector.lock.Lock() // add connection to free list.
		defer connector.lock.Unlock()
		connector.freeClients = append(connector.freeClients, rpcClient)
		return
	}
}

func (connector *Connector) GetRpc(ctx context.Context, block bool) (*rpcclient.Client, error) {
	connector.lock.Lock()
	defer connector.lock.Unlock()
	if len(connector.freeClients) <= connector.usedClients { // if we reached half of the free clients start creating new connections
		go connector.increaseNumberOfClients(ctx) // increase asynchronously the free list.
	}

	if len(connector.freeClients) == 0 {
		if !block {
			return nil, errors.New("out of clients")
		} else {
			for {
				connector.lock.Unlock()
				// if we reached 0 connections we need to create more connections
				// before sleeping, increase asynchronously the free list.
				go connector.increaseNumberOfClients(ctx)
				time.Sleep(50 * time.Millisecond)
				connector.lock.Lock()
				if len(connector.freeClients) != 0 {
					break
				}
			}
		}
	}

	ret := connector.freeClients[len(connector.freeClients)-1]
	connector.freeClients = connector.freeClients[:len(connector.freeClients)-1]
	connector.usedClients++

	return ret, nil
}

func (connector *Connector) ReturnRpc(rpc *rpcclient.Client) {
	connector.lock.Lock()
	defer connector.lock.Unlock()

	connector.usedClients--
	if len(connector.freeClients) > (connector.usedClients + int(NumberOfParallelConnections) /* the number we started with */) {
		rpc.Close() // close connection
		return      // return without appending back to decrease idle connections
	}
	connector.freeClients = append(connector.freeClients, rpc)

}

type GRPCConnector struct {
	lock        sync.RWMutex
	freeClients []*grpc.ClientConn
	usedClients int
	addr        string
}

func NewGRPCConnector(ctx context.Context, nConns uint, addr string) *GRPCConnector {
	connector := &GRPCConnector{
		freeClients: make([]*grpc.ClientConn, 0, nConns),
		addr:        addr,
	}

	NumberOfParallelConnections = nConns // set number of parallel connections requested by user (or default.)
	reachedClientLimit := false

	for i := uint(0); i < nConns; i++ {
		if reachedClientLimit {
			break
		}
		var grpcClient *grpc.ClientConn
		var err error
		numberOfConnectionAttempts := 0
		for {
			numberOfConnectionAttempts += 1
			if numberOfConnectionAttempts > MaximumNumberOfParallelConnectionsAttempts {
				utils.LavaFormatError("Reached maximum number of parallel connections attempts, consider decreasing number of connections",
					nil,
					&map[string]string{"Number of parallel connections": strconv.FormatUint(uint64(nConns), 10),
						"Currently Connected": strconv.FormatUint(uint64(len(connector.freeClients)), 10),
					},
				)
				reachedClientLimit = true
				break
			}
			if ctx.Err() != nil {
				connector.Close()
				return nil
			}
			nctx, cancel := context.WithTimeout(ctx, DialTimeout)
			grpcClient, err = grpc.DialContext(nctx, addr, grpc.WithBlock(), grpc.WithTransportCredentials(insecure.NewCredentials()))
			if err != nil {
				utils.LavaFormatWarning("Could not connect to the client, retrying", err, nil)
				cancel()
				continue
			}
			cancel()
			break
		}
		connector.freeClients = append(connector.freeClients, grpcClient)
	}
	go connector.connectorLoop(ctx)
	return connector
}

func (connector *GRPCConnector) increaseNumberOfClients(ctx context.Context) {
	utils.LavaFormatInfo("No clients are free, increasing number of clients", nil)
	var grpcClient *grpc.ClientConn
	var err error
	for connectionAttempt := 0; connectionAttempt < MaximumNumberOfParallelConnectionsAttempts; connectionAttempt++ {
		nctx, cancel := context.WithTimeout(ctx, DialTimeout)
		grpcClient, err = grpc.DialContext(nctx, connector.addr, grpc.WithBlock(), grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			utils.LavaFormatWarning("increaseNumberOfClients, Could not connect to the node, retrying",
				err,
				&map[string]string{
					"Number Of Attempts": strconv.Itoa(connectionAttempt),
				})
			cancel()
			continue
		}
		cancel()

		connector.lock.Lock() // add connection to free list.
		defer connector.lock.Unlock()
		connector.freeClients = append(connector.freeClients, grpcClient)
		return
	}
}

func (connector *GRPCConnector) GetRpc(ctx context.Context, block bool) (*grpc.ClientConn, error) {
	connector.lock.Lock()
	defer connector.lock.Unlock()

	if len(connector.freeClients) <= connector.usedClients { // if we reached half of the free clients start creating new connections
		go connector.increaseNumberOfClients(ctx) // increase asynchronously the free list.
	}

	if len(connector.freeClients) == 0 {
		if !block {
			return nil, errors.New("out of clients")
		} else {
			for {
				connector.lock.Unlock()
				// if we reached 0 connections we need to create more connections
				// before sleeping, increase asynchronously the free list.
				go connector.increaseNumberOfClients(ctx)
				time.Sleep(50 * time.Millisecond)
				connector.lock.Lock()
				if len(connector.freeClients) != 0 {
					break
				}
			}
		}
	}

	ret := connector.freeClients[len(connector.freeClients)-1]
	connector.freeClients = connector.freeClients[:len(connector.freeClients)-1]
	connector.usedClients++

	return ret, nil
}

func (connector *GRPCConnector) ReturnRpc(rpc *grpc.ClientConn) {
	connector.lock.Lock()
	defer connector.lock.Unlock()

	connector.usedClients--
	if len(connector.freeClients) > (connector.usedClients + int(NumberOfParallelConnections) /* the number we started with */) {
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
