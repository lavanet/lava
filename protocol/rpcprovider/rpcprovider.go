package rpcprovider

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"time"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/protocol/chainlib"
	"github.com/lavanet/lava/protocol/chaintracker"
	"github.com/lavanet/lava/protocol/common"
	"github.com/lavanet/lava/protocol/lavasession"
	"github.com/lavanet/lava/protocol/rpcprovider/reliabilitymanager"
	"github.com/lavanet/lava/protocol/rpcprovider/rewardserver"
	"github.com/lavanet/lava/protocol/statetracker"
	"github.com/lavanet/lava/relayer/performance"
	"github.com/lavanet/lava/relayer/sigs"
	"github.com/lavanet/lava/utils"
	pairingtypes "github.com/lavanet/lava/x/pairing/types"
	"github.com/spf13/viper"
)

const (
	ChainTrackerDefaultMemory = 100
)

var (
	Yaml_config_properties = []string{"network-address", "chain-id", "api-interface", "node-url"}
)

type ProviderStateTrackerInf interface {
	RegisterChainParserForSpecUpdates(ctx context.Context, chainParser chainlib.ChainParser, chainID string) error
	RegisterReliabilityManagerForVoteUpdates(ctx context.Context, voteUpdatable statetracker.VoteUpdatable, endpointP *lavasession.RPCProviderEndpoint)
	RegisterForEpochUpdates(ctx context.Context, epochUpdatable statetracker.EpochUpdatable)
	TxRelayPayment(ctx context.Context, relayRequests []*pairingtypes.RelayRequest, description string) error
	SendVoteReveal(voteID string, vote *reliabilitymanager.VoteData) error
	SendVoteCommitment(voteID string, vote *reliabilitymanager.VoteData) error
	LatestBlock() int64
	GetVrfPkAndMaxCuForUser(ctx context.Context, consumerAddress string, chainID string, epocu uint64) (vrfPk *utils.VrfPubKey, maxCu uint64, err error)
	VerifyPairing(ctx context.Context, consumerAddress string, providerAddress string, epoch uint64, chainID string) (valid bool, index int64, err error)
	GetProvidersCountForConsumer(ctx context.Context, consumerAddress string, epoch uint64, chainID string) (uint32, error)
	GetEpochSize(ctx context.Context) (uint64, error)
	EarliestBlockInMemory(ctx context.Context) (uint64, error)
	RegisterPaymentUpdatableForPayments(ctx context.Context, paymentUpdatable statetracker.PaymentUpdatable)
	GetRecommendedEpochNumToCollectPayment(ctx context.Context) (uint64, error)
	GetEpochSizeMultipliedByRecommendedEpochNumToCollectPayment(ctx context.Context) (uint64, error)
}

type RPCProvider struct {
	providerStateTracker ProviderStateTrackerInf
	rpcProviderServers   map[string]*RPCProviderServer
	rpcProviderListeners map[string]*ProviderListener
}

