package keeper

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	transfertypes "github.com/cosmos/ibc-go/v7/modules/apps/transfer/types"
	"github.com/lavanet/lava/utils"
	"github.com/lavanet/lava/utils/maps"
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

// The PendingIbcIprpcFund object holds all the necessary information for a pending IPRPC over IBC fund request that its min
// IPRPC cost was not covered. The min cost can be covered using the cover-ibc-iprpc-costs TX. Then, the funds will be transferred
// to the IPRPC pool from the next month for the pending fund's duration field. The corresponding PendingIbcIprpcFund will be deleted.
// Also, every PendingIbcIprpcFund has an expiration on which the object is deleted. There will be no refund of the funds
// upon expiration. The expiration period is determined by the reward module's parameter IbcIprpcExpiration.

// NewPendingIbcIprpcFund sets a new PendingIbcIprpcFund object. It validates the input and sets the object with the right index and expiry
func (k Keeper) NewPendingIbcIprpcFund(ctx sdk.Context, creator string, spec string, duration uint64, fund sdk.Coin) error {
	// validate spec and funds
	_, found := k.specKeeper.GetSpec(ctx, spec)
	if !found {
		return utils.LavaFormatError("spec not found", fmt.Errorf("cannot create PendingIbcIprpcFund"),
			utils.LogAttr("creator", creator),
			utils.LogAttr("spec", spec),
			utils.LogAttr("duration", duration),
			utils.LogAttr("funds", fund),
		)
	}
	if fund.IsNil() || !fund.IsValid() {
		return utils.LavaFormatError("invalid funds", fmt.Errorf("cannot create PendingIbcIprpcFund"),
			utils.LogAttr("creator", creator),
			utils.LogAttr("spec", spec),
			utils.LogAttr("duration", duration),
			utils.LogAttr("funds", fund),
		)
	}

	// divide funds by duration since we use addSpecFunds() when applying the PendingIbcIprpcFund
	// which assumes that each month will get the input fund
	monthlyFund := sdk.NewCoin(fund.Denom, fund.Amount.QuoRaw(int64(duration)))
	if monthlyFund.IsZero() {
		return utils.LavaFormatWarning("fund amount cannot be less than duration", fmt.Errorf("cannot create PendingIbcIprpcFund"),
			utils.LogAttr("creator", creator),
			utils.LogAttr("spec", spec),
			utils.LogAttr("duration", duration),
			utils.LogAttr("funds", fund),
		)
	}

	// leftovers will be transferred to the community pool
	leftovers := sdk.NewCoin(fund.Denom, fund.Amount.Sub(monthlyFund.Amount.MulRaw(int64(duration))))
	if !leftovers.IsZero() {
		receiverName, _ := types.IbcIprpcReceiverAddress()
		err := k.FundCommunityPoolFromModule(ctx, sdk.NewCoins(leftovers), receiverName)
		if err != nil {
			return utils.LavaFormatError("cannot transfer monthly fund leftovers to community pool for PendingIbcIprpcFund", err,
				utils.LogAttr("creator", creator),
				utils.LogAttr("spec", spec),
				utils.LogAttr("duration", duration),
				utils.LogAttr("funds", fund),
				utils.LogAttr("leftovers", leftovers),
			)
		}
	}

	// get index for the new object
	latestPendingIbcIprpcFund := k.GetLatestPendingIbcIprpcFund(ctx)
	newIndex := uint64(0)
	if !latestPendingIbcIprpcFund.IsEmpty() {
		newIndex = latestPendingIbcIprpcFund.Index + 1
	}

	// get expiry from current block time (using the rewards module parameter)
	expiry := k.CalcPendingIbcIprpcFundExpiration(ctx)

	pendingIbcIprpcFund := types.PendingIbcIprpcFund{
		Index:    newIndex,
		Creator:  creator,
		Spec:     spec,
		Duration: duration,
		Fund:     monthlyFund,
		Expiry:   expiry,
	}

	// sanity check
	if !pendingIbcIprpcFund.IsValid() {
		return utils.LavaFormatError("PendingIbcIprpcFund is invalid. expiry and duration must be positive, fund cannot be zero", fmt.Errorf("cannot create PendingIbcIprpcFund"),
			utils.LogAttr("creator", creator),
			utils.LogAttr("spec", spec),
			utils.LogAttr("duration", duration),
			utils.LogAttr("funds", fund),
			utils.LogAttr("expiry", expiry),
		)
	}

	return nil
}

