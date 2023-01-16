package statetracker

import (
	"context"

	"github.com/cosmos/cosmos-sdk/client"
	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
)

type StateQuery struct{}

func (sq *StateQuery) New(ctx context.Context, clientCtx client.Context) (ret *StateQuery, err error) {
	// set up the rpcClient necessary to make queries
	return sq, nil
}

func (sq *StateQuery) GetPairing(latestBlock int64) (pairingList []epochstoragetypes.StakeEntry, epoch uint64, nextBlockForUpdate uint64) {
	// query the node via our clientCtx and run the get pairing query with the client address (in the clientCtx from)
	// latestBlock arg can be used for caching the result
	return
}
