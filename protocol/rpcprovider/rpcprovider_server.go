package rpcprovider

import (
	"context"

	"github.com/btcsuite/btcd/btcec"
	"github.com/lavanet/lava/protocol/chainlib"
	"github.com/lavanet/lava/protocol/chaintracker"
	"github.com/lavanet/lava/relayer/lavasession"
	"github.com/lavanet/lava/relayer/performance"
)

type RPCProviderServer struct {
}

type ReliabilityManagerInf interface {
	GetLatestBlockData(fromBlock int64, toBlock int64, specificBlock int64) (latestBlock int64, requestedHashes []*chaintracker.BlockStore, err error)
	GetLatestBlockNum() int64
}

type RewardServerInf interface {
	ClaimRewardsForProviderSessions(ctx context.Context, providerSessionsWithConsumer *lavasession.ProviderSessionsWithConsumer, epoch uint64, consumerAddr string)
}

func (rpcps *RPCProviderServer) ServeRPCRequests(
	ctx context.Context, rpcProviderEndpoint *lavasession.RPCProviderEndpoint,
	chainParser chainlib.ChainParser,
	rewardServer RewardServerInf,
	providerSessionManager *lavasession.ProviderSessionManager,
	reliabilityManager ReliabilityManagerInf,
	privKey *btcec.PrivateKey,
	cache *performance.Cache, chainProxy chainlib.ChainProxy) {

}
