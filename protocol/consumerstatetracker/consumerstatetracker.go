package consumerstatetracker

import (
	"context"
	"sync"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
	chaintracker "github.com/lavanet/lava/protocol/chainTracker"
	"github.com/lavanet/lava/protocol/rpcconsumer/apilib"
	"github.com/lavanet/lava/relayer/lavasession"
	spectypes "github.com/lavanet/lava/x/spec/types"
)

const (
	CallbackKeyForPairingUpdate = "pairing-update"
)

// ConsumerStateTracker CSTis a class for tracking consumer data from the lava blockchain, such as epoch changes.
// it allows also to query specific data form the blockchain and acts as a single place to send transactions
type ConsumerStateTracker struct {
	consumerAddress                   sdk.AccAddress
	chainTracker                      *chaintracker.ChainTracker
	StateQuery                        *StateQuery
	TxSender                          *TxSender
	registrationLock                  sync.RWMutex
	newLavaBlockCallbacks             map[string]func(int64)
	registeredConsumerSessionManagers []*lavasession.ConsumerSessionManager
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
	cst.registeredConsumerSessionManagers = []*lavasession.ConsumerSessionManager{}
	cst.consumerAddress = clientCtx.FromAddress
	return cst, nil
}

func (cst *ConsumerStateTracker) newLavaBlock(latestBlock int64) {
	// go over the registered callbacks and call them if relevant
	cst.registrationLock.RLock()
	defer cst.registrationLock.RUnlock()
}

func (cst *ConsumerStateTracker) updatePairingForRegistered(block int64) {
	// check if we need to update pairing (rule for pairing update - overlap)
	// add the necessary data to perform the getPairing query
	// when updating go over the endpoints and make sure only the right endpoints in terms of geolocation and apiInterfaces are sent
	// get the pairing data
	// to over registered consumerSessionManagers fetch their RPCEndpoint
	// create the consumerSessionWithProvider list for each of the consumer session managers accordig to its requirements and RPCEndpoint
	// call the function to update pairing
}

func (cst *ConsumerStateTracker) RegisterConsumerSessionManagerForPairingUpdates(ctx context.Context, consumerSessionManager *lavasession.ConsumerSessionManager) {
	// register this CSM to get the updated pairing list when a new epoch starts
	cst.registrationLock.Lock()
	defer cst.registrationLock.Unlock()
	// make sure new lava block exists as a callback in stateTracker
	// add updatePairingForRegistered as a callback on a new block
	if _, ok := cst.newLavaBlockCallbacks[CallbackKeyForPairingUpdate]; !ok {
		cst.newLavaBlockCallbacks[CallbackKeyForPairingUpdate] = cst.updatePairingForRegistered
	}
	// add consumerSessionManager to the list of consumerSessionManagers that need an update
	cst.registeredConsumerSessionManagers = append(cst.registeredConsumerSessionManagers, consumerSessionManager)
}

func (cst *ConsumerStateTracker) RegisterApiParserForSpecUpdates(ctx context.Context, apiParser apilib.APIParser) {
	// register this apiParser for spec updates
	// currently just set the first one, and have a TODO to handle spec changes
	// get the spec and set it into the apiParser
	spec := spectypes.Spec{}
	apiParser.SetSpec(spec)
}
