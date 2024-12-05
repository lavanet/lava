package statetracker

import (
	"context"
	"time"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/lavanet/lava/v4/protocol/chaintracker"
	"github.com/lavanet/lava/v4/protocol/common"
	"github.com/lavanet/lava/v4/protocol/lavaprotocol/finalizationconsensus"
	"github.com/lavanet/lava/v4/protocol/lavasession"
	"github.com/lavanet/lava/v4/protocol/metrics"
	updaters "github.com/lavanet/lava/v4/protocol/statetracker/updaters"
	"github.com/lavanet/lava/v4/utils"
	conflicttypes "github.com/lavanet/lava/v4/x/conflict/types"
	plantypes "github.com/lavanet/lava/v4/x/plans/types"
	protocoltypes "github.com/lavanet/lava/v4/x/protocol/types"
)

type ConsumerTxSenderInf interface {
	TxSenderConflictDetection(ctx context.Context, finalizationConflict *conflicttypes.FinalizationConflict, responseConflict *conflicttypes.ResponseConflict) error
}

// ConsumerStateTracker CSTis a class for tracking consumer data from the lava blockchain, such as epoch changes.
// it allows also to query specific data form the blockchain and acts as a single place to send transactions
type ConsumerStateTracker struct {
	StateQuery *updaters.ConsumerStateQuery
	ConsumerTxSenderInf
	*StateTracker
	ConsumerEmergencyTrackerInf
	disableConflictTransactions bool
}

func NewConsumerStateTracker(ctx context.Context, txFactory tx.Factory, clientCtx client.Context, chainFetcher chaintracker.ChainFetcher, metrics *metrics.ConsumerMetricsManager, disableConflictTransactions bool) (ret *ConsumerStateTracker, err error) {
	emergencyTracker, blockNotFoundCallback := NewEmergencyTracker(metrics)
	stateQuery := updaters.NewConsumerStateQuery(ctx, clientCtx)
	stateTrackerBase, err := NewStateTracker(ctx, txFactory, stateQuery.StateQuery, chainFetcher, blockNotFoundCallback)
	if err != nil {
		return nil, err
	}
	txSender, err := NewConsumerTxSender(ctx, clientCtx, txFactory)
	if err != nil {
		return nil, err
	}
	cst := &ConsumerStateTracker{
		StateTracker:                stateTrackerBase,
		StateQuery:                  stateQuery,
		ConsumerTxSenderInf:         txSender,
		ConsumerEmergencyTrackerInf: emergencyTracker,
		disableConflictTransactions: disableConflictTransactions,
	}

	err = cst.RegisterForDowntimeParamsUpdates(ctx, emergencyTracker)
	return cst, err
}

func (cst *ConsumerStateTracker) RegisterConsumerSessionManagerForPairingUpdates(ctx context.Context, consumerSessionManager *lavasession.ConsumerSessionManager, staticProvidersList []*lavasession.RPCProviderEndpoint) {
	// register this CSM to get the updated pairing list when a new epoch starts
	pairingUpdater := updaters.NewPairingUpdater(cst.StateQuery, consumerSessionManager.RPCEndpoint().ChainID)
	pairingUpdaterRaw := cst.StateTracker.RegisterForUpdates(ctx, pairingUpdater)
	pairingUpdater, ok := pairingUpdaterRaw.(*updaters.PairingUpdater)
	if !ok {
		utils.LavaFormatFatal("invalid updater type returned from RegisterForUpdates", nil, utils.Attribute{Key: "updater", Value: pairingUpdaterRaw})
	}

	err := pairingUpdater.RegisterPairing(ctx, consumerSessionManager, staticProvidersList)
	if err != nil {
		// if failed registering pairing, continue trying asynchronously
		go func() {
			numberOfAttempts := 0
			for {
				utils.LavaFormatError("Failed retry RegisterPairing", err, utils.LogAttr("attempt", numberOfAttempts), utils.Attribute{Key: "data", Value: consumerSessionManager.RPCEndpoint()})
				time.Sleep(5 * time.Second) // sleep so we don't spam get pairing for no reason
				err := pairingUpdater.RegisterPairing(ctx, consumerSessionManager, staticProvidersList)
				if err == nil {
					break
				}
			}
		}()
	}
}

func (cst *ConsumerStateTracker) RegisterForPairingUpdates(ctx context.Context, pairingUpdatable updaters.PairingUpdatable, specId string) {
	pairingUpdater := updaters.NewPairingUpdater(cst.StateQuery, specId)
	pairingUpdaterRaw := cst.StateTracker.RegisterForUpdates(ctx, pairingUpdater)
	pairingUpdater, ok := pairingUpdaterRaw.(*updaters.PairingUpdater)
	if !ok {
		utils.LavaFormatFatal("invalid updater type returned from RegisterForUpdates", nil, utils.Attribute{Key: "updater", Value: pairingUpdaterRaw})
	}
	err := pairingUpdater.RegisterPairingUpdatable(ctx, &pairingUpdatable)
	if err != nil {
		utils.LavaFormatError("failed registering updatable for pairing updates", err)
	}
}

