package sentry

import (
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func SimulateAndBroadCastTx(clientCtx client.Context, txf tx.Factory, msg sdk.Msg) error {
	txf = txf.WithSimulateAndExecute(true)
	err := tx.GenerateOrBroadcastTxWithFactory(clientCtx, txf, msg)
	if err != nil {
		return err
	}
	return nil
}
