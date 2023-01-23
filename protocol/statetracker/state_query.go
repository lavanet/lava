package statetracker

import (
	"context"

	"github.com/cosmos/cosmos-sdk/client"
	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
	pairingtypes "github.com/lavanet/lava/x/pairing/types"
)

type StateQuery interface {
	GetPairing(latestBlock int64) ([]epochstoragetypes.StakeEntry, uint64, uint64, error)
}

type stateQuery struct {
	rpcClient          client.Context
	pairingQueryClient pairingtypes.QueryClient
}

func NewStateQuery(ctx context.Context, clientCtx client.Context) StateQuery {
	// set up the rpcClient necessary to make queries
	return &stateQuery{
		rpcClient:          clientCtx,
		pairingQueryClient: pairingtypes.NewQueryClient(clientCtx),
	}
}

func (sq *stateQuery) GetPairing(latestBlock int64) (pairingList []epochstoragetypes.StakeEntry, epoch uint64, nextBlockForUpdate uint64, err error) {
	// query the node via our clientCtx and run the get pairing query with the client address (in the clientCtx from)
	pairing, err := sq.pairingQueryClient.GetPairing(context.TODO(), &pairingtypes.QueryGetPairingRequest{
		ChainID: sq.rpcClient.ChainID,
		Client:  sq.rpcClient.FromName,
	})

	if err != nil {
		return []epochstoragetypes.StakeEntry{}, 0, 0, err
	}

	return pairing.GetProviders(), pairing.GetCurrentEpoch(), pairing.GetCurrentEpoch() + 1, nil
}
