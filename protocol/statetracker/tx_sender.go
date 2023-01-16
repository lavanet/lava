package statetracker

import (
	"context"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/tx"
)

type TxSender struct{}

func (ts *TxSender) New(ctx context.Context, txFactory tx.Factory, clientCtx client.Context) (ret *TxSender, err error) {
	// set up the rpcClient, and factory necessary to make queries
	return ts, nil
}
