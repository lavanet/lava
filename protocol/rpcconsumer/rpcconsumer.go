package rpcconsumer

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"github.com/coniks-sys/coniks-go/crypto/vrf"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/lavanet/lava/protocol/chainlib"
	"github.com/lavanet/lava/protocol/lavaprotocol"
	"github.com/lavanet/lava/protocol/statetracker"
	"github.com/lavanet/lava/relayer/lavasession"
	"github.com/lavanet/lava/relayer/performance"
	"github.com/lavanet/lava/relayer/sigs"
	"github.com/lavanet/lava/utils"
	pairingtypes "github.com/lavanet/lava/x/pairing/types"
)

type ConsumerStateTrackerInf interface {
	RegisterConsumerSessionManagerForPairingUpdates(ctx context.Context, consumerSessionManager *lavasession.ConsumerSessionManager)
	RegisterApiParserForSpecUpdates(ctx context.Context, chainParser chainlib.ChainParser)
	ReportProviderForFinalizationData(context.Context, *pairingtypes.RelayReply)
	RegisterFinalizationConsensusForUpdates(context.Context, *lavaprotocol.FinalizationConsensus)
}

type RPCConsumer struct {
	consumerSessionManagers map[string]*lavasession.ConsumerSessionManager
	consumerStateTracker    ConsumerStateTrackerInf
	rpcConsumerServers      map[string]*RPCConsumerServer
}

// spawns a new RPCConsumer server with all it's processes and internals ready for communications
func (rpcc *RPCConsumer) Start(ctx context.Context, txFactory tx.Factory, clientCtx client.Context, rpcEndpoints []*lavasession.RPCEndpoint, requiredResponses int, vrf_sk vrf.PrivateKey) (err error) {
	// spawn up ConsumerStateTracker
	consumerStateTracker := statetracker.ConsumerStateTracker{}
	rpcc.consumerStateTracker, err = consumerStateTracker.New(ctx, txFactory, clientCtx)
	if err != nil {
		return err
	}
	rpcc.consumerSessionManagers = make(map[string]*lavasession.ConsumerSessionManager, len(rpcEndpoints))
	rpcc.rpcConsumerServers = make(map[string]*RPCConsumerServer, len(rpcEndpoints))

	publicAddress := ""           //TODO
	cache := &performance.Cache{} //TODO

	keyName, err := sigs.GetKeyName(clientCtx)
	if err != nil {
		utils.LavaFormatFatal("failed getting key name from clientCtx", err, nil)

	}
	privKey, err := sigs.GetPrivKey(clientCtx, keyName)
	if err != nil {
		utils.LavaFormatFatal("failed getting private key from key name", err, &map[string]string{"keyName": keyName})
	}
	clientKey, _ := clientCtx.Keyring.Key(keyName)

	utils.LavaFormatInfo("RPCConsumer pubkey: "+fmt.Sprintf("%s", clientKey.GetPubKey().Address()), nil)

	for _, rpcEndpoint := range rpcEndpoints {
		// validate uniqueness of endpoint
		// create ConsumerSessionManager for each endpoint
		consumerSessionManager := lavasession.NewConsumerSessionManager(rpcEndpoint)
		key := rpcEndpoint.Key()
		rpcc.consumerSessionManagers[key] = consumerSessionManager
		rpcc.consumerStateTracker.RegisterConsumerSessionManagerForPairingUpdates(ctx, rpcc.consumerSessionManagers[key])
		rpcc.rpcConsumerServers[key] = &RPCConsumerServer{}
		rpcc.rpcConsumerServers[key].ServeRPCRequests(ctx, rpcEndpoint, rpcc.consumerStateTracker, rpcc.consumerSessionManagers[key], publicAddress, requiredResponses, privKey, vrf_sk, cache)
	}

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)
	<-signalChan
	return nil
}