func (rpcp *RPCProvider) Start(ctx context.Context, txFactory tx.Factory, clientCtx client.Context, rpcProviderEndpoints []*lavasession.RPCProviderEndpoint, cache *performance.Cache, parallelConnections uint) (err error) {
	ctx, cancel := context.WithCancel(ctx)
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)
	defer func() {
		signal.Stop(signalChan)
		cancel()
	}()
	rpcp.rpcProviderServers = make(map[string]*RPCProviderServer)
	rpcp.rpcProviderListeners = make(map[string]*ProviderListener)
	// single state tracker
	lavaChainFetcher := chainlib.NewLavaChainFetcher(ctx, clientCtx)
	providerStateTracker, err := statetracker.NewProviderStateTracker(ctx, txFactory, clientCtx, lavaChainFetcher)
	if err != nil {
		return err
	}
	rpcp.providerStateTracker = providerStateTracker
	rpcp.rpcProviderServers = make(map[string]*RPCProviderServer, len(rpcProviderEndpoints))
	// single reward server
	rewardServer := rewardserver.NewRewardServer(providerStateTracker)
	rpcp.providerStateTracker.RegisterForEpochUpdates(ctx, rewardServer)
	rpcp.providerStateTracker.RegisterPaymentUpdatableForPayments(ctx, rewardServer)
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
	blockMemorySize, err := rpcp.providerStateTracker.GetEpochSizeMultipliedByRecommendedEpochNumToCollectPayment(ctx) // get the number of blocks to keep in PSM.
	if err != nil {
		utils.LavaFormatFatal("Failed fetching GetEpochSizeMultipliedByRecommendedEpochNumToCollectPayment in RPCProvider Start", err, nil)
	}
	for _, rpcProviderEndpoint := range rpcProviderEndpoints {
		providerSessionManager := lavasession.NewProviderSessionManager(rpcProviderEndpoint, blockMemorySize)
		key := rpcProviderEndpoint.Key()
		rpcp.providerStateTracker.RegisterForEpochUpdates(ctx, providerSessionManager)
		chainParser, err := chainlib.NewChainParser(rpcProviderEndpoint.ApiInterface)
		if err != nil {
			return err
		}
		providerStateTracker.RegisterChainParserForSpecUpdates(ctx, chainParser, rpcProviderEndpoint.ChainID)
		_, averageBlockTime, _, _ := chainParser.ChainBlockStats()
		chainProxy, err := chainlib.GetChainProxy(ctx, parallelConnections, rpcProviderEndpoint, averageBlockTime)
		if err != nil {
			utils.LavaFormatError("panic severity critical error, failed creating chain proxy, continuing with others", err, &map[string]string{"parallelConnections": strconv.FormatUint(uint64(parallelConnections), 10), "rpcProviderEndpoint": fmt.Sprintf("%+v", rpcProviderEndpoint)})
			continue
		}

		_, averageBlockTime, blocksToFinalization, blocksInFinalizationData := chainParser.ChainBlockStats()
		blocksToSaveChainTracker := uint64(blocksToFinalization + blocksInFinalizationData)
		chainTrackerConfig := chaintracker.ChainTrackerConfig{
			BlocksToSave:      blocksToSaveChainTracker,
			AverageBlockTime:  averageBlockTime,
			ServerBlockMemory: ChainTrackerDefaultMemory + blocksToSaveChainTracker,
		}
		chainFetcher := chainlib.NewChainFetcher(ctx, chainProxy, chainParser, rpcProviderEndpoint)
		chainTracker, err := chaintracker.NewChainTracker(ctx, chainFetcher, chainTrackerConfig)
		if err != nil {
			utils.LavaFormatError("panic severity critical error, aborting support for chain due to node access, continuing with others", err, &map[string]string{"chainTrackerConfig": fmt.Sprintf("%+v", chainTrackerConfig), "endpoint": rpcProviderEndpoint.String()})
			continue
		}
		reliabilityManager := reliabilitymanager.NewReliabilityManager(chainTracker, providerStateTracker, addr.String(), chainProxy, chainParser)
		providerStateTracker.RegisterReliabilityManagerForVoteUpdates(ctx, reliabilityManager, rpcProviderEndpoint)

		rpcProviderServer := &RPCProviderServer{}
		if _, ok := rpcp.rpcProviderServers[key]; ok {
			utils.LavaFormatFatal("Trying to add the same key twice to rpcProviderServers check config file.", nil,
				&map[string]string{"key": key})
		}
		rpcp.rpcProviderServers[key] = rpcProviderServer
		rpcProviderServer.ServeRPCRequests(ctx, rpcProviderEndpoint, chainParser, rewardServer, providerSessionManager, reliabilityManager, privKey, cache, chainProxy, providerStateTracker, addr)

		// set up grpc listener
		var listener *ProviderListener
		if rpcProviderEndpoint.NetworkAddress == "" && len(rpcp.rpcProviderListeners) > 0 {
			// handle case only one network address was defined
			for _, listener_p := range rpcp.rpcProviderListeners {
				listener = listener_p
				break
			}
		} else {
			var ok bool
			listener, ok = rpcp.rpcProviderListeners[rpcProviderEndpoint.NetworkAddress]
			if !ok {
				listener = NewProviderListener(ctx, rpcProviderEndpoint.NetworkAddress)
				rpcp.rpcProviderListeners[listener.Key()] = listener
			}
		}
		if listener == nil {
			utils.LavaFormatFatal("listener not defined, cant register RPCProviderServer", nil, &map[string]string{"RPCProviderEndpoint": rpcProviderEndpoint.String()})
		}
		listener.RegisterReceiver(rpcProviderServer, rpcProviderEndpoint)
	}

	select {
	case <-ctx.Done():
		utils.LavaFormatInfo("Provider Server ctx.Done", nil)
	case <-signalChan:
		utils.LavaFormatInfo("Provider Server signalChan", nil)
	}

	for _, listener := range rpcp.rpcProviderListeners {
		shutdownCtx, shutdownRelease := context.WithTimeout(context.Background(), 10*time.Second)
		listener.Shutdown(shutdownCtx)
		defer shutdownRelease()
	}

	return nil
}

func ParseEndpoints(viper_endpoints *viper.Viper, geolocation uint64) (endpoints []*lavasession.RPCProviderEndpoint, err error) {
	err = viper_endpoints.UnmarshalKey(common.EndpointsConfigName, &endpoints)
	if err != nil {
		utils.LavaFormatFatal("could not unmarshal endpoints", err, &map[string]string{"viper_endpoints": fmt.Sprintf("%v", viper_endpoints.AllSettings())})
	}
	for _, endpoint := range endpoints {
		endpoint.Geolocation = geolocation
	}
	return
}
