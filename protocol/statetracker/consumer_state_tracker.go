package statetracker

import (
	"context"
	"time"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/lavanet/lava/v5/protocol/chaintracker"
	"github.com/lavanet/lava/v5/protocol/lavasession"
	"github.com/lavanet/lava/v5/protocol/metrics"
	updaters "github.com/lavanet/lava/v5/protocol/statetracker/updaters"
	"github.com/lavanet/lava/v5/utils"
	plantypes "github.com/lavanet/lava/v5/x/plans/types"
	protocoltypes "github.com/lavanet/lava/v5/x/protocol/types"
)

// Data Reliability disabled - Phase 2 removal
// DELETED: var DisableDR = false
// DELETED: ConsumerTxSenderInf interface (TxSenderConflictDetection)

// ConsumerStateTracker CSTis a class for tracking consumer data from the lava blockchain, such as epoch changes.
// it allows also to query specific data form the blockchain and acts as a single place to send transactions
type ConsumerStateTracker struct {
	StateQuery *updaters.ConsumerStateQuery
	// Data Reliability disabled - Phase 2: removed ConsumerTxSenderInf (conflict detection)
	*StateTracker
	ConsumerEmergencyTrackerInf
}

func NewConsumerStateTracker(ctx context.Context, txFactory tx.Factory, clientCtx client.Context, chainFetcher chaintracker.ChainFetcher, metrics *metrics.ConsumerMetricsManager) (ret *ConsumerStateTracker, err error) {
	emergencyTracker, blockNotFoundCallback := NewEmergencyTracker(metrics)
	stateQuery := updaters.NewConsumerStateQuery(ctx, clientCtx)
	stateTrackerBase, err := NewStateTracker(ctx, txFactory, stateQuery.StateQuery, chainFetcher, blockNotFoundCallback)
	if err != nil {
		return nil, err
	}
	// Data Reliability disabled - Phase 2: removed txSender creation (conflict detection)
	cst := &ConsumerStateTracker{
		StateTracker:                stateTrackerBase,
		StateQuery:                  stateQuery,
		ConsumerEmergencyTrackerInf: emergencyTracker,
	}

	err = cst.RegisterForDowntimeParamsUpdates(ctx, emergencyTracker)
	return cst, err
}

func (cst *ConsumerStateTracker) RegisterConsumerSessionManagerForPairingUpdates(ctx context.Context, consumerSessionManager *lavasession.ConsumerSessionManager, staticProvidersList []*lavasession.RPCStaticProviderEndpoint, backupProvidersList []*lavasession.RPCStaticProviderEndpoint) {
	// register this CSM to get the updated pairing list when a new epoch starts
	pairingUpdater := updaters.NewPairingUpdater(cst.StateQuery, consumerSessionManager.RPCEndpoint().ChainID)
	pairingUpdaterRaw := cst.StateTracker.RegisterForUpdates(ctx, pairingUpdater)
	pairingUpdater, ok := pairingUpdaterRaw.(*updaters.PairingUpdater)
	if !ok {
		utils.LavaFormatFatal("invalid updater type returned from RegisterForUpdates", nil, utils.Attribute{Key: "updater", Value: pairingUpdaterRaw})
	}

	err := pairingUpdater.RegisterPairing(ctx, consumerSessionManager, staticProvidersList, backupProvidersList)
	if err != nil {
		// if failed registering pairing, continue trying asynchronously
		go func() {
			numberOfAttempts := 0
			for {
				utils.LavaFormatError("Failed retry RegisterPairing", err, utils.LogAttr("attempt", numberOfAttempts), utils.Attribute{Key: "data", Value: consumerSessionManager.RPCEndpoint()})
				time.Sleep(5 * time.Second) // sleep so we don't spam get pairing for no reason
				err := pairingUpdater.RegisterPairing(ctx, consumerSessionManager, staticProvidersList, backupProvidersList)
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

// Data Reliability disabled - Phase 2 removal
// DELETED: RegisterFinalizationConsensusForUpdates() function
// DELETED: TxConflictDetection() function

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
