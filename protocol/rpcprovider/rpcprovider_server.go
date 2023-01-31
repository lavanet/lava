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
	SendNewProof(ctx context.Context, singleProviderSession *lavasession.SingleProviderSession, epoch uint64, consumerAddr string)
}

func (rpcps *RPCProviderServer) ServeRPCRequests(
	ctx context.Context, rpcProviderEndpoint *lavasession.RPCProviderEndpoint,
	chainParser chainlib.ChainParser,
	rewardServer RewardServerInf,
	providerSessionManager *lavasession.ProviderSessionManager,
	reliabilityManager ReliabilityManagerInf,
	privKey *btcec.PrivateKey,
	cache *performance.Cache, chainProxy chainlib.ChainProxy) {
	// spin up a grpc listener
	// verify the relay metadata is valid (epoch, signature)
	// verify the consumer is authorised
	// create/bring a session
	// verify the relay data is valid (cu, chainParser, requested block)
	// check cache hit
	// send the relay to the node using chainProxy
	// set cache entry (async)
	// attach data reliability finalization data
	// sign the response
	// send the proof to reward server
	// finalize the session
}
