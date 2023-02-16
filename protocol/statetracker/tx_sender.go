package statetracker

import (
	"bytes"
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/protocol/common"
	"github.com/lavanet/lava/protocol/rpcprovider/reliabilitymanager"
	"github.com/lavanet/lava/utils"
	conflicttypes "github.com/lavanet/lava/x/conflict/types"
)

const (
	defaultGasPrice          = "0.000000001ulava"
	defaultGasAdjustment     = 1.5
	RETRY_INCORRECT_SEQUENCE = 5
)

type TxSender struct {
	txFactory tx.Factory
	clientCtx client.Context
}

func NewTxSender(ctx context.Context, clientCtx client.Context, txFactory tx.Factory) (ret *TxSender, err error) {
	// set up the rpcClient, and factory necessary to make queries
	clientCtx.SkipConfirm = true
	ts := &TxSender{txFactory: txFactory, clientCtx: clientCtx}
	return ts, nil
}

func (ts *TxSender) SimulateAndBroadCastTxWithRetryOnSeqMismatch(msg sdk.Msg) error {
	txfactory := ts.txFactory.WithGasPrices(defaultGasPrice)
	txfactory = txfactory.WithGasAdjustment(defaultGasAdjustment)
	if err := msg.ValidateBasic(); err != nil {
		return err
	}
	clientCtx := ts.clientCtx
	txfactory, err := ts.prepareFactory(txfactory)
	if err != nil {
		return err
	}

	_, gasUsed, err := tx.CalculateGas(clientCtx, txfactory, msg)
	if err != nil {
		return err
	}

	txfactory = txfactory.WithGas(gasUsed)
	myWriter := bytes.Buffer{}
	hasSequenceError := false
	success := false
	idx := -1
	sequenceNumberParsed := 0
	summarizedTransactionResult := ""
	for ; idx < RETRY_INCORRECT_SEQUENCE && !success; idx++ {
		if hasSequenceError { // a retry
			// if sequence number error happened it means that we already sent a tx this block.
			// we need to wait a block for the tx to be approved,
			// only then we can ask for a new sequence number continue and try again.
			var seq uint64
			if sequenceNumberParsed != 0 {
				utils.LavaFormatInfo("Sequence Number extracted from transaction error, retrying", &map[string]string{"sequence": strconv.Itoa(sequenceNumberParsed)})
				seq = uint64(sequenceNumberParsed)
			} else {
				var err error
				_, seq, err = clientCtx.AccountRetriever.GetAccountNumberSequence(clientCtx, clientCtx.GetFromAddress())
				if err != nil {
					utils.LavaFormatError("failed to get correct sequence number for account, give up", err, nil)
					break // give up
				}
			}
			txfactory = txfactory.WithSequence(seq)
			myWriter.Reset()
			utils.LavaFormatInfo("Retrying with sequence number:", &map[string]string{
				"SeqNum": strconv.FormatUint(seq, 10),
			})
		}
		var transactionResult string
		err = tx.GenerateOrBroadcastTxWithFactory(clientCtx, txfactory, msg)
		if err != nil {
			utils.LavaFormatWarning("Sending CheckProfitabilityAndBroadCastTx failed", err, &map[string]string{
				"msg": fmt.Sprintf("%+v", msg),
			})
			transactionResult = err.Error() // incase we got an error the tx result is basically the error
		} else {
			transactionResult = myWriter.String()
		}
		var returnCode int
		summarizedTransactionResult, returnCode = common.ParseTransactionResult(transactionResult)

		if returnCode == 0 { // if we get some other code which isn't 0 then keep retrying
			success = true
		} else if strings.Contains(transactionResult, "account sequence") {
			hasSequenceError = true
			sequenceNumberParsed, err = common.FindSequenceNumber(transactionResult)
			if err != nil {
				utils.LavaFormatWarning("Failed findSequenceNumber", err, &map[string]string{"sequence": transactionResult})
			}
			summarizedTransactionResult = transactionResult
		}
	}
	if !success {
		return utils.LavaFormatError(fmt.Sprintf("failed sending transaction %s", summarizedTransactionResult), nil, nil)
	}
	utils.LavaFormatInfo(fmt.Sprintf("succeeded sending transaction %s", summarizedTransactionResult), nil)
	return nil
}

