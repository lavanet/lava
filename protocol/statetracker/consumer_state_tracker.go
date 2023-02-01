package statetracker

import (
	"context"
	"fmt"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/protocol/chainlib"
	"github.com/lavanet/lava/protocol/chaintracker"
	"github.com/lavanet/lava/protocol/lavaprotocol"
	"github.com/lavanet/lava/relayer/lavasession"
	"github.com/lavanet/lava/utils"
	conflicttypes "github.com/lavanet/lava/x/conflict/types"
)

// ConsumerStateTracker CSTis a class for tracking consumer data from the lava blockchain, such as epoch changes.
// it allows also to query specific data form the blockchain and acts as a single place to send transactions
type ConsumerStateTracker struct {
	consumerAddress sdk.AccAddress
	stateQuery      *ConsumerStateQuery
	txSender        *ConsumerTxSender
	*StateTracker
}

func NewConsumerStateTracker(ctx context.Context, txFactory tx.Factory, clientCtx client.Context, chainFetcher chaintracker.ChainFetcher) (ret *ConsumerStateTracker, err error) {
	stateTrackerBase, err := NewStateTracker(ctx, txFactory, clientCtx, chainFetcher)
	if err != nil {
		return nil, err
	}
	txSender, err := NewConsumerTxSender(ctx, clientCtx, txFactory)
	cst := &ConsumerStateTracker{StateTracker: stateTrackerBase, stateQuery: NewConsumerStateQuery(ctx, clientCtx), txSender: txSender}
	return cst, nil

}

func (cst *ConsumerStateTracker) RegisterConsumerSessionManagerForPairingUpdates(ctx context.Context, consumerSessionManager *lavasession.ConsumerSessionManager) {
	// register this CSM to get the updated pairing list when a new epoch starts
	pairingUpdater := NewPairingUpdater(cst.consumerAddress, cst.stateQuery)
	pairingUpdaterRaw := cst.StateTracker.RegisterForUpdates(ctx, pairingUpdater)
	pairingUpdater, ok := pairingUpdaterRaw.(*PairingUpdater)
	if !ok {
		utils.LavaFormatFatal("invalid updater type returned from RegisterForUpdates", nil, &map[string]string{"updater": fmt.Sprintf("%+v", pairingUpdaterRaw)})
	}
	pairingUpdater.RegisterPairing(ctx, consumerSessionManager)
}

func (cst *ConsumerStateTracker) RegisterFinalizationConsensusForUpdates(ctx context.Context, finalizationConsensus *lavaprotocol.FinalizationConsensus) {
	finalizationConsensusUpdater := NewFinalizationConsensusUpdater(cst.consumerAddress, cst.stateQuery)
	finalizationConsensusUpdaterRaw := cst.StateTracker.RegisterForUpdates(ctx, finalizationConsensusUpdater)
	finalizationConsensusUpdater, ok := finalizationConsensusUpdaterRaw.(*FinalizationConsensusUpdater)
	if !ok {
		utils.LavaFormatFatal("invalid updater type returned from RegisterForUpdates", nil, &map[string]string{"updater": fmt.Sprintf("%+v", finalizationConsensusUpdaterRaw)})
	}
	finalizationConsensusUpdater.RegisterFinalizationConsensus(finalizationConsensus)
}

func (cst *ConsumerStateTracker) RegisterChainParserForSpecUpdates(ctx context.Context, chainParser chainlib.ChainParser, chainID string) error {
	// TODO: handle spec changes
	spec, err := cst.stateQuery.GetSpec(ctx, chainID)
	if err != nil {
		return err
	}
	chainParser.SetSpec(*spec)
	return nil
}

func (cst *ConsumerStateTracker) TxConflictDetection(ctx context.Context, finalizationConflict *conflicttypes.FinalizationConflict, responseConflict *conflicttypes.ResponseConflict, sameProviderConflict *conflicttypes.FinalizationConflict) error {
	err := cst.txSender.TxConflictDetection(ctx, finalizationConflict, responseConflict, sameProviderConflict)
	return err
}
