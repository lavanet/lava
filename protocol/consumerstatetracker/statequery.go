package consumerstatetracker

import (
	"context"

	"github.com/cosmos/cosmos-sdk/client"
)

type StateQuery struct {
}

func (sq *StateQuery) New(ctx context.Context, clientCtx client.Context) (ret *StateQuery, err error) {
	// set up the rpcClient necessary to make queries
	return sq, nil
}
