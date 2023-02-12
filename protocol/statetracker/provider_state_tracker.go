package statetracker

import (
	"context"
	"fmt"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/lavanet/lava/protocol/chainlib"
	"github.com/lavanet/lava/protocol/chaintracker"
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

func (pst *ProviderStateTracker) RegisterForEpochUpdates(ctx context.Context, epochUpdatable *EpochUpdatable) {
	// create an epoch updater
	// add epoch updater to the updater map
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

func (pst *ProviderStateTracker) RegisterReliabilityManagerForVoteUpdates(ctx context.Context, reliabilityManager *reliabilitymanager.ReliabilityManager) {
	// TODO: change to an interface instead of reliabilitymanager.ReliabilityManager
}

func (pst *ProviderStateTracker) QueryVerifyPairing(ctx context.Context, consumer string, blockHeight uint64) {
	// TODO: implement
}

func (pst *ProviderStateTracker) TxRelayPayment(ctx context.Context, relayRequests []*pairingtypes.RelayRequest) {
	// TODO: implement
}
