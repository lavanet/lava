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

const DialTimeout = 500 * time.Millisecond
const ParallelConnectionsFlag = "parallel-connections"
const DefaultNumberOfParallelConnections = 10
const MaximumNumberOfParallelConnectionsAttempts = 10

type Connector struct {
	lock        utils.LavaMutex
	freeClients []*rpcclient.Client
	usedClients int
}

func NewConnector(ctx context.Context, nConns uint, addr string) *Connector {
	connector := &Connector{
		freeClients: make([]*rpcclient.Client, 0, nConns),
	}

	for i := uint(0); i < nConns; i++ {
		var rpcClient *rpcclient.Client
		var err error
		numberOfConnectionAttempts := 0
		for {
			numberOfConnectionAttempts += 1
			if numberOfConnectionAttempts > MaximumNumberOfParallelConnectionsAttempts {
				utils.LavaFormatFatal("Reached maximum number of parallel connections attempts, consider decreasing number of connections",
					nil,
					&map[string]string{"Number of parallel connections": strconv.FormatUint(uint64(nConns), 10)},
				)
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

func (connector *Connector) GetRpc(block bool) (*rpcclient.Client, error) {
	connector.lock.Lock()
	defer connector.lock.Unlock()

	if len(connector.freeClients) == 0 {
		if !block {
			return nil, errors.New("out of clients")
		} else {
			for {
				connector.lock.Unlock()
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
	connector.freeClients = append(connector.freeClients, rpc)
}

type GRPCConnector struct {
	lock        sync.RWMutex
	freeClients []*grpc.ClientConn
	usedClients int
}

func NewGRPCConnector(ctx context.Context, nConns uint, addr string) *GRPCConnector {
	connector := &GRPCConnector{
		freeClients: make([]*grpc.ClientConn, 0, nConns),
	}

	for i := uint(0); i < nConns; i++ {
		var grpcClient *grpc.ClientConn
		var err error
		numberOfConnectionAttempts := 0
		for {
			numberOfConnectionAttempts += 1
			if numberOfConnectionAttempts > MaximumNumberOfParallelConnectionsAttempts {
				utils.LavaFormatFatal("Reached maximum number of parallel connections attempts, consider decreasing number of connections",
					nil,
					&map[string]string{"Number of parallel connections": strconv.FormatUint(uint64(nConns), 10)},
				)
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

func (connector *GRPCConnector) GetRpc(block bool) (*grpc.ClientConn, error) {
	connector.lock.Lock()
	defer connector.lock.Unlock()

	if len(connector.freeClients) == 0 {
		if !block {
			return nil, errors.New("out of clients")
		} else {
			for {
				connector.lock.Unlock()
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