// SetPendingIbcIprpcFund set an PendingIbcIprpcFund in the PendingIbcIprpcFund store
func (k Keeper) SetPendingIbcIprpcFund(ctx sdk.Context, pendingIbcIprpcFund types.PendingIbcIprpcFund) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.PendingIbcIprpcFundPrefix))
	b := k.cdc.MustMarshal(&pendingIbcIprpcFund)
	store.Set(maps.GetIDBytes(pendingIbcIprpcFund.Index), b)
}

// IsPendingIbcIprpcFund gets an PendingIbcIprpcFund from the PendingIbcIprpcFund store
func (k Keeper) GetPendingIbcIprpcFund(ctx sdk.Context, id uint64) (val types.PendingIbcIprpcFund, found bool) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.PendingIbcIprpcFundPrefix))
	b := store.Get(maps.GetIDBytes(id))
	if b == nil {
		return val, false
	}
	k.cdc.MustUnmarshal(b, &val)
	return val, true
}

// RemovePendingIbcIprpcFund removes an PendingIbcIprpcFund from the PendingIbcIprpcFund store
func (k Keeper) RemovePendingIbcIprpcFund(ctx sdk.Context, id uint64) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.PendingIbcIprpcFundPrefix))
	store.Delete(maps.GetIDBytes(id))
}

// GetAllPendingIbcIprpcFund returns all PendingIbcIprpcFund from the PendingIbcIprpcFund store
func (k Keeper) GetAllPendingIbcIprpcFund(ctx sdk.Context) (list []types.PendingIbcIprpcFund) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.PendingIbcIprpcFundPrefix))
	iterator := sdk.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.PendingIbcIprpcFund
		k.cdc.MustUnmarshal(iterator.Value(), &val)
		list = append(list, val)
	}

	return
}

// RemoveExpiredPendingIbcIprpcFunds removes all the expired PendingIbcIprpcFund objects from the PendingIbcIprpcFund store
func (k Keeper) RemoveExpiredPendingIbcIprpcFunds(ctx sdk.Context) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.PendingIbcIprpcFundPrefix))
	iterator := sdk.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.PendingIbcIprpcFund
		k.cdc.MustUnmarshal(iterator.Value(), &val)
		if val.IsExpired(ctx) {
			receiverName, _ := types.IbcIprpcReceiverAddress()
			err := k.FundCommunityPoolFromModule(ctx, sdk.NewCoins(val.Fund), receiverName)
			if err != nil {
				utils.LavaFormatError("failed funding community pool from expired IBC IPRPC fund, removing without funding", err,
					utils.LogAttr("pending_ibc_iprpc_fund", val.String()),
				)
			}
			k.RemovePendingIbcIprpcFund(ctx, val.Index)
		} else {
			break
		}
	}
}

// GetLatestPendingIbcIprpcFund gets the latest PendingIbcIprpcFund from the PendingIbcIprpcFund store
func (k Keeper) GetLatestPendingIbcIprpcFund(ctx sdk.Context) types.PendingIbcIprpcFund {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.PendingIbcIprpcFundPrefix))
	iterator := sdk.KVStoreReversePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.PendingIbcIprpcFund
		k.cdc.MustUnmarshal(iterator.Value(), &val)
		return val
	}

	return types.PendingIbcIprpcFund{}
}

// CalcPendingIbcIprpcFundMinCost calculates the required cost to apply it (transfer the funds to the IPRPC pool)
func (k Keeper) CalcPendingIbcIprpcFundMinCost(ctx sdk.Context, pendingIbcIprpcFund types.PendingIbcIprpcFund) sdk.Coin {
	minCost := k.GetMinIprpcCost(ctx)
	minCost.Amount = minCost.Amount.MulRaw(int64(pendingIbcIprpcFund.Duration))
	return minCost
}

// CalcPendingIbcIprpcFundExpiration returns the expiration timestamp of a PendingIbcIprpcFund
func (k Keeper) CalcPendingIbcIprpcFundExpiration(ctx sdk.Context) uint64 {
	return uint64(ctx.BlockTime().Add(k.IbcIprpcExpiration(ctx)).UTC().Unix())
}
