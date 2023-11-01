package statetracker

import (
	"context"
	"time"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/lavanet/lava/protocol/chaintracker"
	"github.com/lavanet/lava/protocol/lavasession"
	"github.com/lavanet/lava/protocol/rpcprovider/reliabilitymanager"
	"github.com/lavanet/lava/utils"
	pairingtypes "github.com/lavanet/lava/x/pairing/types"
	protocoltypes "github.com/lavanet/lava/x/protocol/types"
)

// ProviderStateTracker PST is a class for tracking provider data from the lava blockchain, such as epoch changes.
// it allows also to query specific data form the blockchain and acts as a single place to send transactions
type ProviderStateTracker struct {
	stateQuery *ProviderStateQuery
	txSender   *ProviderTxSender
	*StateTracker
	*EmergencyTracker
}

func NewProviderStateTracker(ctx context.Context, txFactory tx.Factory, clientCtx client.Context, chainFetcher chaintracker.ChainFetcher) (ret *ProviderStateTracker, err error) {
	emergencyTracker, blockNotFoundCallback := NewEmergencyTracker()
	stateTrackerBase, err := NewStateTracker(ctx, txFactory, clientCtx, chainFetcher, blockNotFoundCallback)
	if err != nil {
		return nil, err
	}
	txSender, err := NewProviderTxSender(ctx, clientCtx, txFactory)
	if err != nil {
		return nil, err
	}
	pst := &ProviderStateTracker{
		StateTracker:     stateTrackerBase,
		stateQuery:       NewProviderStateQuery(ctx, clientCtx),
		txSender:         txSender,
		EmergencyTracker: emergencyTracker,
	}
	return pst, nil
}

func (pst *ProviderStateTracker) RegisterForEpochUpdates(ctx context.Context, epochUpdatable EpochUpdatable) {
	epochUpdater := NewEpochUpdater(&pst.stateQuery.EpochStateQuery)
	epochUpdaterRaw := pst.StateTracker.RegisterForUpdates(ctx, epochUpdater)
	epochUpdater, ok := epochUpdaterRaw.(*EpochUpdater)
	if !ok {
		utils.LavaFormatFatal("invalid updater type returned from RegisterForUpdates", nil, utils.Attribute{Key: "updater", Value: epochUpdaterRaw})
	}

	// register for updates in case of emergency mode is enabled
	epochUpdaterWithEmergencyRaw := pst.StateTracker.RegisterForEmergencyModeUpdates(ctx, epochUpdater)
	epochUpdater, ok = epochUpdaterWithEmergencyRaw.(*EpochUpdater)
	if !ok {
		utils.LavaFormatFatal("invalid updater type returned from RegisterForUpdates", nil, utils.Attribute{Key: "updater", Value: epochUpdaterWithEmergencyRaw})
	}
	epochUpdater.RegisterEpochUpdatable(ctx, epochUpdatable, 0) // adding 0 delay for provider updater
}

func (pst *ProviderStateTracker) RegisterForSpecUpdates(ctx context.Context, specUpdatable SpecUpdatable, endpoint lavasession.RPCEndpoint) error {
	// register for spec updates sets spec and updates when a spec has been modified
	specUpdater := NewSpecUpdater(endpoint.ChainID, pst.stateQuery, pst.EventTracker)
	specUpdaterRaw := pst.StateTracker.RegisterForUpdates(ctx, specUpdater)
	specUpdater, ok := specUpdaterRaw.(*SpecUpdater)
	if !ok {
		utils.LavaFormatFatal("invalid updater type returned from RegisterForUpdates", nil, utils.Attribute{Key: "updater", Value: specUpdaterRaw})
	}
	return specUpdater.RegisterSpecUpdatable(ctx, &specUpdatable, endpoint)
}

func (pst *ProviderStateTracker) RegisterForVersionUpdates(ctx context.Context, version *protocoltypes.Version, versionValidator VersionValidationInf) {
	versionUpdater := NewVersionUpdater(pst.stateQuery, pst.EventTracker, version, versionValidator)
	versionUpdaterRaw := pst.StateTracker.RegisterForUpdates(ctx, versionUpdater)
	versionUpdater, ok := versionUpdaterRaw.(*VersionUpdater)
	if !ok {
		utils.LavaFormatFatal("invalid updater type returned from RegisterForUpdates", nil, utils.Attribute{Key: "updater", Value: versionUpdaterRaw})
	}
	versionUpdater.RegisterVersionUpdatable()
}

func (pst *ProviderStateTracker) RegisterReliabilityManagerForVoteUpdates(ctx context.Context, voteUpdatable VoteUpdatable, endpointP *lavasession.RPCProviderEndpoint) {
	voteUpdater := NewVoteUpdater(pst.EventTracker)
	voteUpdaterRaw := pst.StateTracker.RegisterForUpdates(ctx, voteUpdater)
	voteUpdater, ok := voteUpdaterRaw.(*VoteUpdater)
	if !ok {
		utils.LavaFormatFatal("invalid updater type returned from RegisterForUpdates", nil, utils.Attribute{Key: "updater", Value: voteUpdaterRaw})
	}
	endpoint := lavasession.RPCEndpoint{ChainID: endpointP.ChainID, ApiInterface: endpointP.ApiInterface}
	voteUpdater.RegisterVoteUpdatable(ctx, &voteUpdatable, endpoint)
}

