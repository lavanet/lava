package statetracker

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/goccy/go-json"

	coretypes "github.com/cometbft/cometbft/rpc/core/types"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
	typestx "github.com/cosmos/cosmos-sdk/types/tx"
	"github.com/cosmos/cosmos-sdk/x/feegrant"
	"github.com/lavanet/lava/v2/protocol/common"
	"github.com/lavanet/lava/v2/protocol/rpcprovider/reliabilitymanager"
	updaters "github.com/lavanet/lava/v2/protocol/statetracker/updaters"
	"github.com/lavanet/lava/v2/utils"
	commontypes "github.com/lavanet/lava/v2/utils/common/types"
	conflicttypes "github.com/lavanet/lava/v2/x/conflict/types"
	pairingtypes "github.com/lavanet/lava/v2/x/pairing/types"
)

const (
	DefaultGasPrice      = "0.00002" + commontypes.TokenDenom
	DefaultGasAdjustment = "3.0"
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
	lavaReward := sdk.NewCoin(commontypes.TokenDenom, sdk.NewInt(0))
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

func (ts *TxSender) SimulateAndBroadCastTxWithRetryOnSeqMismatch(ctx context.Context, msg sdk.Msg, checkProfitability bool, feeGranter sdk.AccAddress) error {
	txfactory := ts.txFactory
	if feeGranter != nil {
		txfactory = ts.txFactory.WithFeeGranter(feeGranter)
	}

	if err := msg.ValidateBasic(); err != nil {
		return err
	}
	clientCtx := ts.clientCtx
	txfactory, err := ts.prepareFactory(txfactory)
	if err != nil {
		return err
	}

	success := false
	idx := -1
	sequenceNumberParsed := 0
	latestResult := common.TxResultData{}
	var gasUsed uint64
	for ; idx < RETRY_INCORRECT_SEQUENCE && !success; idx++ {
		utils.LavaFormatDebug("Attempting to send relay payment transaction", utils.LogAttr("index", idx+1))
		txfactory, gasUsed, err = ts.simulateTxWithRetry(clientCtx, txfactory, msg)
		if err != nil {
			return utils.LavaFormatError("Failed Simulating transaction", err)
		}
		// incase we got an error the tx result is basically the error
		latestResult, err = ts.SendTxAndVerifyCommit(txfactory, msg)
		transactionResult := latestResult.RawLog
		if err == nil { // if we get some other code which isn't 0 then keep retrying
			success = true
			break
		} else if strings.Contains(transactionResult, "out of gas") {
			utils.LavaFormatInfo("Transaction got out of gas error, retrying next block.")
		} else {
			txfactory, err = ts.parseTxErrorsAndTryGettingANewFactory(transactionResult, clientCtx, txfactory, gasUsed)
			if err != nil {
				return utils.LavaFormatError("Failed getting a new tx factory", err)
			}
			// else continue with the new factory
		}
		utils.LavaFormatDebug("Failed sending transaction, will retry", utils.LogAttr("Index", idx), utils.LogAttr("reason:", err), utils.LogAttr("rawLog", transactionResult))
	}
	if !success {
		return utils.LavaFormatError("Failed sending transaction with all retries and giving up", nil, utils.Attribute{Key: "result", Value: latestResult}, utils.Attribute{Key: "Number Of Retries executed", Value: idx}, utils.Attribute{Key: "Parsed Sequence", Value: sequenceNumberParsed})
	}
	utils.LavaFormatInfo("Succeeded sending transaction", utils.Attribute{Key: "hash", Value: hex.EncodeToString(latestResult.Txhash)})
	return nil
}

func (ts *TxSender) getSequenceNumberFromErrorOrClient(clientCtx client.Context, errString string) (uint64, error) {
	sequenceNumberParsed, err := common.FindSequenceNumber(errString)
	if err != nil {
		utils.LavaFormatWarning("getSequenceNumberFromErrorOrClient: Failed to findSequenceNumber, fetching from client", err)
		_, seq, err := clientCtx.AccountRetriever.GetAccountNumberSequence(clientCtx, clientCtx.GetFromAddress())
		if err != nil {
			return 0, utils.LavaFormatError("failed to get correct sequence number for account, give up", err)
		}
		return seq, nil
	}
	return uint64(sequenceNumberParsed), nil
}

// this method will attempt to create a new tx factory from a sequence number error provided the expected sequence number is inside the error string
// the new factory should succeed in executing the tx
func (ts *TxSender) getNewFactoryFromASequenceNumberError(errString string, txfactory tx.Factory, clientCtx client.Context) (tx.Factory, error) {
	sequence, errSequence := ts.getSequenceNumberFromErrorOrClient(clientCtx, errString)
	if errSequence != nil {
		return txfactory, errSequence
	}
	utils.LavaFormatDebug("Retrying with new sequence factory", utils.LogAttr("sequence", sequence))
	return txfactory.WithSequence(sequence), nil
}

// trying to fetch sequence number required from the tx error and creates a new factory using this sequence
func (ts *TxSender) parseTxErrorsAndTryGettingANewFactory(txResultString string, clientCtx client.Context, txfactory tx.Factory, gasUsed uint64) (tx.Factory, error) {
	if strings.Contains(txResultString, "account sequence") { // case for more than one tx in a block
		utils.LavaFormatInfo("Identified account sequence reason, attempting to run a new simulation with the correct sequence number")
		return ts.getNewFactoryFromASequenceNumberError(txResultString, txfactory, clientCtx)
	} else if strings.Contains(txResultString, "insufficient fees; got:") { // handle a case where node minimum gas fees is misconfigured
		return ts.txFactory, parseInsufficientFeesError(txResultString, gasUsed)
	}
	return txfactory, fmt.Errorf(txResultString)
}

func (ts *TxSender) simulateTxWithRetry(clientCtx client.Context, txfactory tx.Factory, msg sdk.Msg) (tx.Factory, uint64, error) {
	for retrySimulation := 0; retrySimulation < RETRY_INCORRECT_SEQUENCE; retrySimulation++ {
		utils.LavaFormatDebug("Running Simulation", utils.LogAttr("idx", retrySimulation))
		_, gasUsed, err := tx.CalculateGas(clientCtx, txfactory, msg)
		if err != nil {
			utils.LavaFormatInfo("Simulation failed", utils.LogAttr("reason:", err))
			errString := err.Error()
			var errParsed error
			txfactory, errParsed = ts.parseTxErrorsAndTryGettingANewFactory(errString, clientCtx, txfactory, gasUsed)
			if errParsed != nil {
				return txfactory, 0, errParsed
			}
			continue // we errored, we will retry if parseTxErrors managed to get a new factory
		}
		txfactory = txfactory.WithGas(gasUsed)
		return txfactory, gasUsed, nil
	}
	return txfactory, 0, utils.LavaFormatError("Failed Calculating gas for reward transaction", nil)
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
	utils.LavaFormatInfo("Sent Transaction", utils.LogAttr("Hash", hex.EncodeToString(resultData.Txhash)))
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
	timeOutReached := false
	go func() {
		for {
			// we will never catch the tx hash in the first attempt as not enough time have passed, so we sleep at the beginning of the loop
			time.Sleep(5 * time.Second)
			if timeOutReached {
				utils.LavaFormatWarning("Timeout waiting for transaction", nil, utils.LogAttr("hash", resultData.Txhash))
				return
			}
			ctx, cancel := context.WithTimeout(utils.WithUniqueIdentifier(context.Background(), guid), 5*time.Second)
			result, err := clientCtx.Client.Tx(ctx, resultData.Txhash, false)
			cancel()
			if err == nil {
				utils.LavaFormatDebug("Tx Found successfully on chain!", utils.LogAttr("Hash", hex.EncodeToString(resultData.Txhash)))
				txResultChan <- result
				return
			}
			utils.LavaFormatDebug("Keep Waiting tx results...", utils.LogAttr("reason", err))
			if debug {
				utils.LavaFormatWarning("Tx query got error", err, utils.Attribute{Key: "GUID", Value: ctx}, utils.Attribute{Key: "resultData", Value: resultData})
			}
		}
	}()
	select {
	case txRes := <-txResultChan:
		resultData = common.TxResultData{
			RawLog: txRes.TxResult.Log,
			Txhash: resultData.Txhash,
			Code:   int(txRes.TxResult.Code),
		}
		utils.LavaFormatDebug("Tx Hash found on blockchain", utils.LogAttr("Txhash", hex.EncodeToString(resultData.Txhash)), utils.LogAttr("Code", resultData.Code))
		break
	case <-time.After(5 * time.Minute):
		timeOutReached = true
		return common.TxResultData{}, utils.LavaFormatError("failed sending tx, wasn't found after timeout", nil, utils.Attribute{Key: "hash", Value: hex.EncodeToString(resultData.Txhash)})
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
	err := ts.SimulateAndBroadCastTxWithRetryOnSeqMismatch(ctx, msg, false, nil)
	if err != nil {
		return utils.LavaFormatError("discrepancyChecker - SimulateAndBroadCastTx Failed", err)
	}
	return nil
}

type ProviderTxSender struct {
	*TxSender
	vaults     map[string]sdk.AccAddress // chain ID -> vault that is a fee granter
	vaultsLock sync.RWMutex
}

func NewProviderTxSender(ctx context.Context, clientCtx client.Context, txFactory tx.Factory) (ret *ProviderTxSender, err error) {
	txSender, err := NewTxSender(ctx, clientCtx, txFactory)
	if err != nil {
		return nil, err
	}
	ts := &ProviderTxSender{TxSender: txSender}
	ctxWithTimeout, cancel := context.WithTimeout(ctx, updaters.TimeOutForFetchingLavaBlocks)
	defer cancel()
	ts.vaults, err = ts.getVaults(ctxWithTimeout)
	if err != nil {
		utils.LavaFormatInfo("failed getting vaults from chain, could be provider is not staked yet", utils.LogAttr("reason", err))
	}
	return ts, nil
}

func (pts *ProviderTxSender) UpdateEpoch(epoch uint64) {
	// need to update vault information
	ctx, cancel := context.WithTimeout(context.Background(), updaters.TimeOutForFetchingLavaBlocks)
	defer cancel()
	vaults, err := pts.getVaults(ctx)
	if err != nil {
		// failed getting vaults, try again next epoch
		return
	}
	utils.LavaFormatDebug("ProviderTxSender got epoch update, updating vaults", utils.LogAttr("new vaults", vaults), utils.LogAttr("epoch", epoch))
	pts.vaultsLock.Lock()
	defer pts.vaultsLock.Unlock()
	pts.vaults = vaults
}

func (pts *ProviderTxSender) getVaults(ctx context.Context) (map[string]sdk.AccAddress, error) {
	vaults := map[string]sdk.AccAddress{}
	pairingQuerier := pairingtypes.NewQueryClient(pts.clientCtx)
	res, err := pairingQuerier.Provider(ctx, &pairingtypes.QueryProviderRequest{Address: pts.clientCtx.FromAddress.String()})
	if err != nil {
		utils.LavaFormatWarning("could not get provider vault addresses. provider is not staked on any chain", err,
			utils.LogAttr("provider", pts.clientCtx.FromAddress.String()),
		)
		return vaults, err
	}

	// this can happen if the provider is not staked yet
	if len(res.StakeEntries) == 0 {
		return vaults, utils.LavaFormatWarning("couldn't find entries on chain - could be that the provider is not staked yet", nil)
	}

	feegrantQuerier := feegrant.NewQueryClient(pts.clientCtx)
	for _, stakeEntry := range res.StakeEntries {
		if stakeEntry.Vault == stakeEntry.Address {
			// if provider == vault, there is no feegrant, skip the stake entry
			continue
		}
		vaultAcc, err := sdk.AccAddressFromBech32(stakeEntry.Vault)
		if err != nil {
			utils.LavaFormatError("critical: invalid vault address in stake entry", err,
				utils.LogAttr("provider", pts.clientCtx.FromAddress.String()),
				utils.LogAttr("vault", stakeEntry.Vault),
				utils.LogAttr("chain_id", stakeEntry.Chain),
			)
			continue
		}
		res, err := feegrantQuerier.Allowance(ctx, &feegrant.QueryAllowanceRequest{
			Granter: stakeEntry.Vault,
			Grantee: pts.clientCtx.FromAddress.String(),
		})
		if err != nil {
			// Allowance doesn't return an error if allowance doesn't exist. Error is returned on parsing issues
			utils.LavaFormatWarning("could not get gas fee allowance for provider and vault", err,
				utils.LogAttr("provider", pts.clientCtx.FromAddress.String()),
				utils.LogAttr("vault", stakeEntry.Vault),
				utils.LogAttr("chain_id", stakeEntry.Chain),
			)
			continue
		}
		if res.Allowance != nil {
			vaults[stakeEntry.Chain] = vaultAcc
		}
	}

	return vaults, nil
}

func (pts *ProviderTxSender) getFeeGranterFromVaults(chainId string) sdk.AccAddress {
	pts.vaultsLock.RLock()
	defer pts.vaultsLock.RUnlock()
	if len(pts.vaults) == 0 {
		return nil
	}

	feeGranter, ok := pts.vaults[chainId]
	if ok {
		return feeGranter
	}
	// else get the first fee granter
	for _, val := range pts.vaults {
		return val
	}

	return nil
}

func (pts *ProviderTxSender) TxRelayPayment(ctx context.Context, relayRequests []*pairingtypes.RelaySession, description string, latestBlocks []*pairingtypes.LatestBlockReport) error {
	msg := pairingtypes.NewMsgRelayPayment(pts.clientCtx.FromAddress.String(), relayRequests, description, latestBlocks)
	utils.LavaFormatDebug("Sending reward TX", utils.LogAttr("Number_of_relay_sessions_for_payment", len(relayRequests)))
	feeGranter := pts.getFeeGranterFromVaults("")
	err := pts.SimulateAndBroadCastTxWithRetryOnSeqMismatch(ctx, msg, true, feeGranter)
	if err != nil {
		return utils.LavaFormatError("relay_payment - sending Tx Failed", err)
	}
	return nil
}

func (pts *ProviderTxSender) SendVoteReveal(ctx context.Context, voteID string, vote *reliabilitymanager.VoteData, specId string) error {
	msg := conflicttypes.NewMsgConflictVoteReveal(pts.clientCtx.FromAddress.String(), voteID, vote.Nonce, vote.RelayDataHash)
	feeGranter := pts.getFeeGranterFromVaults(specId)
	err := pts.SimulateAndBroadCastTxWithRetryOnSeqMismatch(ctx, msg, false, feeGranter)
	if err != nil {
		return utils.LavaFormatError("SendVoteReveal - SimulateAndBroadCastTx Failed", err)
	}
	return nil
}

func (pts *ProviderTxSender) SendVoteCommitment(ctx context.Context, voteID string, vote *reliabilitymanager.VoteData, specId string) error {
	msg := conflicttypes.NewMsgConflictVoteCommit(pts.clientCtx.FromAddress.String(), voteID, vote.CommitHash)
	feeGranter := pts.getFeeGranterFromVaults(specId)
	err := pts.SimulateAndBroadCastTxWithRetryOnSeqMismatch(ctx, msg, false, feeGranter)
	if err != nil {
		return utils.LavaFormatError("SendVoteCommitment - SimulateAndBroadCastTx Failed", err)
	}
	return nil
}

func parseInsufficientFeesError(msg string, gasUsed uint64) error {
	feesPart := strings.Split(msg, "insufficient fees; got: ")[1]
	prices := strings.Split(feesPart, commontypes.TokenDenom)
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
	return utils.LavaFormatError("Bad Lava Node Configuration detected, Gas fees inconsistencies can be related to the app.toml configuration of the lava node you are using under 'minimum-gas-prices', Please remove the field or set it to the required amount or change rpc to a different lava node", nil,
		utils.Attribute{Key: "Required Minimum Gas Prices", Value: DefaultGasPrice},
		utils.Attribute{Key: "Current (estimated) Minimum Gas Prices", Value: strconv.FormatFloat(minimumGasPricesGot, 'f', -1, 64) + commontypes.TokenDenom},
	)
}
