package rpcconsumer

import (
	"context"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/lavanet/lava/protocol/consumerstatetracker"
	"github.com/lavanet/lava/relayer/lavasession"
)

type RPCConsumer struct {
	consumerSessionManagers map[string]*lavasession.ConsumerSessionManager
	consumerStateTracker    *consumerstatetracker.ConsumerStateTracker
	rpcConsumerServers      map[string]*RPCConsumerServer
}

// spawns a new RPCConsumer server with all it's processes and internals ready for communications
func (rpcc *RPCConsumer) Start(ctx context.Context, txFactory tx.Factory, clientCtx client.Context, rpcEndpoints []*lavasession.RPCEndpoint) (err error) {
	// spawn up ConsumerStateTracker
	consumerStateTracker := consumerstatetracker.ConsumerStateTracker{}
	rpcc.consumerStateTracker, err = consumerStateTracker.New(ctx, txFactory, clientCtx)
	if err != nil {
		return err
	}
	rpcc.consumerSessionManagers = make(map[string]*lavasession.ConsumerSessionManager, len(rpcEndpoints))
	rpcc.rpcConsumerServers = make(map[string]*RPCConsumerServer, len(rpcEndpoints))

	for _, rpcEndpoint := range rpcEndpoints {
		// validate uniqueness of endpoint
		// create ConsumerSessionManager for each endpoint
		consumerSessionManager := lavasession.ConsumerSessionManager{}
		key := rpcEndpoint.Key()
		rpcc.consumerSessionManagers[key] = &consumerSessionManager
		rpcc.consumerStateTracker.RegisterConsumerSessionManagerForPairingUpdates(ctx, rpcc.consumerSessionManagers[key])
		rpcc.rpcConsumerServers[key] = &RPCConsumerServer{}
		rpcc.rpcConsumerServers[key].ServeRPCRequests(ctx, rpcEndpoint, rpcc.consumerStateTracker, rpcc.consumerSessionManagers[key])
	}

	return nil
}
