package sentry

import (
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/utils"
)

func SimulateAndBroadCastTx(clientCtx client.Context, txf tx.Factory, msg sdk.Msg) error {
	txf = txf.WithGasPrices("0.000000001ulava")
	txf = txf.WithGasAdjustment(1.5)
	if err := msg.ValidateBasic(); err != nil {
		return err
	}

	txf, err := prepareFactory(clientCtx, txf)
	if err != nil {
		return err
	}

	_, gasUsed, err := tx.CalculateGas(clientCtx, txf, msg)
	if err != nil {
		return err
	}

	txf = txf.WithGas(gasUsed)

	err = tx.GenerateOrBroadcastTxWithFactory(clientCtx, txf, msg)
	if err != nil {
		return err
	}
	return nil
}

// this function is extracted from the tx package so that we can use it locally to set the tx factory correctly
func prepareFactory(clientCtx client.Context, txf tx.Factory) (tx.Factory, error) {
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

func CheckProfitabilityAndBroadCastTx(clientCtx client.Context, txf tx.Factory, msg sdk.Msg) error {
	txf = txf.WithGasPrices("0.000000001ulava")
	txf = txf.WithGasAdjustment(1.5)
	if err := msg.ValidateBasic(); err != nil {
		return err
	}

	txf, err := prepareFactory(clientCtx, txf)
	if err != nil {
		return err
	}

	simResult, gasUsed, err := tx.CalculateGas(clientCtx, txf, msg)
	if err != nil {
		return err
	}

	txEvents := simResult.GetResult().Events
	lavaReward := sdk.NewCoin("ulava", sdk.NewInt(0))
	for _, txEvent := range txEvents {
		if txEvent.Type == "lava_relay_payment" {
			for _, attribute := range txEvent.Attributes {
				if string(attribute.Key) == "BasePay" {
					lavaRewardTemp, err := sdk.ParseCoinNormalized(string(attribute.Value))
					if err != nil {
						return err
					}
					lavaReward = lavaReward.Add(lavaRewardTemp)
					break
				}
			}
		}
	}

	txf = txf.WithGas(gasUsed)

	gasFee := txf.GasPrices()[0]
	gasFee.Amount = gasFee.Amount.MulInt64(int64(gasUsed))
	lavaRewardDec := sdk.NewDecCoinFromCoin(lavaReward)

	if gasFee.IsGTE(lavaRewardDec) {
		utils.LavaFormatError("lava_relay_payment claim is not profitable", nil, &map[string]string{"gasFee": gasFee.String(), "lavareward:": lavaRewardDec.String(), "msg": msg.String()})
	}

	err = tx.GenerateOrBroadcastTxWithFactory(clientCtx, txf, msg)
	if err != nil {
		return err
	}
	return nil
}
