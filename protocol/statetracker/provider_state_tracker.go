package statetracker

import (
	"context"
	"time"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/lavanet/lava/v4/protocol/chaintracker"
	"github.com/lavanet/lava/v4/protocol/lavasession"
	"github.com/lavanet/lava/v4/protocol/metrics"
	"github.com/lavanet/lava/v4/protocol/rpcprovider/reliabilitymanager"
	updaters "github.com/lavanet/lava/v4/protocol/statetracker/updaters"
	"github.com/lavanet/lava/v4/utils"
	pairingtypes "github.com/lavanet/lava/v4/x/pairing/types"
	protocoltypes "github.com/lavanet/lava/v4/x/protocol/types"
)

// ProviderStateTracker PST is a class for tracking provider data from the lava blockchain, such as epoch changes.
// it allows also to query specific data form the blockchain and acts as a single place to send transactions
type ProviderStateTracker struct {
	StateQuery *updaters.ProviderStateQuery
	txSender   *ProviderTxSender
	*StateTracker
	*EmergencyTracker
}

func NewProviderStateTracker(ctx context.Context, txFactory tx.Factory, clientCtx client.Context, chainFetcher chaintracker.ChainFetcher, metrics *metrics.ProviderMetricsManager) (ret *ProviderStateTracker, err error) {
	emergencyTracker, blockNotFoundCallback := NewEmergencyTracker(metrics)
	stateQuery := updaters.NewProviderStateQuery(ctx, updaters.NewStateQueryAccessInst(clientCtx))
	stateTrackerBase, err := NewStateTracker(ctx, txFactory, stateQuery.StateQuery, chainFetcher, blockNotFoundCallback)
	if err != nil {
		return nil, err
	}
	txSender, err := NewProviderTxSender(ctx, clientCtx, txFactory)
	if err != nil {
		return nil, err
	}
	pst := &ProviderStateTracker{
		StateTracker:     stateTrackerBase,
		StateQuery:       stateQuery,
		txSender:         txSender,
		EmergencyTracker: emergencyTracker,
	}

	pst.RegisterForEpochUpdates(ctx, emergencyTracker)
	pst.RegisterForEpochUpdates(ctx, txSender)
	err = pst.RegisterForDowntimeParamsUpdates(ctx, emergencyTracker)
	return pst, err
}

func (pst *ProviderStateTracker) RegisterForEpochUpdates(ctx context.Context, epochUpdatable updaters.EpochUpdatable) {
	epochUpdater := updaters.NewEpochUpdater(&pst.StateQuery.EpochStateQuery)
	epochUpdaterRaw := pst.StateTracker.RegisterForUpdates(ctx, epochUpdater)
	epochUpdater, ok := epochUpdaterRaw.(*updaters.EpochUpdater)
	if !ok {
		utils.LavaFormatFatal("invalid updater type returned from RegisterForUpdates", nil, utils.Attribute{Key: "updater", Value: epochUpdaterRaw})
	}
	epochUpdater.RegisterEpochUpdatable(ctx, epochUpdatable, 0) // adding 0 delay for provider updater
}

func (pst *ProviderStateTracker) RegisterForSpecUpdates(ctx context.Context, specUpdatable updaters.SpecUpdatable, endpoint lavasession.RPCEndpoint) error {
	// register for spec updates sets spec and updates when a spec has been modified
	specUpdater := updaters.NewSpecUpdater(endpoint.ChainID, pst.StateQuery, pst.EventTracker)
	specUpdaterRaw := pst.StateTracker.RegisterForUpdates(ctx, specUpdater)
	specUpdater, ok := specUpdaterRaw.(*updaters.SpecUpdater)
	if !ok {
		utils.LavaFormatFatal("invalid updater type returned from RegisterForUpdates", nil, utils.Attribute{Key: "updater", Value: specUpdaterRaw})
	}
	return specUpdater.RegisterSpecUpdatable(ctx, &specUpdatable, endpoint)
}

func (pst *ProviderStateTracker) RegisterForSpecVerifications(ctx context.Context, specVerifier updaters.SpecVerifier, chainId string) error {
	// register for spec verifications sets spec and verifies when a spec has been modified
	specUpdater := updaters.NewSpecUpdater(chainId, pst.StateQuery, pst.EventTracker)
	specUpdaterRaw := pst.StateTracker.RegisterForUpdates(ctx, specUpdater)
	specUpdater, ok := specUpdaterRaw.(*updaters.SpecUpdater)
	if !ok {
		utils.LavaFormatFatal("invalid updater type returned from RegisterForSpecVerifications", nil, utils.Attribute{Key: "updater", Value: specUpdaterRaw})
	}
	return specUpdater.RegisterSpecVerifier(ctx, &specVerifier, chainId)
}

func (pst *ProviderStateTracker) RegisterForVersionUpdates(ctx context.Context, version *protocoltypes.Version, versionValidator updaters.VersionValidationInf) {
	versionUpdater := updaters.NewVersionUpdater(pst.StateQuery, pst.EventTracker, version, versionValidator)
	versionUpdaterRaw := pst.StateTracker.RegisterForUpdates(ctx, versionUpdater)
	versionUpdater, ok := versionUpdaterRaw.(*updaters.VersionUpdater)
	if !ok {
		utils.LavaFormatFatal("invalid updater type returned from RegisterForUpdates", nil, utils.Attribute{Key: "updater", Value: versionUpdaterRaw})
	}
	versionUpdater.RegisterVersionUpdatable()
}

