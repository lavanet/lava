package statetracker

import (
	"context"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/lavanet/lava/protocol/chaintracker"
	"github.com/lavanet/lava/protocol/lavaprotocol"
	"github.com/lavanet/lava/protocol/lavasession"
	"github.com/lavanet/lava/utils"
	conflicttypes "github.com/lavanet/lava/x/conflict/types"
)

// ConsumerStateTracker CSTis a class for tracking consumer data from the lava blockchain, such as epoch changes.
// it allows also to query specific data form the blockchain and acts as a single place to send transactions
type ConsumerStateTracker struct {
	stateQuery *ConsumerStateQuery
	txSender   *ConsumerTxSender
	*StateTracker
}

func NewConsumerStateTracker(ctx context.Context, txFactory tx.Factory, clientCtx client.Context, chainFetcher chaintracker.ChainFetcher) (ret *ConsumerStateTracker, err error) {
	stateTrackerBase, err := NewStateTracker(ctx, txFactory, clientCtx, chainFetcher)
	if err != nil {
		return nil, err
	}
	txSender, err := NewConsumerTxSender(ctx, clientCtx, txFactory)
	if err != nil {
		return nil, err
	}
	cst := &ConsumerStateTracker{StateTracker: stateTrackerBase, stateQuery: NewConsumerStateQuery(ctx, clientCtx), txSender: txSender}
	return cst, nil
}

func (cst *ConsumerStateTracker) RegisterConsumerSessionManagerForPairingUpdates(ctx context.Context, consumerSessionManager *lavasession.ConsumerSessionManager) {
	// register this CSM to get the updated pairing list when a new epoch starts
	pairingUpdater := NewPairingUpdater(cst.stateQuery)
	pairingUpdaterRaw := cst.StateTracker.RegisterForUpdates(ctx, pairingUpdater)
	pairingUpdater, ok := pairingUpdaterRaw.(*PairingUpdater)
	if !ok {
		utils.LavaFormatFatal("invalid updater type returned from RegisterForUpdates", nil, utils.Attribute{Key: "updater", Value: pairingUpdaterRaw})
	}
	err := pairingUpdater.RegisterPairing(ctx, consumerSessionManager)
	if err != nil {
		utils.LavaFormatError("failed registering for pairing updates", err, utils.Attribute{Key: "data", Value: consumerSessionManager.RPCEndpoint()})
	}
}

func (cst *ConsumerStateTracker) RegisterFinalizationConsensusForUpdates(ctx context.Context, finalizationConsensus *lavaprotocol.FinalizationConsensus) {
	finalizationConsensusUpdater := NewFinalizationConsensusUpdater(cst.stateQuery)
	finalizationConsensusUpdaterRaw := cst.StateTracker.RegisterForUpdates(ctx, finalizationConsensusUpdater)
	finalizationConsensusUpdater, ok := finalizationConsensusUpdaterRaw.(*FinalizationConsensusUpdater)
	if !ok {
		utils.LavaFormatFatal("invalid updater type returned from RegisterForUpdates", nil, utils.Attribute{Key: "updater", Value: finalizationConsensusUpdaterRaw})
	}
	finalizationConsensusUpdater.RegisterFinalizationConsensus(finalizationConsensus)
}

func (cst *ConsumerStateTracker) TxConflictDetection(ctx context.Context, finalizationConflict *conflicttypes.FinalizationConflict, responseConflict *conflicttypes.ResponseConflict, sameProviderConflict *conflicttypes.FinalizationConflict) error {
	err := cst.txSender.TxConflictDetection(ctx, finalizationConflict, responseConflict, sameProviderConflict)
	return err
}

func (cst *ConsumerStateTracker) RegisterForSpecUpdates(ctx context.Context, specUpdatable SpecUpdatable, endpoint lavasession.RPCEndpoint) error {
	// register for spec updates sets spec and updates when a spec has been modified
	specUpdater := NewSpecUpdater(endpoint.ChainID, cst.stateQuery, cst.eventTracker)
	specUpdaterRaw := cst.StateTracker.RegisterForUpdates(ctx, specUpdater)
	specUpdater, ok := specUpdaterRaw.(*SpecUpdater)
	if !ok {
		utils.LavaFormatFatal("invalid updater type returned from RegisterForUpdates", nil, utils.Attribute{Key: "updater", Value: specUpdaterRaw})
	}
	return specUpdater.RegisterSpecUpdatable(ctx, &specUpdatable, endpoint)
}

func (cst *ConsumerStateTracker) RegisterForVersionUpdates(ctx context.Context, versionUpdatable VersionUpdatable) {
	versionUpdater := NewVersionUpdater(cst.stateQuery, cst.eventTracker)
	versionUpdaterRaw := cst.StateTracker.RegisterForUpdates(ctx, versionUpdater)
	versionUpdater, ok := versionUpdaterRaw.(*VersionUpdater)
	if !ok {
		utils.LavaFormatFatal("invalid updater type returned from RegisterForUpdates", nil, utils.Attribute{Key: "updater", Value: versionUpdaterRaw})
	}
	versionUpdater.RegisterVersionUpdatable(ctx, versionUpdatable)
}

func (cst *ConsumerStateTracker) CheckProtocolVersion(ctx context.Context) error {
	return cst.stateQuery.CheckProtocolVersion(ctx)
}
