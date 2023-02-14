package statetracker

import (
	"context"
	"fmt"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/lavanet/lava/protocol/chainlib"
	"github.com/lavanet/lava/protocol/chaintracker"
	"github.com/lavanet/lava/protocol/lavasession"
	"github.com/lavanet/lava/protocol/rpcprovider/reliabilitymanager"
	"github.com/lavanet/lava/utils"
	pairingtypes "github.com/lavanet/lava/x/pairing/types"
)

// ProviderStateTracker PST is a class for tracking provider data from the lava blockchain, such as epoch changes.
// it allows also to query specific data form the blockchain and acts as a single place to send transactions
type ProviderStateTracker struct {
	stateQuery *ProviderStateQuery
	txSender   *ProviderTxSender
	*StateTracker
}

func NewProviderStateTracker(ctx context.Context, txFactory tx.Factory, clientCtx client.Context, chainFetcher chaintracker.ChainFetcher) (ret *ProviderStateTracker, err error) {
	stateTrackerBase, err := NewStateTracker(ctx, txFactory, clientCtx, chainFetcher)
	if err != nil {
		return nil, err
	}
	txSender, err := NewProviderTxSender(ctx, clientCtx, txFactory)
	if err != nil {
		return nil, err
	}
	pst := &ProviderStateTracker{StateTracker: stateTrackerBase, stateQuery: NewProviderStateQuery(ctx, clientCtx), txSender: txSender}
	return pst, nil
}

func (pst *ProviderStateTracker) RegisterForEpochUpdates(ctx context.Context, epochUpdatable EpochUpdatable) {
	epochUpdater := NewEpochUpdater(pst.stateQuery)
	epochUpdaterRaw := pst.StateTracker.RegisterForUpdates(ctx, epochUpdater)
	epochUpdater, ok := epochUpdaterRaw.(*EpochUpdater)
	if !ok {
		utils.LavaFormatFatal("invalid updater type returned from RegisterForUpdates", nil, &map[string]string{"updater": fmt.Sprintf("%+v", epochUpdaterRaw)})
	}
	epochUpdater.RegisterEpochUpdatable(ctx, epochUpdatable)
}

func (pst *ProviderStateTracker) RegisterChainParserForSpecUpdates(ctx context.Context, chainParser chainlib.ChainParser, chainID string) error {
	spec, err := pst.stateQuery.GetSpec(ctx, chainID)
	if err != nil {
		return err
	}
	chainParser.SetSpec(*spec)
	return nil
}

func (pst *ProviderStateTracker) RegisterReliabilityManagerForVoteUpdates(ctx context.Context, voteUpdatable VoteUpdatable, endpointP *lavasession.RPCProviderEndpoint) {
	voteUpdater := NewVoteUpdater(pst.stateQuery)
	voteUpdaterRaw := pst.StateTracker.RegisterForUpdates(ctx, voteUpdater)
	voteUpdater, ok := voteUpdaterRaw.(*VoteUpdater)
	if !ok {
		utils.LavaFormatFatal("invalid updater type returned from RegisterForUpdates", nil, &map[string]string{"updater": fmt.Sprintf("%+v", voteUpdaterRaw)})
	}
	endpoint := lavasession.RPCEndpoint{ChainID: endpointP.ChainID, ApiInterface: endpointP.ApiInterface}
	voteUpdater.RegisterVoteUpdatable(ctx, &voteUpdatable, endpoint)
}

func (pst *ProviderStateTracker) QueryVerifyPairing(ctx context.Context, consumer string, blockHeight uint64) {
	// TODO: implement
}

func (pst *ProviderStateTracker) TxRelayPayment(ctx context.Context, relayRequests []*pairingtypes.RelayRequest) {
	// TODO: implement
}

func (pst *ProviderStateTracker) SendVoteReveal(voteID string, vote *reliabilitymanager.VoteData) error {
	return pst.txSender.SendVoteReveal(voteID, vote)
}
func (pst *ProviderStateTracker) SendVoteCommitment(voteID string, vote *reliabilitymanager.VoteData) error {
	return pst.txSender.SendVoteCommitment(voteID, vote)
}

func (pst *ProviderStateTracker) LatestBlock() int64 {
	return pst.StateTracker.chainTracker.GetLatestBlockNum()
}

func (pst *ProviderStateTracker) GetVrfPkAndMaxCuForUser(ctx context.Context, consumerAddress string, chainID string, epoch uint64) (vrfPk *utils.VrfPubKey, maxCu uint64, err error) {
	return pst.stateQuery.GetVrfPkAndMaxCuForUser(ctx, consumerAddress, chainID, epoch)
}

func (pst *ProviderStateTracker) VerifyPairing(ctx context.Context, consumerAddress string, providerAddress string, epoch uint64, chainID string) (valid bool, index int64, err error) {
	return pst.stateQuery.VerifyPairing(ctx, consumerAddress, providerAddress, epoch, chainID)
}

func (pst *ProviderStateTracker) GetProvidersCountForConsumer(ctx context.Context, consumerAddress string, epoch uint64, chainID string) (uint32, error) {
	return pst.stateQuery.GetProvidersCountForConsumer(ctx, consumerAddress, epoch, chainID)
}