func (pst *ProviderStateTracker) RegisterReliabilityManagerForVoteUpdates(ctx context.Context, voteUpdatable updaters.VoteUpdatable, endpointP *lavasession.RPCProviderEndpoint) {
	voteUpdater := updaters.NewVoteUpdater(pst.EventTracker)
	voteUpdaterRaw := pst.StateTracker.RegisterForUpdates(ctx, voteUpdater)
	voteUpdater, ok := voteUpdaterRaw.(*updaters.VoteUpdater)
	if !ok {
		utils.LavaFormatFatal("invalid updater type returned from RegisterForUpdates", nil, utils.Attribute{Key: "updater", Value: voteUpdaterRaw})
	}
	endpoint := lavasession.RPCEndpoint{ChainID: endpointP.ChainID, ApiInterface: endpointP.ApiInterface}
	voteUpdater.RegisterVoteUpdatable(ctx, &voteUpdatable, endpoint)
}

func (pst *ProviderStateTracker) RegisterPaymentUpdatableForPayments(ctx context.Context, paymentUpdatable updaters.PaymentUpdatable) {
	paymentUpdater := updaters.NewPaymentUpdater(pst.EventTracker)
	paymentUpdaterRaw := pst.StateTracker.RegisterForUpdates(ctx, paymentUpdater)
	paymentUpdater, ok := paymentUpdaterRaw.(*updaters.PaymentUpdater)
	if !ok {
		utils.LavaFormatFatal("invalid updater type returned from RegisterForUpdates", nil, utils.Attribute{Key: "updater", Value: paymentUpdaterRaw})
	}

	paymentUpdater.RegisterPaymentUpdatable(ctx, &paymentUpdatable)
}

func (pst *ProviderStateTracker) RegisterForDowntimeParamsUpdates(ctx context.Context, downtimeParamsUpdatable updaters.DowntimeParamsUpdatable) error {
	// register for downtimeParams updates sets downtimeParams and updates when downtimeParams has been changed
	downtimeParamsUpdater := updaters.NewDowntimeParamsUpdater(pst.StateQuery, pst.EventTracker)
	downtimeParamsUpdaterRaw := pst.StateTracker.RegisterForUpdates(ctx, downtimeParamsUpdater)
	downtimeParamsUpdater, ok := downtimeParamsUpdaterRaw.(*updaters.DowntimeParamsUpdater)
	if !ok {
		utils.LavaFormatFatal("invalid updater type returned from RegisterForUpdates", nil, utils.Attribute{Key: "updater", Value: downtimeParamsUpdaterRaw})
	}

	return downtimeParamsUpdater.RegisterDowntimeParamsUpdatable(ctx, &downtimeParamsUpdatable)
}

func (pst *ProviderStateTracker) TxRelayPayment(ctx context.Context, relayRequests []*pairingtypes.RelaySession, description string, latestBlocks []*pairingtypes.LatestBlockReport) error {
	return pst.txSender.TxRelayPayment(ctx, relayRequests, description, latestBlocks)
}

func (pst *ProviderStateTracker) SendVoteReveal(voteID string, vote *reliabilitymanager.VoteData, specID string) error {
	return pst.txSender.SendVoteReveal(context.Background(), voteID, vote, specID)
}

func (pst *ProviderStateTracker) SendVoteCommitment(voteID string, vote *reliabilitymanager.VoteData, specID string) error {
	return pst.txSender.SendVoteCommitment(context.Background(), voteID, vote, specID)
}

func (pst *ProviderStateTracker) LatestBlock() int64 {
	return pst.StateTracker.chainTracker.GetAtomicLatestBlockNum()
}

func (pst *ProviderStateTracker) GetMaxCuForUser(ctx context.Context, consumerAddress, chainID string, epoch uint64) (maxCu uint64, err error) {
	return pst.StateQuery.GetMaxCuForUser(ctx, consumerAddress, chainID, epoch)
}

func (pst *ProviderStateTracker) VerifyPairing(ctx context.Context, consumerAddress, providerAddress string, epoch uint64, chainID string) (valid bool, total int64, projectId string, err error) {
	return pst.StateQuery.VerifyPairing(ctx, consumerAddress, providerAddress, epoch, chainID)
}

func (pst *ProviderStateTracker) GetEpochSize(ctx context.Context) (uint64, error) {
	return pst.StateQuery.GetEpochSize(ctx)
}

func (pst *ProviderStateTracker) EarliestBlockInMemory(ctx context.Context) (uint64, error) {
	return pst.StateQuery.EarliestBlockInMemory(ctx)
}

func (pst *ProviderStateTracker) GetRecommendedEpochNumToCollectPayment(ctx context.Context) (uint64, error) {
	return pst.StateQuery.GetRecommendedEpochNumToCollectPayment(ctx)
}

func (pst *ProviderStateTracker) GetEpochSizeMultipliedByRecommendedEpochNumToCollectPayment(ctx context.Context) (uint64, error) {
	return pst.StateQuery.GetEpochSizeMultipliedByRecommendedEpochNumToCollectPayment(ctx)
}

func (pst *ProviderStateTracker) GetProtocolVersion(ctx context.Context) (*updaters.ProtocolVersionResponse, error) {
	return pst.StateQuery.GetProtocolVersion(ctx)
}

func (pst *ProviderStateTracker) GetAverageBlockTime() time.Duration {
	return pst.StateTracker.GetAverageBlockTime()
}
