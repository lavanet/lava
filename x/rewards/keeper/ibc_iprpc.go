package keeper

import (
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

func (k Keeper) SetPendingIprpcOverIbcFunds(ctx sdk.Context, memo types.IprpcMemo, amount sdk.Coin) error {
	// TODO: implement
	return nil
}