func (pst *ProviderStateTracker) RegisterPaymentUpdatableForPayments(ctx context.Context, paymentUpdatable PaymentUpdatable) {
	paymentUpdater := NewPaymentUpdater(pst.EventTracker)
	paymentUpdaterRaw := pst.StateTracker.RegisterForUpdates(ctx, paymentUpdater)
	paymentUpdater, ok := paymentUpdaterRaw.(*PaymentUpdater)
	if !ok {
		utils.LavaFormatFatal("invalid updater type returned from RegisterForUpdates", nil, utils.Attribute{Key: "updater", Value: paymentUpdaterRaw})
	}

	paymentUpdater.RegisterPaymentUpdatable(ctx, &paymentUpdatable)
}

func (pst *ProviderStateTracker) RegisterForDowntimeParamsUpdates(ctx context.Context, downtimeParamsUpdatable DowntimeParamsUpdatable) error {
	// register for downtimeParams updates sets downtimeParams and updates when downtimeParams has been changed
	downtimeParamsUpdater := NewDowntimeParamsUpdater(pst.stateQuery, pst.EventTracker)
	downtimeParamsUpdaterRaw := pst.StateTracker.RegisterForUpdates(ctx, downtimeParamsUpdater)
	downtimeParamsUpdater, ok := downtimeParamsUpdaterRaw.(*DowntimeParamsUpdater)
	if !ok {
		utils.LavaFormatFatal("invalid updater type returned from RegisterForUpdates", nil, utils.Attribute{Key: "updater", Value: downtimeParamsUpdaterRaw})
	}

	return downtimeParamsUpdater.RegisterDowntimeParamsUpdatable(ctx, &downtimeParamsUpdatable)
}

func (pst *ProviderStateTracker) TxRelayPayment(ctx context.Context, relayRequests []*pairingtypes.RelaySession, description string, latestBlocks []*pairingtypes.LatestBlockReport) error {
	return pst.txSender.TxRelayPayment(ctx, relayRequests, description, latestBlocks)
}

func (pst *ProviderStateTracker) SendVoteReveal(voteID string, vote *reliabilitymanager.VoteData) error {
	return pst.txSender.SendVoteReveal(voteID, vote)
}

func (pst *ProviderStateTracker) SendVoteCommitment(voteID string, vote *reliabilitymanager.VoteData) error {
	return pst.txSender.SendVoteCommitment(voteID, vote)
}

func (pst *ProviderStateTracker) LatestBlock() int64 {
	return pst.StateTracker.chainTracker.GetAtomicLatestBlockNum()
}

func (pst *ProviderStateTracker) GetMaxCuForUser(ctx context.Context, consumerAddress, chainID string, epoch uint64) (maxCu uint64, err error) {
	latestBlockTime := pst.EventTracker.getLatestBlockTime()
	downtimeParams := pst.chainTracker.GetDowntimeParams()

	delay := time.Now().UTC().Sub(latestBlockTime)

	epochDuration := downtimeParams.EpochDuration.Milliseconds()
	virtualEpoch := uint64(0)
	// check if emergency mode is enabled
	if delay > downtimeParams.DowntimeDuration {
		// division delay by epoch duration rounded up, subtract one to skip regular epoch
		virtualEpoch = uint64((delay.Milliseconds()+epochDuration-1)/epochDuration - 1)
	}

	return pst.stateQuery.GetMaxCuForUser(ctx, consumerAddress, chainID, epoch, virtualEpoch)
}

func (pst *ProviderStateTracker) VerifyPairing(ctx context.Context, consumerAddress, providerAddress string, epoch uint64, chainID string) (valid bool, total int64, projectId string, err error) {
	return pst.stateQuery.VerifyPairing(ctx, consumerAddress, providerAddress, epoch, chainID)
}

func (pst *ProviderStateTracker) GetEpochSize(ctx context.Context) (uint64, error) {
	return pst.stateQuery.GetEpochSize(ctx)
}

func (pst *ProviderStateTracker) EarliestBlockInMemory(ctx context.Context) (uint64, error) {
	return pst.stateQuery.EarliestBlockInMemory(ctx)
}

func (pst *ProviderStateTracker) GetRecommendedEpochNumToCollectPayment(ctx context.Context) (uint64, error) {
	return pst.stateQuery.GetRecommendedEpochNumToCollectPayment(ctx)
}

func (pst *ProviderStateTracker) GetEpochSizeMultipliedByRecommendedEpochNumToCollectPayment(ctx context.Context) (uint64, error) {
	return pst.stateQuery.GetEpochSizeMultipliedByRecommendedEpochNumToCollectPayment(ctx)
}

func (pst *ProviderStateTracker) GetProtocolVersion(ctx context.Context) (*ProtocolVersionResponse, error) {
	return pst.stateQuery.GetProtocolVersion(ctx)
}