// this function is extracted from the tx package so that we can use it locally to set the tx factory correctly
func (ts *TxSender) prepareFactory(txf tx.Factory) (tx.Factory, error) {
	clientCtx := ts.clientCtx
	from := clientCtx.GetFromAddress()

	if err := clientCtx.AccountRetriever.EnsureExists(clientCtx, from); err != nil {
		return txf, err
	}

	initNum, initSeq := txf.AccountNumber(), txf.Sequence()
	if initNum == 0 || initSeq == 0 {
		num, seq, err := clientCtx.AccountRetriever.GetAccountNumberSequence(clientCtx, from)
		if err != nil {
			return txf, err
		}

		if initNum == 0 {
			txf = txf.WithAccountNumber(num)
		}

		if initSeq == 0 {
			txf = txf.WithSequence(seq)
		}
	}

	return txf, nil
}

type ConsumerTxSender struct {
	*TxSender
}

func NewConsumerTxSender(ctx context.Context, clientCtx client.Context, txFactory tx.Factory) (ret *ConsumerTxSender, err error) {
	txSender, err := NewTxSender(ctx, clientCtx, txFactory)
	if err != nil {
		return nil, err
	}
	ts := &ConsumerTxSender{TxSender: txSender}
	return ts, nil
}

func (ts *ConsumerTxSender) TxConflictDetection(ctx context.Context, finalizationConflict *conflicttypes.FinalizationConflict, responseConflict *conflicttypes.ResponseConflict, sameProviderConflict *conflicttypes.FinalizationConflict) error {
	// TODO: retry logic for sequence number mismatch
	// TODO: make sure we are not spamming the same conflicts, previous code only detecs relay by relay, it has no state tracking wether it reported already
	msg := conflicttypes.NewMsgDetection(ts.clientCtx.FromAddress.String(), finalizationConflict, responseConflict, sameProviderConflict)
	err := ts.SimulateAndBroadCastTxWithRetryOnSeqMismatch(msg)
	if err != nil {
		return utils.LavaFormatError("discrepancyChecker - SimulateAndBroadCastTx Failed", err, nil)
	}
	return nil
}

type ProviderTxSender struct {
	*TxSender
}

func NewProviderTxSender(ctx context.Context, clientCtx client.Context, txFactory tx.Factory) (ret *ProviderTxSender, err error) {
	txSender, err := NewTxSender(ctx, clientCtx, txFactory)
	if err != nil {
		return nil, err
	}
	ts := &ProviderTxSender{TxSender: txSender}
	return ts, nil
}

func (pts *ProviderTxSender) SendVoteReveal(voteID string, vote *reliabilitymanager.VoteData) error {
	msg := conflicttypes.NewMsgConflictVoteReveal(pts.clientCtx.FromAddress.String(), voteID, vote.Nonce, vote.RelayDataHash)
	err := pts.SimulateAndBroadCastTxWithRetryOnSeqMismatch(msg)
	if err != nil {
		return utils.LavaFormatError("SendVoteReveal - SimulateAndBroadCastTx Failed", err, nil)
	}
	return nil
}

func (pts *ProviderTxSender) SendVoteCommitment(voteID string, vote *reliabilitymanager.VoteData) error {
	msg := conflicttypes.NewMsgConflictVoteCommit(pts.clientCtx.FromAddress.String(), voteID, vote.CommitHash)
	err := pts.SimulateAndBroadCastTxWithRetryOnSeqMismatch(msg)
	if err != nil {
		return utils.LavaFormatError("SendVoteCommitment - SimulateAndBroadCastTx Failed", err, nil)
	}
	return nil
}
