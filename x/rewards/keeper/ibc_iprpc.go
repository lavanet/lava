package keeper

import (
	"encoding/json"
	"fmt"
	"strconv"

	sdk "github.com/cosmos/cosmos-sdk/types"
	transfertypes "github.com/cosmos/ibc-go/v7/modules/apps/transfer/types"
	"github.com/lavanet/lava/utils"
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
funds are saved in the PendingIprpcFund scaffolded map.

*/

// ExtractIprpcMemoFromPacket extracts the memo field from an ibc-transfer packet and verifies that it's in the right format
// and holds valid values. If the memo is not in the right format, a custom error is returned so the packet will be skipped and
// passed to the next IBC module in the transfer stack normally (and not return an error ack)
func (k Keeper) ExtractIprpcMemoFromPacket(ctx sdk.Context, transferData transfertypes.FungibleTokenPacketData) (types.IprpcMemo, error) {
	memo := types.IprpcMemo{}
	transferMemo := make(map[string]interface{})
	err := json.Unmarshal([]byte(transferData.Memo), &transferMemo)
	if err != nil || transferMemo["iprpc"] == nil {
		// memo is not for IPRPC over IBC, return custom error to skip processing for this packet
		return types.IprpcMemo{}, types.ErrMemoNotIprpcOverIbc
	}

	if iprpcData, ok := transferMemo["iprpc"].(map[string]interface{}); ok {
		// verify creator field
		creator, ok := iprpcData["creator"]
		if !ok {
			return printInvalidMemoWarning(iprpcData, "memo data does not contain creator field")
		}
		creatorStr, ok := creator.(string)
		if !ok {
			return printInvalidMemoWarning(iprpcData, "memo's creator field is not string")
		}
		if creatorStr == "" {
			return printInvalidMemoWarning(iprpcData, "memo's creator field cannot be empty")
		}
		memo.Creator = creatorStr

		// verify spec field
		spec, ok := iprpcData["spec"]
		if !ok {
			return printInvalidMemoWarning(iprpcData, "memo data does not contain spec field")
		}
		specStr, ok := spec.(string)
		if !ok {
			return printInvalidMemoWarning(iprpcData, "memo's spec field is not string")
		}
		_, found := k.specKeeper.GetSpec(ctx, specStr)
		if !found {
			return printInvalidMemoWarning(iprpcData, "memo's spec field does not exist on chain")
		}
		memo.Spec = specStr

		// verify duration field
		duration, ok := iprpcData["duration"]
		if !ok {
			return printInvalidMemoWarning(iprpcData, "memo data does not contain duration field")
		}
		durationStr, ok := duration.(string)
		if !ok {
			return printInvalidMemoWarning(iprpcData, "memo's duration field is not a string number")
		}
		durationUint64, err := strconv.ParseUint(durationStr, 10, 64)
		if err != nil {
			return printInvalidMemoWarning(iprpcData, "memo's duration field cannot be non-positive. err: "+err.Error())
		}
		memo.Duration = durationUint64
	}

	return memo, nil
}

func printInvalidMemoWarning(iprpcData map[string]interface{}, description string) (types.IprpcMemo, error) {
	utils.LavaFormatWarning("invalid ibc over iprpc memo", fmt.Errorf(description),
		utils.LogAttr("data", iprpcData),
	)
	return types.IprpcMemo{}, types.ErrIprpcMemoInvalid
}

func (k Keeper) SetPendingIprpcOverIbcFunds(ctx sdk.Context, memo types.IprpcMemo, amount sdk.Coin) error {
	// TODO: implement
	return nil
}
