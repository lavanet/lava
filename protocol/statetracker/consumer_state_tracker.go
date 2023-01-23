package statetracker

import (
	"context"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/protocol/chainlib"
	"github.com/lavanet/lava/protocol/lavaprotocol"
	"github.com/lavanet/lava/relayer/lavasession"
	conflicttypes "github.com/lavanet/lava/x/conflict/types"
	spectypes "github.com/lavanet/lava/x/spec/types"
)

// ConsumerStateTracker CST is a class for tracking consumer data from the lava blockchain, such as epoch changes.
// it allows also to query specific data form the blockchain and acts as a single place to send transactions
type ConsumerStateTracker struct {
	consumerAddress              sdk.AccAddress
	stateQuery                   StateQuery
	txSender                     *TxSender
	pairingUpdater               PairingUpdater
	finalizationConsensusUpdater FinalizationConsensusUpdater
}

type Updater interface {
	Update(latestBlock int64) error
}

func (cst *ConsumerStateTracker) New(ctx context.Context, txFactory tx.Factory, clientCtx client.Context) (ret *ConsumerStateTracker, err error) {
	// set up StateQuery
	// Spin up chain tracker on the lava node, its address is in the --node flag (or its default), on new block call to newLavaBlock
	// use StateQuery to get the lava spec and spin up the chain tracker with the right params
	// set up txSender the same way

	stateQuery := NewStateQuery(ctx, clientCtx)
	cst.stateQuery = stateQuery

	txSender := TxSender{}
	cst.txSender, err = txSender.New(ctx, txFactory, clientCtx)
	if err != nil {
		return nil, err
	}
	cst.consumerAddress = clientCtx.FromAddress
	return cst, nil
}

func (cst *ConsumerStateTracker) RegisterConsumerSessionManagerForPairingUpdates(ctx context.Context, consumerSessionManager *lavasession.ConsumerSessionManager) {
	// register this CSM to get the updated pairing list when a new epoch starts
	// make sure new lava block exists as a callback in stateTracker
	// add updatePairingForRegistered as a callback on a new block

	cst.pairingUpdater.RegisterPairing(consumerSessionManager)
}

func (cst *ConsumerStateTracker) RegisterFinalizationConsensusForUpdates(ctx context.Context, finalizationConsensus *lavaprotocol.FinalizationConsensus) {
	cst.finalizationConsensusUpdater.RegisterFinalizationConsensus(finalizationConsensus)
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
