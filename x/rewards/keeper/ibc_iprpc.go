package keeper

import (
	"encoding/json"
	"fmt"

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
		durationFloat64, ok := duration.(float64)
		if !ok {
			return printInvalidMemoWarning(iprpcData, "memo's duration field is not uint64")
		}
		if durationFloat64 <= 0 {
			return printInvalidMemoWarning(iprpcData, "memo's duration field cannot be non-positive")
		}
		memo.Duration = uint64(durationFloat64)
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

// The PendingIprpcFund object holds all the necessary information for a pending IPRPC over IBC fund request that its min
// IPRPC cost was not covered. Each PendingIprpcFund is created for a specific month and spec. The min IPRPC cost covers
// all the requests for a specific month. If the costs are covered, the corresponding PendingIprpcFund is deleted.
// Also, every PendingIprpcFund has an expiration on which the object is deleted. There will be no refund of the funds
// upon expiration. The expiration period is determined by the reward module's parameter.

// PendingIprpcFund methods

// NewPendingIprpcFund sets a new PendingIprpcFund object. It validates the input and sets the object with the right index and expiry
func (k Keeper) NewPendingIprpcFund(ctx sdk.Context, creator string, spec string, month uint64, funds sdk.Coins) error {
	// validate spec and funds
	_, found := k.specKeeper.GetSpec(ctx, spec)
	if !found {
		return utils.LavaFormatError("spec not found", fmt.Errorf("cannot create PendingIprpcFund"),
			utils.LogAttr("creator", creator),
			utils.LogAttr("spec", spec),
			utils.LogAttr("month", month),
			utils.LogAttr("funds", funds),
		)
	}
	if !funds.IsValid() {
		return utils.LavaFormatError("invalid funds", fmt.Errorf("cannot create PendingIprpcFund"),
			utils.LogAttr("creator", creator),
			utils.LogAttr("spec", spec),
			utils.LogAttr("month", month),
			utils.LogAttr("funds", funds),
		)
	}

	// get index for the new object
	latestPendingIprpcFund := k.GetLatestPendingIprpcFund(ctx)
	newIndex := uint64(0)
	if !latestPendingIprpcFund.IsEmpty() {
		newIndex = latestPendingIprpcFund.Index + 1
	}

	// get expiry from current block time (using the rewards module parameter)
	expiry := uint64(ctx.BlockTime().Add(k.IbcIprpcExpiration(ctx)).UTC().Unix())

	k.SetPendingIprpcFund(ctx, types.PendingIprpcFund{
		Index:   newIndex,
		Creator: creator,
		Spec:    spec,
		Month:   month,
		Funds:   funds,
		Expiry:  expiry,
	})

	return nil
}

// SetPendingIprpcFund set an PendingIprpcFund in the PendingIprpcFund store
func (k Keeper) SetPendingIprpcFund(ctx sdk.Context, PendingIprpcFund types.PendingIprpcFund) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.PendingIprpcFundPrefix))
	b := k.cdc.MustMarshal(&PendingIprpcFund)
	store.Set(maps.GetIDBytes(PendingIprpcFund.Index), b)
}

// IsPendingIprpcFund gets an PendingIprpcFund from the PendingIprpcFund store
func (k Keeper) GetPendingIprpcFund(ctx sdk.Context, id uint64) (val types.PendingIprpcFund, found bool) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.PendingIprpcFundPrefix))
	b := store.Get(maps.GetIDBytes(id))
	if b == nil {
		return val, false
	}
	k.cdc.MustUnmarshal(b, &val)
	return val, true
}

// RemovePendingIprpcFund removes an PendingIprpcFund from the PendingIprpcFund store
func (k Keeper) RemovePendingIprpcFund(ctx sdk.Context, id uint64) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.PendingIprpcFundPrefix))
	store.Delete(maps.GetIDBytes(id))
}

// GetAllPendingIprpcFund returns all PendingIprpcFund from the PendingIprpcFund store
func (k Keeper) GetAllPendingIprpcFund(ctx sdk.Context) (list []types.PendingIprpcFund) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.PendingIprpcFundPrefix))
	iterator := sdk.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.PendingIprpcFund
		k.cdc.MustUnmarshal(iterator.Value(), &val)
		list = append(list, val)
	}

	return
}

// RemoveExpiredPendingIprpcFund removes all the expired PendingIprpcFund objects from the PendingIprpcFund store
func (k Keeper) RemoveExpiredPendingIprpcFund(ctx sdk.Context) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.PendingIprpcFundPrefix))
	iterator := sdk.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.PendingIprpcFund
		k.cdc.MustUnmarshal(iterator.Value(), &val)
		if val.Expiry <= uint64(ctx.BlockTime().UTC().Unix()) {
			k.RemovePendingIprpcFund(ctx, val.Index)
		} else {
			break
		}
	}
}

// GetLatestPendingIprpcFund gets the latest PendingIprpcFund from the PendingIprpcFund store
func (k Keeper) GetLatestPendingIprpcFund(ctx sdk.Context) types.PendingIprpcFund {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.PendingIprpcFundPrefix))
	iterator := sdk.KVStoreReversePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.PendingIprpcFund
		k.cdc.MustUnmarshal(iterator.Value(), &val)
		return val
	}

	return types.PendingIprpcFund{}
}

func (k Keeper) IsIprpcMinCostCovered(ctx sdk.Context, pendingIprpcFund types.PendingIprpcFund) bool {
	return pendingIprpcFund.CostCovered.IsGTE(k.GetMinIprpcCost(ctx))
}
