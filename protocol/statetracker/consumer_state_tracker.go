package statetracker

import (
	"context"
	"fmt"
	"sync"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/protocol/chainlib"
	"github.com/lavanet/lava/protocol/chaintracker"
	"github.com/lavanet/lava/protocol/lavaprotocol"
	"github.com/lavanet/lava/relayer/lavasession"
	"github.com/lavanet/lava/utils"
	conflicttypes "github.com/lavanet/lava/x/conflict/types"
	spectypes "github.com/lavanet/lava/x/spec/types"
)

// ConsumerStateTracker CSTis a class for tracking consumer data from the lava blockchain, such as epoch changes.
// it allows also to query specific data form the blockchain and acts as a single place to send transactions
type ConsumerStateTracker struct {
	consumerAddress      sdk.AccAddress
	chainTracker         *chaintracker.ChainTracker
	stateQuery           *StateQuery
	txSender             *TxSender
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
	cst.stateQuery, err = stateQuery.New(ctx, clientCtx)
	if err != nil {
		return nil, err
	}

	txSender := TxSender{}
	cst.txSender, err = txSender.New(ctx, txFactory, clientCtx)
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
		pairingUpdater = NewPairingUpdater(cst.consumerAddress, cst.stateQuery)
		cst.newLavaBlockUpdaters[pairingUpdater.UpdaterKey()] = pairingUpdater
	}
	pairingUpdater, ok = pairingUpdater_raw.(*PairingUpdater)
	if !ok {
		utils.LavaFormatFatal("invalid_updater_key in RegisterConsumerSessionManagerForPairingUpdates", nil, &map[string]string{"updaters_map": fmt.Sprintf("%+v", cst.newLavaBlockUpdaters)})
	}
	pairingUpdater.RegisterPairing(consumerSessionManager)
}

func (cst *ConsumerStateTracker) RegisterFinalizationConsensusForUpdates(ctx context.Context, finalizationConsensus *lavaprotocol.FinalizationConsensus) {
	cst.registrationLock.Lock()
	defer cst.registrationLock.Unlock()

	var finalizationConsensusUpdater *FinalizationConsensusUpdater = nil // UpdaterKey is nil safe
	finalizationConsensusUpdater_raw, ok := cst.newLavaBlockUpdaters[finalizationConsensusUpdater.UpdaterKey()]
	if !ok {
		finalizationConsensusUpdater = NewFinalizationConsensusUpdater(cst.consumerAddress, cst.stateQuery)
		cst.newLavaBlockUpdaters[finalizationConsensusUpdater.UpdaterKey()] = finalizationConsensusUpdater
	}
	finalizationConsensusUpdater, ok = finalizationConsensusUpdater_raw.(*FinalizationConsensusUpdater)
	if !ok {
		utils.LavaFormatFatal("invalid_updater_key in RegisterFinalizationConsensusForUpdates", nil, &map[string]string{"updaters_map": fmt.Sprintf("%+v", cst.newLavaBlockUpdaters)})
	}
	finalizationConsensusUpdater.RegisterFinalizationConsensus(finalizationConsensus)
}

func (cst *ConsumerStateTracker) RegisterApiParserForSpecUpdates(ctx context.Context, chainParser chainlib.ChainParser) {
	// register this chainParser for spec updates
	// currently just set the first one, and have a TODO to handle spec changes
	// get the spec and set it into the chainParser
	spec := spectypes.Spec{}
	chainParser.SetSpec(spec)
}

func (cst *ConsumerStateTracker) TxConflictDetection(ctx context.Context, finalizationConflict *conflicttypes.FinalizationConflict, responseConflict *conflicttypes.ResponseConflict, sameProviderConflict *conflicttypes.FinalizationConflict) {
	cst.txSender.TxConflictDetection(ctx, finalizationConflict, responseConflict, sameProviderConflict)
}
