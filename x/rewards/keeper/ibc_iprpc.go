package keeper

import (
	"errors"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	transfertypes "github.com/cosmos/ibc-go/v7/modules/apps/transfer/types"
	"github.com/lavanet/lava/utils"
	"github.com/lavanet/lava/utils/decoder"
	"github.com/lavanet/lava/x/rewards/types"
)

/*

The rewards module (which acts as an IBC middleware) analyzes incoming ibc-transfer packets and checks their memo field.
If the memo field is in the IPRPC over IBC format, it uses the tokens from the packet and saves them for a future fund to
the IPRPC pool.

An example of the expected IPRPC over IBC memo field:
{
  "iprpc": {
    "creator": "my-moniker",
    "spec": "ETH1",
    "duration": 3
  }
}

The tokens will be transferred to the pool once the minimum IPRPC funding fee is paid. In the meantime, the IPRPC over IBC
funds are saved in the IbcIprpcFund scaffolded map.

*/

// ExtractIprpcMemoFromPacket extracts the memo field from an ibc-transfer packet and verifies that it's in the right format
// and holds valid values. If the memo is not in the right format, a custom error is returned so the packet will be skipped and
// passed to the next IBC module in the transfer stack normally (and not return an error ack)
func (k Keeper) ExtractIprpcMemoFromPacket(ctx sdk.Context, transferData transfertypes.FungibleTokenPacketData) (types.IprpcMemo, error) {
	var memo types.IprpcMemo
	err := decoder.Decode(transferData.Memo, "iprpc", &memo, nil, nil, nil)
	if err != nil {
		// memo is not for IPRPC over IBC, return custom error to skip processing for this packet
		utils.LavaFormatWarning("memo is not in IPRPC over IBC format", err, utils.LogAttr("memo", memo))
		return types.IprpcMemo{}, types.ErrMemoNotIprpcOverIbc
	}
	err = k.validateIprpcMemo(ctx, memo)
	if err != nil {
		return types.IprpcMemo{}, err
	}

	return memo, nil
}

func (k Keeper) validateIprpcMemo(ctx sdk.Context, memo types.IprpcMemo) error {
	if _, found := k.specKeeper.GetSpec(ctx, memo.Spec); !found {
		return printInvalidMemoWarning(memo, "memo's spec does not exist on chain")
	}

	if memo.Creator == "" {
		return printInvalidMemoWarning(memo, "memo's creator cannot be empty")
	} else if _, found := k.specKeeper.GetSpec(ctx, memo.Creator); found {
		return printInvalidMemoWarning(memo, "memo's creator cannot be an on-chain spec index")
	}

	if memo.Duration == uint64(0) {
		return printInvalidMemoWarning(memo, "memo's duration cannot be zero")
	}

	return nil
}

func printInvalidMemoWarning(memo types.IprpcMemo, description string) error {
	utils.LavaFormatWarning("invalid ibc over iprpc memo", fmt.Errorf(description),
		utils.LogAttr("memo", memo.String()),
	)
	return types.ErrIprpcMemoInvalid
}

// SendIbcIprpcReceiverTokensToPendingIprpcPool sends tokens from the IbcIprpcReceiver to the PendingIprpcPool as part of the IPRPC over IBC mechanism
// if the transfer fails, we try to transfer the tokens to the community pool
func (k Keeper) SendIbcIprpcReceiverTokensToPendingIprpcPool(ctx sdk.Context, amount sdk.Coin) error {
	// sanity check: IbcIprpcReceiver has enough funds to send
	ibcIprpcReceiverBalances := k.bankKeeper.GetAllBalances(ctx, types.IbcIprpcReceiverAddress())
	if ibcIprpcReceiverBalances.AmountOf(amount.Denom).LT(amount.Amount) {
		return utils.LavaFormatError("critical: IbcIprpcReceiver does not have enough funds to send to PendingIprpcPool", fmt.Errorf("send funds to PendingIprpcPool failed"),
			utils.LogAttr("IbcIprpcReceiver_balance", ibcIprpcReceiverBalances.AmountOf(amount.Denom)),
			utils.LogAttr("amount_to_send_to_PendingIprpcPool", amount.Amount),
		)
	}

	// transfer the token from the temp IbcIprpcReceiverAddress to the pending IPRPC fund request pool
	err1 := k.bankKeeper.SendCoinsFromAccountToModule(ctx, types.IbcIprpcReceiverAddress(), string(types.PendingIprpcPoolName), sdk.NewCoins(amount))
	if err1 != nil {
		// we couldn't transfer the funds to the pending IPRPC fund request pool, try moving it to the community pool
		err2 := k.distributionKeeper.FundCommunityPool(ctx, sdk.NewCoins(amount), types.IbcIprpcReceiverAddress())
		if err2 != nil {
			// community pool transfer failed, token kept locked in IbcIprpcReceiverAddress, return err ack
			return utils.LavaFormatError("could not send tokens from IbcIprpcReceiverAddress to pending IPRPC pool or community pool, tokens are locked in IbcIprpcReceiverAddress",
				errors.Join(err1, err2),
				utils.LogAttr("IbcIprpcReceiver_balance", ibcIprpcReceiverBalances.AmountOf(amount.Denom)),
				utils.LogAttr("amount_to_send_to_PendingIprpcPool", amount.Amount),
			)
		} else {
			return utils.LavaFormatError("could not send tokens from IbcIprpcReceiverAddress to pending IPRPC pool, sent to community pool", err1,
				utils.LogAttr("IbcIprpcReceiver_balance", ibcIprpcReceiverBalances.AmountOf(amount.Denom)),
				utils.LogAttr("amount_to_send_to_PendingIprpcPool", amount.Amount),
			)
		}
	}

	return nil
}