func (cst *ConsumerStateTracker) RegisterFinalizationConsensusForUpdates(ctx context.Context, finalizationConsensus *finalizationconsensus.FinalizationConsensus) {
	finalizationConsensusUpdater := updaters.NewFinalizationConsensusUpdater(cst.StateQuery, finalizationConsensus.SpecId)
	finalizationConsensusUpdaterRaw := cst.StateTracker.RegisterForUpdates(ctx, finalizationConsensusUpdater)
	finalizationConsensusUpdater, ok := finalizationConsensusUpdaterRaw.(*updaters.FinalizationConsensusUpdater)
	if !ok {
		utils.LavaFormatFatal("invalid updater type returned from RegisterForUpdates", nil, utils.Attribute{Key: "updater", Value: finalizationConsensusUpdaterRaw})
	}
	finalizationConsensusUpdater.RegisterFinalizationConsensus(finalizationConsensus)
}

func (cst *ConsumerStateTracker) TxConflictDetection(ctx context.Context, finalizationConflict *conflicttypes.FinalizationConflict, responseConflict *conflicttypes.ResponseConflict, conflictHandler common.ConflictHandlerInterface) error {
	if cst.disableConflictTransactions {
		utils.LavaFormatInfo("found Conflict, but transactions are disabled, returning")
		return nil
	}
	if conflictHandler.ConflictAlreadyReported() {
		return nil // already reported
	}
	err := cst.TxSenderConflictDetection(ctx, finalizationConflict, responseConflict)
	if err == nil { // if conflict report succeeded, we can set this provider as reported, so we wont need to report again.
		conflictHandler.StoreConflictReported()
	}
	return err
}

func (cst *ConsumerStateTracker) RegisterForSpecUpdates(ctx context.Context, specUpdatable updaters.SpecUpdatable, endpoint lavasession.RPCEndpoint) error {
	// register for spec updates sets spec and updates when a spec has been modified
	specUpdater := updaters.NewSpecUpdater(endpoint.ChainID, cst.StateQuery, cst.EventTracker)
	specUpdaterRaw := cst.StateTracker.RegisterForUpdates(ctx, specUpdater)
	specUpdater, ok := specUpdaterRaw.(*updaters.SpecUpdater)
	if !ok {
		utils.LavaFormatFatal("invalid updater type returned from RegisterForUpdates", nil, utils.Attribute{Key: "updater", Value: specUpdaterRaw})
	}
	return specUpdater.RegisterSpecUpdatable(ctx, &specUpdatable, endpoint)
}

func (cst *ConsumerStateTracker) GetConsumerPolicy(ctx context.Context, consumerAddress, chainID string) (*plantypes.Policy, error) {
	return cst.StateQuery.GetEffectivePolicy(ctx, consumerAddress, chainID)
}

func (cst *ConsumerStateTracker) RegisterForVersionUpdates(ctx context.Context, version *protocoltypes.Version, versionValidator updaters.VersionValidationInf) {
	versionUpdater := updaters.NewVersionUpdater(cst.StateQuery, cst.EventTracker, version, versionValidator)
	versionUpdaterRaw := cst.StateTracker.RegisterForUpdates(ctx, versionUpdater)
	versionUpdater, ok := versionUpdaterRaw.(*updaters.VersionUpdater)
	if !ok {
		utils.LavaFormatFatal("invalid updater type returned from RegisterForUpdates", nil, utils.Attribute{Key: "updater", Value: versionUpdaterRaw})
	}
	versionUpdater.RegisterVersionUpdatable()
}

func (cst *ConsumerStateTracker) RegisterForDowntimeParamsUpdates(ctx context.Context, downtimeParamsUpdatable updaters.DowntimeParamsUpdatable) error {
	// register for downtimeParams updates sets downtimeParams and updates when downtimeParams has been changed
	downtimeParamsUpdater := updaters.NewDowntimeParamsUpdater(cst.StateQuery, cst.EventTracker)
	downtimeParamsUpdaterRaw := cst.StateTracker.RegisterForUpdates(ctx, downtimeParamsUpdater)
	downtimeParamsUpdater, ok := downtimeParamsUpdaterRaw.(*updaters.DowntimeParamsUpdater)
	if !ok {
		utils.LavaFormatFatal("invalid updater type returned from RegisterForUpdates", nil, utils.Attribute{Key: "updater", Value: downtimeParamsUpdaterRaw})
	}

	return downtimeParamsUpdater.RegisterDowntimeParamsUpdatable(ctx, &downtimeParamsUpdatable)
}

func (cst *ConsumerStateTracker) GetProtocolVersion(ctx context.Context) (*updaters.ProtocolVersionResponse, error) {
	return cst.StateQuery.GetProtocolVersion(ctx)
}
