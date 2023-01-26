package rpcprovider

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strconv"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/protocol/chainlib"
	"github.com/lavanet/lava/protocol/rpcprovider/reliabilitymanager"
	"github.com/lavanet/lava/protocol/rpcprovider/rewardserver"
	"github.com/lavanet/lava/protocol/statetracker"
	"github.com/lavanet/lava/relayer/lavasession"
	"github.com/lavanet/lava/relayer/performance"
	"github.com/lavanet/lava/relayer/sigs"
	"github.com/lavanet/lava/utils"
	"github.com/spf13/viper"
)

const (
	EndpointsConfigName = "endpoints"
)

var (
	Yaml_config_properties = []string{"network-address", "chain-id", "api-interface", "node-url"}
	NumFieldsInConfig      = len(Yaml_config_properties)
)

type ProviderStateTrackerInf interface {
	RegisterProviderSessionManagerForEpochUpdates(ctx context.Context, providerSessionManager *lavasession.ProviderSessionManager)
	RegisterChainParserForSpecUpdates(ctx context.Context, chainParser chainlib.ChainParser)
	RegisterRewardServerForEpochUpdates(ctx context.Context, rewardServer rewardserver.RewardServer)
	RegisterReliabilityManagerForVoteUpdates(ctx context.Context, reliabilityManager reliabilitymanager.ReliabilityManager)
	QueryVerifyPairing(ctx context.Context, consumer string, blockHeight uint64)
}

type RPCProvider struct {
	providerStateTracker ProviderStateTrackerInf
	rpcProviderServers   map[string]*RPCProviderServer
}

func (rpcp *RPCProvider) Start(ctx context.Context, txFactory tx.Factory, clientCtx client.Context, rpcProviderEndpoints []*lavasession.RPCProviderEndpoint, cache *performance.Cache) (err error) {
	providerStateTracker := statetracker.ProviderStateTracker{}
	rpcp.providerStateTracker, err = providerStateTracker.New(ctx, txFactory, clientCtx)
	if err != nil {
		return err
	}
	rpcp.rpcProviderServers = make(map[string]*RPCProviderServer, len(rpcProviderEndpoints))

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
	utils.LavaFormatInfo("RPCProvider pubkey: "+addr.String(), nil)
	utils.LavaFormatInfo("RPCProvider setting up endpoints", &map[string]string{"length": strconv.Itoa(len(rpcProviderEndpoints))})
	for _, rpcProviderEndpoint := range rpcProviderEndpoints {
		providerSessionManager := lavasession.NewProviderSessionManager(rpcProviderEndpoint, &providerStateTracker)
		key := rpcProviderEndpoint.Key()
		rpcp.providerStateTracker.RegisterProviderSessionManagerForEpochUpdates(ctx, providerSessionManager)
		chainParser, err := chainlib.NewChainParser(rpcProviderEndpoint.ApiInterface)
		if err != nil {
			return err
		}
		providerStateTracker.RegisterChainParserForSpecUpdates(ctx, chainParser)

		rewardServer := rewardserver.RewardServer{}
		providerStateTracker.RegisterRewardServerForEpochUpdates(ctx, rewardServer)

		reliabilityManager := reliabilitymanager.ReliabilityManager{}
		providerStateTracker.RegisterReliabilityManagerForVoteUpdates(ctx, reliabilityManager)

		rpcp.rpcProviderServers[key] = &RPCProviderServer{}
		utils.LavaFormatInfo("RPCProvider Listening", &map[string]string{"endpoints": lavasession.PrintRPCProviderEndpoint(rpcProviderEndpoint)})
		_ = privKey
		// rpcp.rpcProviderServers[key].ServeRPCRequests(ctx, rpcEndpoint, rpcp.providerStateTracker, chainParser, finalizationConsensus, providerSessionManager, requiredResponses, privKey, vrf_sk, cache)
	}

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)
	<-signalChan
	return nil
}

func ParseEndpoints(viper_endpoints *viper.Viper, geolocation uint64) (endpoints []*lavasession.RPCProviderEndpoint, err error) {
	err = viper_endpoints.UnmarshalKey(EndpointsConfigName, &endpoints)
	if err != nil {
		utils.LavaFormatFatal("could not unmarshal endpoints", err, &map[string]string{"viper_endpoints": fmt.Sprintf("%v", viper_endpoints.AllSettings())})
	}
	for _, endpoint := range endpoints {
		endpoint.Geolocation = geolocation
	}
	return
}
