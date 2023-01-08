package consumerstatetracker

import (
	"context"
	"sync"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/tx"
	chaintracker "github.com/lavanet/lava/protocol/chainTracker"
	"github.com/lavanet/lava/protocol/rpcconsumer/apilib"
	"github.com/lavanet/lava/relayer/lavasession"
	spectypes "github.com/lavanet/lava/x/spec/types"
)

// ConsumerStateTracker CSTis a class for tracking consumer data from the lava blockchain, such as epoch changes.
// it allows also to query specific data form the blockchain and acts as a single place to send transactions
type ConsumerStateTracker struct {
	chainTracker     *chaintracker.ChainTracker
	StateQuery       *StateQuery
	TxSender         *TxSender
	registrationLock sync.RWMutex
}

func (cst *ConsumerStateTracker) New(ctx context.Context, txFactory tx.Factory, clientCtx client.Context) (ret *ConsumerStateTracker, err error) {
	// set up StateQuery
	// Spin up chain tracker on the lava node, its address is in the --node flag (or its default), on new block call to newLavaBlock
	// use StateQuery to get the lava spec and spin up the chain tracker with the right params
	// set up txSender the same way
	stateQuery := StateQuery{}
	cst.StateQuery, err = stateQuery.New(ctx, clientCtx)
	if err != nil {
		return nil, err
	}

	txSender := TxSender{}
	cst.TxSender, err = txSender.New(ctx, txFactory, clientCtx)
	if err != nil {
		return nil, err
	}
	return cst, nil
}

func (cst *ConsumerStateTracker) newLavaBlock(latestBlock int64) {
	// go over the registered callbacks and call them
	cst.registrationLock.RLock()
	defer cst.registrationLock.RUnlock()
}

func (cst *ConsumerStateTracker) RegisterConsumerSessionManagerForPairingUpdates(ctx context.Context, consumerSessionManager *lavasession.ConsumerSessionManager) {
	// register this CSM to get the updated pairing list when a new epoch starts
	// add the necessary data to perform the getPairing query
	cst.registrationLock.Lock()
	defer cst.registrationLock.Unlock()
}

func (cst *ConsumerStateTracker) RegisterApiParserForSpecUpdates(ctx context.Context, apiParser apilib.APIParser) {
	// register this apiParser for spec updates
	// currently just set the first one, and have a TODO to handle spec changes
	// get the spec and set it into the apiParser
	spec := spectypes.Spec{}
	apiParser.SetSpec(spec)
}
