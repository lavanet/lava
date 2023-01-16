package consumerstatetracker

import (
	"context"
	"fmt"
	"sync"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/protocol/apilib"
	chaintracker "github.com/lavanet/lava/protocol/chainTracker"
	"github.com/lavanet/lava/relayer/lavasession"
	"github.com/lavanet/lava/utils"
	pairingtypes "github.com/lavanet/lava/x/pairing/types"
	spectypes "github.com/lavanet/lava/x/spec/types"
)

// ConsumerStateTracker CSTis a class for tracking consumer data from the lava blockchain, such as epoch changes.
// it allows also to query specific data form the blockchain and acts as a single place to send transactions
type ConsumerStateTracker struct {
	consumerAddress      sdk.AccAddress
	chainTracker         *chaintracker.ChainTracker
	StateQuery           *StateQuery
	TxSender             *TxSender
	registrationLock     sync.RWMutex
	newLavaBlockUpdaters map[string]Updater
}

type Updater interface {
	Update(int64)
	UpdaterKey() string
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
	cst.consumerAddress = clientCtx.FromAddress
	return cst, nil
}

func (cst *ConsumerStateTracker) newLavaBlock(latestBlock int64) {
	// go over the registered updaters and trigger update
	cst.registrationLock.RLock()
	defer cst.registrationLock.RUnlock()
	for _, updater := range cst.newLavaBlockUpdaters {
		updater.Update(latestBlock)
	}
}

func (cst *ConsumerStateTracker) RegisterConsumerSessionManagerForPairingUpdates(ctx context.Context, consumerSessionManager *lavasession.ConsumerSessionManager) {
	// register this CSM to get the updated pairing list when a new epoch starts
	cst.registrationLock.Lock()
	defer cst.registrationLock.Unlock()
	// make sure new lava block exists as a callback in stateTracker
	// add updatePairingForRegistered as a callback on a new block

	var pairingUpdater *PairingUpdater = nil // UpdaterKey is nil safe
	pairingUpdater_raw, ok := cst.newLavaBlockUpdaters[pairingUpdater.UpdaterKey()]
	if !ok {
		pairingUpdater = NewPairingUpdater(cst.consumerAddress, cst.StateQuery)
		cst.newLavaBlockUpdaters[pairingUpdater.UpdaterKey()] = pairingUpdater
	}
	pairingUpdater, ok = pairingUpdater_raw.(*PairingUpdater)
	if !ok {
		utils.LavaFormatFatal("invalid_updater_key", nil, &map[string]string{"updaters_map": fmt.Sprintf("%+v", cst.newLavaBlockUpdaters)})
	}
	pairingUpdater.RegisterPairing(consumerSessionManager)
}

func (cst *ConsumerStateTracker) RegisterApiParserForSpecUpdates(ctx context.Context, apiParser apilib.APIParser) {
	// register this apiParser for spec updates
	// currently just set the first one, and have a TODO to handle spec changes
	// get the spec and set it into the apiParser
	spec := spectypes.Spec{}
	apiParser.SetSpec(spec)
}

func (cst *ConsumerStateTracker) ReportProviderForFinalizationData(ctx context.Context, reply *pairingtypes.RelayReply) {
	//TODO: implement
	utils.LavaFormatDebug("reporting provider for wrong finalization data", &map[string]string{"reply": fmt.Sprintf("%v", reply)})
}
