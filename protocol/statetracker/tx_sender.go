package statetracker

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"strconv"
	"strings"
	"time"

	coretypes "github.com/cometbft/cometbft/rpc/core/types"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
	typestx "github.com/cosmos/cosmos-sdk/types/tx"
	"github.com/lavanet/lava/protocol/common"
	"github.com/lavanet/lava/protocol/rpcprovider/reliabilitymanager"
	"github.com/lavanet/lava/utils"
	conflicttypes "github.com/lavanet/lava/x/conflict/types"
	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
	pairingtypes "github.com/lavanet/lava/x/pairing/types"
)

const (
	defaultGasPrice      = "0.000000001" + epochstoragetypes.TokenDenom
	defaultGasAdjustment = 3
	// same account can continue failing the more providers you have under the same account
	// for example if you have a provider staked at 20 chains you will ask for 20 payments per epoch.
	// therefore currently our best solution is to continue retrying increasing sequence number until successful
	RETRY_INCORRECT_SEQUENCE = 100
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

func (ts *TxSender) checkProfitability(simResult *typestx.SimulateResponse, gasUsed uint64, txFactory tx.Factory) error {
	txEvents := simResult.GetResult().Events
	lavaReward := sdk.NewCoin("ulava", sdk.NewInt(0))
	for _, txEvent := range txEvents {
		if txEvent.Type == utils.EventPrefix+pairingtypes.RelayPaymentEventName {
			for _, attribute := range txEvent.Attributes {
				eventStr := attribute.Key
				eventStr = strings.SplitN(eventStr, ".", 2)[0]
				if eventStr == "BasePay" {
					lavaRewardTemp, err := sdk.ParseCoinNormalized(attribute.Value)
					if err != nil {
						return utils.LavaFormatError("failed parsing simulation result", nil, utils.Attribute{Key: "attribute", Value: attribute.Value})
					}
					lavaReward = lavaReward.Add(lavaRewardTemp)
					break
				}
			}
		}
	}

	txFactory = txFactory.WithGas(gasUsed)

	gasFee := txFactory.GasPrices()[0]
	gasFee.Amount = gasFee.Amount.MulInt64(int64(gasUsed))
	lavaRewardDec := sdk.NewDecCoinFromCoin(lavaReward)

	if gasFee.IsGTE(lavaRewardDec) {
		return utils.LavaFormatError("lava_relay_payment claim is not profitable", nil, utils.Attribute{Key: "gasFee", Value: gasFee}, utils.Attribute{Key: "lava_reward:", Value: lavaRewardDec})
	}
	return nil
}

func (ts *TxSender) SimulateAndBroadCastTxWithRetryOnSeqMismatch(msg sdk.Msg, checkProfitability bool) error {
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

	retryWithNewSequenceNumber := false
	success := false
	idx := -1
	sequenceNumberParsed := 0
	latestResult := common.TxResultData{}
	for ; idx < RETRY_INCORRECT_SEQUENCE && !success; idx++ {
		if retryWithNewSequenceNumber { // a retry
			// if sequence number error happened it means that we already sent a tx this block.
			// we need to wait a block for the tx to be approved,
			// only then we can ask for a new sequence number continue and try again.
			var seq uint64
			if sequenceNumberParsed != 0 {
				utils.LavaFormatInfo("Sequence Number extracted from transaction error, retrying", utils.Attribute{Key: "sequence", Value: strconv.Itoa(sequenceNumberParsed)})
				seq = uint64(sequenceNumberParsed)
			} else {
				var err error
				_, seq, err = clientCtx.AccountRetriever.GetAccountNumberSequence(clientCtx, clientCtx.GetFromAddress())
				if err != nil {
					utils.LavaFormatError("failed to get correct sequence number for account, give up", err)
					break // give up
				}
			}
			txfactory = txfactory.WithSequence(seq)
			utils.LavaFormatInfo("Retrying with sequence number:", utils.Attribute{Key: "SeqNum", Value: seq})
			// reset the state
			sequenceNumberParsed = 0
		}

		_, gasUsed, err := tx.CalculateGas(clientCtx, txfactory, msg)
		if err != nil {
			return err
		}
		txfactory = txfactory.WithGas(gasUsed)

		// incase we got an error the tx result is basically the error

		latestResult, err = ts.SendTxAndVerifyCommit(txfactory, msg)
		transactionResult := latestResult.RawLog
		if err == nil { // if we get some other code which isn't 0 then keep retrying
			success = true
			break
		} else if strings.Contains(transactionResult, "account sequence") {
			retryWithNewSequenceNumber = true
			sequenceNumberParsed, err = common.FindSequenceNumber(transactionResult)
			if err != nil {
				utils.LavaFormatWarning("Failed findSequenceNumber", err, utils.Attribute{Key: "sequence", Value: transactionResult})
			}
		} else if strings.Contains(transactionResult, "out of gas") {
			utils.LavaFormatInfo("Transaction got out of gas error, retrying next block.")
			retryWithNewSequenceNumber = true // retry with out of gas issue
		} else if strings.Contains(transactionResult, "insufficient fees; got:") { //
			err := parseInsufficientFeesError(transactionResult, gasUsed)
			if err == nil {
				return utils.LavaFormatError("Failed sending transaction", nil, utils.Attribute{Key: "result", Value: latestResult})
			}
		}
	}
	if !success {
		return utils.LavaFormatError("Failed sending transaction with all retries and giving up", nil, utils.Attribute{Key: "result", Value: latestResult}, utils.Attribute{Key: "Number Of Retries executed", Value: idx}, utils.Attribute{Key: "Parsed Sequence", Value: sequenceNumberParsed})
	}
	utils.LavaFormatInfo("Succeeded sending transaction", utils.Attribute{Key: "hash", Value: hex.EncodeToString(latestResult.Txhash)})
	return nil
}

func (ts *TxSender) SendTxAndVerifyCommit(txfactory tx.Factory, msg sdk.Msg) (parsedResult common.TxResultData, err error) {
	myWriter := bytes.Buffer{}
	clientCtx := ts.clientCtx
	clientCtx.Output = &myWriter
	clientCtx.OutputFormat = "json"
	err = tx.GenerateOrBroadcastTxWithFactory(clientCtx, txfactory, msg)
	if err != nil {
		utils.LavaFormatWarning("Sending CheckProfitabilityAndBroadCastTx failed", err, utils.Attribute{Key: "msg", Value: msg})
		return common.TxResultData{}, err
	}
	jsonParsedResult := map[string]any{}
	err = json.Unmarshal(myWriter.Bytes(), &jsonParsedResult)
	if err != nil {
		return common.TxResultData{}, utils.LavaFormatInfo("Failed unmarshaling transaction results", utils.Attribute{Key: "transactionResult", Value: myWriter.String()})
	}
	myWriter.Reset()
	if debug {
		utils.LavaFormatDebug("transaction results", utils.Attribute{Key: "jsonParsedResult", Value: jsonParsedResult})
	}
	resultData, err := common.ParseTransactionResult(jsonParsedResult)
	if err != nil {
		return common.TxResultData{}, err
	}
	if resultData.Code != 0 {
		return resultData, utils.LavaFormatInfo("Failed sending transaction, code is not 0", utils.Attribute{Key: "resultData", Value: resultData})
	}
	// now that our Tx was sent to the mempool successfully, we want to see it's result on chain
	resultData, err = ts.waitForTxCommit(resultData)
	return resultData, err
}

func (ts *TxSender) waitForTxCommit(resultData common.TxResultData) (common.TxResultData, error) {
	clientCtx := ts.clientCtx
	txResultChan := make(chan *coretypes.ResultTx)
	guid := utils.GenerateUniqueIdentifier()
	// check consumer session manager
	go func() {
		for {
			ctx, cancel := context.WithTimeout(utils.WithUniqueIdentifier(context.Background(), guid), 3*time.Second)
			result, err := clientCtx.Client.Tx(ctx, resultData.Txhash, false)
			cancel()
			if err == nil {
				txResultChan <- result
				return
			}
			if debug {
				utils.LavaFormatWarning("Tx query got error", err, utils.Attribute{Key: "GUID", Value: ctx}, utils.Attribute{Key: "resultData", Value: resultData})
			}
			time.Sleep(5 * time.Second)
		}
	}()
	select {
	case txRes := <-txResultChan:
		resultData = common.TxResultData{
			RawLog: txRes.TxResult.Log,
			Txhash: resultData.Txhash,
			Code:   int(txRes.TxResult.Code),
		}
		break
	case <-time.After(5 * time.Minute):
		return common.TxResultData{}, utils.LavaFormatError("failed sending tx, wasn't found after timeout", nil, utils.Attribute{Key: "prev resultData", Value: resultData})
	}
	// we found the tx on chain and it failed
	if resultData.Code != 0 {
		return resultData, utils.LavaFormatInfo("Failed sending transaction, code is not 0", utils.Attribute{Key: "result", Value: resultData})
	}
	return resultData, nil
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

func (ts *ConsumerTxSender) TxSenderConflictDetection(ctx context.Context, finalizationConflict *conflicttypes.FinalizationConflict, responseConflict *conflicttypes.ResponseConflict, sameProviderConflict *conflicttypes.FinalizationConflict) error {
	msg := conflicttypes.NewMsgDetection(ts.clientCtx.FromAddress.String(), finalizationConflict, responseConflict, sameProviderConflict)
	err := ts.SimulateAndBroadCastTxWithRetryOnSeqMismatch(msg, false)
	if err != nil {
		return utils.LavaFormatError("discrepancyChecker - SimulateAndBroadCastTx Failed", err)
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

func (pts *ProviderTxSender) TxRelayPayment(ctx context.Context, relayRequests []*pairingtypes.RelaySession, description string, latestBlocks []*pairingtypes.LatestBlockReport) error {
	msg := pairingtypes.NewMsgRelayPayment(pts.clientCtx.FromAddress.String(), relayRequests, description, latestBlocks)
	err := pts.SimulateAndBroadCastTxWithRetryOnSeqMismatch(msg, true)
	if err != nil {
		return utils.LavaFormatError("relay_payment - sending Tx Failed", err)
	}
	return nil
}

func (pts *ProviderTxSender) SendVoteReveal(voteID string, vote *reliabilitymanager.VoteData) error {
	msg := conflicttypes.NewMsgConflictVoteReveal(pts.clientCtx.FromAddress.String(), voteID, vote.Nonce, vote.RelayDataHash)
	err := pts.SimulateAndBroadCastTxWithRetryOnSeqMismatch(msg, false)
	if err != nil {
		return utils.LavaFormatError("SendVoteReveal - SimulateAndBroadCastTx Failed", err)
	}
	return nil
}

func (pts *ProviderTxSender) SendVoteCommitment(voteID string, vote *reliabilitymanager.VoteData) error {
	msg := conflicttypes.NewMsgConflictVoteCommit(pts.clientCtx.FromAddress.String(), voteID, vote.CommitHash)
	err := pts.SimulateAndBroadCastTxWithRetryOnSeqMismatch(msg, false)
	if err != nil {
		return utils.LavaFormatError("SendVoteCommitment - SimulateAndBroadCastTx Failed", err)
	}
	return nil
}

func parseInsufficientFeesError(msg string, gasUsed uint64) error {
	feesPart := strings.Split(msg, "insufficient fees; got: ")[1]
	prices := strings.Split(feesPart, epochstoragetypes.TokenDenom)
	var required int
	var err error
	for _, p := range prices {
		if strings.Contains(p, " required: ") {
			requiredParsedString := strings.Split(p, " required: ")[1]
			required, err = strconv.Atoi(requiredParsedString)
			if err != nil {
				return utils.LavaFormatError("Failed converting string to number", err, utils.Attribute{Key: "requiredParsedString", Value: requiredParsedString})
			}
		}
	}
	if required == 0 {
		return utils.LavaFormatError("Failed fetching required gas from error", nil, utils.Attribute{Key: "message", Value: prices})
	}
	minimumGasPricesGot := (float64(gasUsed) / float64(required))
	utils.LavaFormatError("Bad Lava Node Configuration detected, Gas fees inconsistencies can be related to the app.toml configuration of the lava node you are using under 'minimum-gas-prices', Please remove the field or set it to the required amount or change rpc to a different lava node", nil,
		utils.Attribute{Key: "Required Minimum Gas Prices", Value: defaultGasPrice},
		utils.Attribute{Key: "Current (estimated) Minimum Gas Prices", Value: strconv.FormatFloat(minimumGasPricesGot, 'f', -1, 64) + epochstoragetypes.TokenDenom},
	)

	return nil
}
