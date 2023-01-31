package statetracker

import (
	"context"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/lavanet/lava/protocol/chainlib"
	"github.com/lavanet/lava/protocol/rpcprovider/reliabilitymanager"
	"github.com/lavanet/lava/relayer/lavasession"
	pairingtypes "github.com/lavanet/lava/x/pairing/types"
)

// ProviderStateTracker PST is a class for tracking provider data from the lava blockchain, such as epoch changes.
// it allows also to query specific data form the blockchain and acts as a single place to send transactions
type ProviderStateTracker struct {
	// TODO: embed stateTracker
}

func (pst *ProviderStateTracker) New(ctx context.Context, txFactory tx.Factory, clientCtx client.Context) (ret *ProviderStateTracker, err error) {
	// set up StateQuery
	// Spin up chain tracker on the lava node, its address is in the --node flag (or its default), on new block call to newLavaBlock
	// use StateQuery to get the lava spec and spin up the chain tracker with the right params
	// set up txSender the same way
	return pst, nil
}

func (pst *ProviderStateTracker) RegisterProviderSessionManagerForEpochUpdates(ctx context.Context, providerSessionManager *lavasession.ProviderSessionManager) {
	// TODO: change to an interface instead of lavasession.ProviderSessionManager
	// create an epoch updater
	// add epoch updater to the updater map

	return
}

func (pst *ProviderStateTracker) RegisterChainParserForSpecUpdates(ctx context.Context, chainParser chainlib.ChainParser) {
	// can be moved to base class
	return
}

func (pst *ProviderStateTracker) RegisterReliabilityManagerForVoteUpdates(ctx context.Context, reliabilityManager *reliabilitymanager.ReliabilityManager) {
	// TODO: change to an interface instead of reliabilitymanager.ReliabilityManager
	return
}

func (pst *ProviderStateTracker) QueryVerifyPairing(ctx context.Context, consumer string, blockHeight uint64) {
	// TODO: implement
	return
}

func (pst *ProviderStateTracker) TxRelayPayment(ctx context.Context, relayRequests []*pairingtypes.RelayRequest) {
	// TODO: implement
	return
}
