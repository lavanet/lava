package statetracker

import (
	"context"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/tx"
	conflicttypes "github.com/lavanet/lava/x/conflict/types"
)

type TxSender struct{}

func (ts *TxSender) New(ctx context.Context, txFactory tx.Factory, clientCtx client.Context) (ret *TxSender, err error) {
	// set up the rpcClient, and factory necessary to make queries
	return ts, nil
}

func (ts *TxSender) TxConflictDetection(ctx context.Context, finalizationConflict *conflicttypes.FinalizationConflict, responseConflict *conflicttypes.ResponseConflict, sameProviderConflict *conflicttypes.FinalizationConflict) {
	// TODO: send a detection tx, simulate, with retry logic for sequence number mismatch
	// TODO: make sure we are not spamming the same conflicts, previous code only detecs relay by relay, it has no state trackign wether it reported already
}
