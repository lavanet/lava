package keeper

import (
	"fmt"

	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/utils"
	"github.com/lavanet/lava/utils/maps"
	"github.com/lavanet/lava/x/rewards/types"
)

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
	expiry := k.CalcPendingIprpcExpiration(ctx)

	k.SetPendingIprpcFund(ctx, types.PendingIprpcFund{
		Index:       newIndex,
		Creator:     creator,
		Spec:        spec,
		Month:       month,
		Funds:       funds,
		Expiry:      expiry,
		CostCovered: sdk.NewCoin(k.stakingKeeper.BondDenom(ctx), math.ZeroInt()),
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

func (k Keeper) CalcPendingIprpcExpiration(ctx sdk.Context) uint64 {
	return uint64(ctx.BlockTime().Add(k.IbcIprpcExpiration(ctx)).UTC().Unix())
}
