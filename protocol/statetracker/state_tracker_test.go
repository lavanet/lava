package statetracker

import (
	"context"
	"strconv"
	"testing"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/lavanet/lava/protocol/lavasession"
	"github.com/stretchr/testify/assert"
	"github.com/tendermint/tendermint/libs/service"
	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"
	tendermintClinet "github.com/tendermint/tendermint/rpc/client"
	ctypes "github.com/tendermint/tendermint/rpc/core/types"
)

type MockChainFetcher struct {
}

func (mcf MockChainFetcher) FetchLatestBlockNum(ctx context.Context) (int64, error) {
	return 0, nil
}

func (mcf MockChainFetcher) FetchBlockHashByNum(ctx context.Context, blockNum int64) (string, error) {
	return "", nil
}

func (mcf MockChainFetcher) FetchEndpoint() lavasession.RPCProviderEndpoint {
	return lavasession.RPCProviderEndpoint{}
}

type MockUpdater struct {
	Block int64
}

func (mu MockUpdater) Update(num int64) {
	mu.Block = num
}

func (mu MockUpdater) UpdaterKey() string {
	return strconv.Itoa(int(mu.Block))
}

type MockContextClient struct {
	service.Service
	tendermintClinet.NetworkClient
	tendermintClinet.ABCIClient
	tendermintClinet.EventsClient
	tendermintClinet.HistoryClient
	tendermintClinet.SignClient
	tendermintClinet.StatusClient
	tendermintClinet.EvidenceClient
	tendermintClinet.MempoolClient
}

func (mcc MockContextClient) ConsensusParams(ctx context.Context, height *int64) (*ctypes.ResultConsensusParams, error) {

	return &ctypes.ResultConsensusParams{
		BlockHeight: 100,
		ConsensusParams: tmproto.ConsensusParams{
			Block: tmproto.BlockParams{
				TimeIotaMs: 10,
			},
		},
	}, nil
}

func newStateTracker() (ret *StateTracker, err error) {
	ctx := context.Background()
	txFactory := tx.Factory{}
	clientCtx := client.Context{Client: MockContextClient{}}
	chainFetcher := MockChainFetcher{}

	cst, err := NewStateTracker(ctx, txFactory, clientCtx, chainFetcher)

	return cst, err
}

func TestStateTracker_newLavaBlock(t *testing.T) {
	st, err := newStateTracker()

	assert.Nil(t, err)

	st.newLavaBlockUpdaters = map[string]Updater{
		"updater1": &MockUpdater{},
	}

	st.newLavaBlock(0)

	assert.Equal(t, st.newLavaBlockUpdaters["updater1"].UpdaterKey(), strconv.Itoa(int(0)))
}

func TestStateTracker_RegisterForUpdates(t *testing.T) {
	st, err := newStateTracker()

	assert.Nil(t, err)

	mockUpdater := MockUpdater{Block: 50}

	// First time register updater
	existingUpdater := st.RegisterForUpdates(context.Background(), mockUpdater)

	assert.Equal(t, mockUpdater, existingUpdater)

	// It should already exists
	existingUpdater = st.RegisterForUpdates(context.Background(), mockUpdater)

	assert.Equal(t, mockUpdater, existingUpdater)

}
