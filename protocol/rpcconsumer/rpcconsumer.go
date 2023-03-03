package rpcconsumer

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strconv"

	"github.com/coniks-sys/coniks-go/crypto/vrf"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/protocol/chainlib"
	"github.com/lavanet/lava/protocol/common"
	"github.com/lavanet/lava/protocol/lavaprotocol"
	"github.com/lavanet/lava/protocol/lavasession"
	"github.com/lavanet/lava/protocol/statetracker"
	"github.com/lavanet/lava/relayer/performance"
	"github.com/lavanet/lava/relayer/sigs"
	"github.com/lavanet/lava/utils"
	conflicttypes "github.com/lavanet/lava/x/conflict/types"
	"github.com/spf13/viper"
)

var Yaml_config_properties = []string{"network-address", "chain-id", "api-interface"}

type ConsumerStateTrackerInf interface {
	RegisterConsumerSessionManagerForPairingUpdates(ctx context.Context, consumerSessionManager *lavasession.ConsumerSessionManager)
	RegisterChainParserForSpecUpdates(ctx context.Context, chainParser chainlib.ChainParser, chainID string) error
	RegisterFinalizationConsensusForUpdates(context.Context, *lavaprotocol.FinalizationConsensus)
	TxConflictDetection(ctx context.Context, finalizationConflict *conflicttypes.FinalizationConflict, responseConflict *conflicttypes.ResponseConflict, sameProviderConflict *conflicttypes.FinalizationConflict) error
}

type RPCConsumer struct {
	consumerStateTracker ConsumerStateTrackerInf
	rpcConsumerServers   map[string]*RPCConsumerServer
}

// spawns a new RPCConsumer server with all it's processes and internals ready for communications
func (rpcc *RPCConsumer) Start(ctx context.Context, txFactory tx.Factory, clientCtx client.Context, rpcEndpoints []*lavasession.RPCEndpoint, requiredResponses int, vrf_sk vrf.PrivateKey, cache *performance.Cache) (err error) {
	// spawn up ConsumerStateTracker
	lavaChainFetcher := chainlib.NewLavaChainFetcher(ctx, clientCtx)
	consumerStateTracker, err := statetracker.NewConsumerStateTracker(ctx, txFactory, clientCtx, lavaChainFetcher)
	if err != nil {
		return err
	}
	rpcc.consumerStateTracker = consumerStateTracker
	rpcc.rpcConsumerServers = make(map[string]*RPCConsumerServer, len(rpcEndpoints))

	keyName, err := sigs.GetKeyName(clientCtx)
	if err != nil {
		utils.LavaFormatFatal("failed getting key name from clientCtx", err, nil)
	}
	privKey, err := sigs.GetPrivKey(clientCtx, keyName)
	if err != nil {
		utils.LavaFormatFatal("failed getting private key from key name", err, &map[string]string{"keyName": keyName})
	}
	clientKey, _ := clientCtx.Keyring.Key(keyName)

	var addr sdk.AccAddress
	err = addr.Unmarshal(clientKey.GetPubKey().Address())
	if err != nil {
		utils.LavaFormatFatal("failed unmarshaling public address", err, &map[string]string{"keyName": keyName, "pubkey": clientKey.GetPubKey().Address().String()})
	}
	utils.LavaFormatInfo("RPCConsumer pubkey: "+addr.String(), nil)
	utils.LavaFormatInfo("RPCConsumer setting up endpoints", &map[string]string{"length": strconv.Itoa(len(rpcEndpoints))})
	for _, rpcEndpoint := range rpcEndpoints {
		consumerSessionManager := lavasession.NewConsumerSessionManager(rpcEndpoint)
		key := rpcEndpoint.Key()
		rpcc.consumerStateTracker.RegisterConsumerSessionManagerForPairingUpdates(ctx, consumerSessionManager)
		chainParser, err := chainlib.NewChainParser(rpcEndpoint.ApiInterface)
		if err != nil {
			return err
		}
		err = consumerStateTracker.RegisterChainParserForSpecUpdates(ctx, chainParser, rpcEndpoint.ChainID)
		if err != nil {
			return err
		}
		finalizationConsensus := &lavaprotocol.FinalizationConsensus{}
		consumerStateTracker.RegisterFinalizationConsensusForUpdates(ctx, finalizationConsensus)
		rpcc.rpcConsumerServers[key] = &RPCConsumerServer{}
		utils.LavaFormatInfo("RPCConsumer Listening", &map[string]string{"endpoints": rpcEndpoint.String()})
		rpcc.rpcConsumerServers[key].ServeRPCRequests(ctx, rpcEndpoint, rpcc.consumerStateTracker, chainParser, finalizationConsensus, consumerSessionManager, requiredResponses, privKey, vrf_sk, cache)
	}

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)
	<-signalChan
	return nil
}

func ParseEndpoints(viper_endpoints *viper.Viper, geolocation uint64) (endpoints []*lavasession.RPCEndpoint, err error) {
	err = viper_endpoints.UnmarshalKey(common.EndpointsConfigName, &endpoints)
	if err != nil {
		utils.LavaFormatFatal("could not unmarshal endpoints", err, &map[string]string{"viper_endpoints": fmt.Sprintf("%v", viper_endpoints.AllSettings())})
	}
	for _, endpoint := range endpoints {
		endpoint.Geolocation = geolocation
	}
	return
}
