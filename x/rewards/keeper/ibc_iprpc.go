package keeper

import (
	"errors"
	"fmt"
	"strconv"

	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	transfertypes "github.com/cosmos/ibc-go/v7/modules/apps/transfer/types"
	"github.com/lavanet/lava/utils"
	"github.com/lavanet/lava/utils/decoder"
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

// The PendingIbcIprpcFund object holds all the necessary information for a pending IPRPC over IBC fund request that its min
// IPRPC cost was not covered. The min cost can be covered using the cover-ibc-iprpc-costs TX. Then, the funds will be transferred
// to the IPRPC pool from the next month for the pending fund's duration field. The corresponding PendingIbcIprpcFund will be deleted.
// Also, every PendingIbcIprpcFund has an expiration on which the object is deleted. There will be no refund of the funds
// upon expiration. The expiration period is determined by the reward module's parameter IbcIprpcExpiration.

// NewPendingIbcIprpcFund sets a new PendingIbcIprpcFund object. It validates the input and sets the object with the right index and expiry
func (k Keeper) NewPendingIbcIprpcFund(ctx sdk.Context, creator string, spec string, duration uint64, fund sdk.Coin) (newPendingIbcIprpcFund types.PendingIbcIprpcFund, leftovers sdk.Coin, err error) {
	zeroCoin := sdk.NewCoin(fund.Denom, math.ZeroInt())
	// validate spec and funds
	_, found := k.specKeeper.GetSpec(ctx, spec)
	if !found {
		return types.PendingIbcIprpcFund{}, zeroCoin, utils.LavaFormatError("spec not found", fmt.Errorf("cannot create PendingIbcIprpcFund"),
			utils.LogAttr("creator", creator),
			utils.LogAttr("spec", spec),
			utils.LogAttr("duration", duration),
			utils.LogAttr("funds", fund),
		)
	}
	if fund.IsNil() || !fund.IsValid() {
		return types.PendingIbcIprpcFund{}, zeroCoin, utils.LavaFormatError("invalid funds", fmt.Errorf("cannot create PendingIbcIprpcFund"),
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
		return types.PendingIbcIprpcFund{}, zeroCoin, utils.LavaFormatWarning("fund amount cannot be less than duration", fmt.Errorf("cannot create PendingIbcIprpcFund"),
			utils.LogAttr("creator", creator),
			utils.LogAttr("spec", spec),
			utils.LogAttr("duration", duration),
			utils.LogAttr("funds", fund),
		)
	}

	// leftovers will be transferred to the community pool in the calling function
	leftovers = sdk.NewCoin(fund.Denom, fund.Amount.Sub(monthlyFund.Amount.MulRaw(int64(duration))))

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
		return types.PendingIbcIprpcFund{}, zeroCoin, utils.LavaFormatError("PendingIbcIprpcFund is invalid. expiry and duration must be positive, fund cannot be zero", fmt.Errorf("cannot create PendingIbcIprpcFund"),
			utils.LogAttr("creator", creator),
			utils.LogAttr("spec", spec),
			utils.LogAttr("duration", duration),
			utils.LogAttr("funds", fund),
			utils.LogAttr("expiry", expiry),
		)
	}

	k.SetPendingIbcIprpcFund(ctx, pendingIbcIprpcFund)

	return pendingIbcIprpcFund, leftovers, nil
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
			err := k.FundCommunityPoolFromModule(ctx, sdk.NewCoins(val.Fund), string(types.PendingIprpcPoolName))
			if err != nil {
				utils.LavaFormatError("failed funding community pool from expired IBC IPRPC fund, removing without funding", err,
					utils.LogAttr("pending_ibc_iprpc_fund", val.String()),
				)
			}
			k.RemovePendingIbcIprpcFund(ctx, val.Index)
			utils.LogLavaEvent(ctx, k.Logger(ctx), types.ExpiredPendingIbcIprpcFundRemovedEventName, map[string]string{
				"index":        strconv.FormatUint(val.Index, 10),
				"creator":      val.Creator,
				"spec":         val.Spec,
				"duration":     strconv.FormatUint(val.Duration, 10),
				"monthly_fund": val.Fund.String(),
				"expiry":       strconv.FormatUint(val.Expiry, 10),
			}, "Expired pending IBC IPRPC fund was removed. Funds were transferred to the community pool")
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

// CoverIbcIprpcFundCost covers the cost of a PendingIbcIprpcFund by sending the min cost funds to the IBC IPRPC receiver
// address and call FundIprpc(). Finally, it removes the PendingIbcIprpcFund object from the store
func (k Keeper) CoverIbcIprpcFundCost(ctx sdk.Context, creator string, index uint64) (costCovered sdk.Coin, err error) {
	zeroCoin := sdk.NewCoin(k.stakingKeeper.BondDenom(ctx), math.ZeroInt())
	creatorAddr, err := sdk.AccAddressFromBech32(creator)
	if err != nil {
		return zeroCoin, utils.LavaFormatWarning("invalid creator address. not Bech32", types.ErrCoverIbcIprpcFundCostFailed,
			utils.LogAttr("creator", creator),
			utils.LogAttr("index", index),
		)
	}

	// get the PendingIbcIprpcFund with index
	piif, found := k.GetPendingIbcIprpcFund(ctx, index)
	if !found {
		return zeroCoin, utils.LavaFormatWarning("PendingIbcIprpcFund not found", types.ErrCoverIbcIprpcFundCostFailed,
			utils.LogAttr("creator", creator),
			utils.LogAttr("index", index),
		)
	}

	// sanity check: PendingIbcIprpcFund is not expired
	if piif.IsExpired(ctx) {
		k.RemovePendingIbcIprpcFund(ctx, index)
		return zeroCoin, utils.LavaFormatWarning("PendingIbcIprpcFund with index is expired (deleted fund)", types.ErrCoverIbcIprpcFundCostFailed,
			utils.LogAttr("creator", creator),
			utils.LogAttr("index", index),
		)
	}

	// send the min cost to the ValidatorsRewardsAllocationPoolName (gov module doesn't pay min cost)
	cost := zeroCoin
	if creator != k.authority {
		cost = k.CalcPendingIbcIprpcFundMinCost(ctx, piif)
		err = k.bankKeeper.SendCoinsFromAccountToModule(ctx, creatorAddr, string(types.ValidatorsRewardsAllocationPoolName), sdk.NewCoins(cost))
		if err != nil {
			return zeroCoin, utils.LavaFormatWarning(types.ErrCoverIbcIprpcFundCostFailed.Error(), err,
				utils.LogAttr("creator", creator),
				utils.LogAttr("index", index),
			)
		}
	}

	// fund the iprpc pool from PendingIprpcPool (inside, the PendingIprpcPool and the gov module are not paying the min cost)
	err = k.FundIprpc(ctx, string(types.PendingIprpcPoolName), piif.Duration, sdk.NewCoins(piif.Fund), piif.Spec)
	if err != nil {
		return zeroCoin, utils.LavaFormatWarning(types.ErrCoverIbcIprpcFundCostFailed.Error(), err,
			utils.LogAttr("creator", creator),
			utils.LogAttr("index", index),
		)
	}

	// remove the PendingIbcIprpcFund
	k.RemovePendingIbcIprpcFund(ctx, index)

	return cost, nil
}

func (k Keeper) SendIbcTokensToPendingIprpcPool(ctx sdk.Context, amount sdk.Coin) error {
	return k.bankKeeper.SendCoinsFromAccountToModule(ctx, types.IbcIprpcReceiverAddress(), string(types.PendingIprpcPoolName), sdk.NewCoins(amount))
}

func (k Keeper) FundCommunityPoolFromIbcIprpcReceiver(ctx sdk.Context, amount sdk.Coin) error {
	return k.distributionKeeper.FundCommunityPool(ctx, sdk.NewCoins(amount), types.IbcIprpcReceiverAddress())
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
		err2 := k.FundCommunityPoolFromIbcIprpcReceiver(ctx, amount)
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
