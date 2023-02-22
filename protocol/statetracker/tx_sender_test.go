package statetracker

import (
	"context"
	"errors"
	"testing"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type MockAccountRetriever struct {
	client.AccountRetriever
}

func (mar MockAccountRetriever) EnsureExists(clientCtx client.Context, addr sdk.AccAddress) error {
	if addr.String() == sdk.AccAddress([]byte("invalidAddr")).String() {
		return errors.New("error")
	}

	return nil
}

func (mar MockAccountRetriever) GetAccountNumberSequence(clientCtx client.Context, addr sdk.AccAddress) (accNum uint64, accSeq uint64, err error) {
	if addr.String() == sdk.AccAddress([]byte("invalidAddr2")).String() {
		return 10, 15, errors.New("error")
	}

	return 10, 15, nil
}

func TestNewTxSender(t *testing.T) {
	mockClientCtx := client.Context{SkipConfirm: true}
	mockTxFactory := tx.Factory{}

	ts, err := NewTxSender(context.Background(), mockClientCtx, mockTxFactory)

	assert.NotNil(t, ts)
	assert.NoError(t, err)
	assert.Equal(t, mockClientCtx, ts.clientCtx)
	assert.Equal(t, mockTxFactory, ts.txFactory)
}

func TestPrepareFactory(t *testing.T) {
	clientCtx := client.Context{FromAddress: sdk.AccAddress([]byte("from_address")), AccountRetriever: MockAccountRetriever{}}

	flagSet := pflag.NewFlagSet("", pflag.ContinueOnError)
	txf := tx.NewFactoryCLI(clientCtx, flagSet)
	ts := &TxSender{txFactory: txf, clientCtx: clientCtx}

	// Test when account number and sequence are not set
	tfx2 := txf.WithAccountNumber(10).WithSequence(15)
	newTxf, err := ts.prepareFactory(txf)
	require.NoError(t, err)
	require.Equal(t, newTxf, tfx2)

	txf = txf.WithAccountNumber(1).WithSequence(2)
	newTxf, err = ts.prepareFactory(txf)
	require.NoError(t, err)
	require.Equal(t, txf, newTxf)

	clientCtx2 := client.Context{FromAddress: sdk.AccAddress([]byte("invalidAddr")), AccountRetriever: MockAccountRetriever{}}
	ts2 := &TxSender{txFactory: txf, clientCtx: clientCtx2}
	_, err = ts2.prepareFactory(txf)
	require.Error(t, err)

	txf = txf.WithAccountNumber(0).WithSequence(0)
	clientCtx3 := client.Context{FromAddress: sdk.AccAddress([]byte("invalidAddr2")), AccountRetriever: MockAccountRetriever{}}
	ts3 := &TxSender{txFactory: txf, clientCtx: clientCtx3}
	_, err = ts3.prepareFactory(txf)
	require.Error(t, err)
}

func TestNewConsumerTxSender(t *testing.T) {
	clientCtx := client.Context{}
	txFactory := tx.Factory{}
	ctx := context.Background()

	consumerTxSender, err := NewConsumerTxSender(ctx, clientCtx, txFactory)

	require.NoError(t, err)
	require.NotNil(t, consumerTxSender.TxSender)
}
