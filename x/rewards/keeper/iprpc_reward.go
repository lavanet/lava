package keeper

import (
	"encoding/binary"

	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/x/rewards/types"
)

// GetIprpcRewardsCurrent get the total number of IprpcReward
func (k Keeper) GetIprpcRewardsCurrent(ctx sdk.Context) uint64 {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), []byte{})
	byteKey := types.KeyPrefix(types.IprpcRewardsCurrentPrefix)
	bz := store.Get(byteKey)

	// Current doesn't exist: no element
	if bz == nil {
		return 0
	}

	// Parse bytes
	return binary.BigEndian.Uint64(bz)
}

// SetIprpcRewardsCurrent set the total number of IprpcReward
func (k Keeper) SetIprpcRewardsCurrent(ctx sdk.Context, current uint64) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), []byte{})
	byteKey := types.KeyPrefix(types.IprpcRewardsCurrentPrefix)
	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, current)
	store.Set(byteKey, bz)
}

// SetIprpcReward set a specific IprpcReward in the store
func (k Keeper) SetIprpcReward(ctx sdk.Context, iprpcReward types.IprpcReward) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.IprpcRewardPrefix))
	b := k.cdc.MustMarshal(&iprpcReward)
	store.Set(GetIprpcRewardIDBytes(iprpcReward.Id), b)
}

// GetIprpcReward returns a IprpcReward from its id
func (k Keeper) GetIprpcReward(ctx sdk.Context, id uint64) (val types.IprpcReward, found bool) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.IprpcRewardPrefix))
	b := store.Get(GetIprpcRewardIDBytes(id))
	if b == nil {
		return val, false
	}
	k.cdc.MustUnmarshal(b, &val)
	return val, true
}

// RemoveIprpcReward removes a IprpcReward from the store
func (k Keeper) RemoveIprpcReward(ctx sdk.Context, id uint64) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.IprpcRewardPrefix))
	store.Delete(GetIprpcRewardIDBytes(id))
}

// GetAllIprpcReward returns all IprpcReward
func (k Keeper) GetAllIprpcReward(ctx sdk.Context) (list []types.IprpcReward) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.IprpcRewardPrefix))
	iterator := sdk.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.IprpcReward
		k.cdc.MustUnmarshal(iterator.Value(), &val)
		list = append(list, val)
	}

	return
}

// GetIprpcRewardIDBytes returns the byte representation of the ID
func GetIprpcRewardIDBytes(id uint64) []byte {
	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, id)
	return bz
}

// GetIprpcRewardIDFromBytes returns ID in uint64 format from a byte array
func GetIprpcRewardIDFromBytes(bz []byte) uint64 {
	return binary.BigEndian.Uint64(bz)
}

// PopIprpcReward gets the lowest id IprpcReward object and removes it
func (k Keeper) PopIprpcReward(ctx sdk.Context) (types.IprpcReward, bool) {
	current := k.GetIprpcRewardsCurrent(ctx)
	k.SetIprpcRewardsCurrent(ctx, current+1)
	return k.GetIprpcReward(ctx, current)
}

// AddSpecFunds adds funds for a specific spec for <duration> of months.
// This function is used by the fund-iprpc TX.
func (k Keeper) addSpecFunds(ctx sdk.Context, spec string, fund sdk.Coins, duration uint64, fromNextMonth bool) {
	startID := k.GetIprpcRewardsCurrent(ctx)
	if fromNextMonth {
		startID += 1
	}

	for i := startID; i < duration; i++ {
		iprpcReward, found := k.GetIprpcReward(ctx, i)
		if found {
			// found IPRPC reward, find if spec exists
			specIndex := -1
			for i := 0; i < len(iprpcReward.SpecFunds); i++ {
				if iprpcReward.SpecFunds[i].Spec == spec {
					specIndex = i
				}
			}
			// update spec funds
			if specIndex >= 0 {
				iprpcReward.SpecFunds[specIndex].Fund = iprpcReward.SpecFunds[specIndex].Fund.Add(fund...)
			} else {
				iprpcReward.SpecFunds = append(iprpcReward.SpecFunds, types.Specfund{Spec: spec, Fund: fund})
			}
		} else {
			// did not find IPRPC reward -> create a new one
			iprpcReward.Id = i
			iprpcReward.SpecFunds = []types.Specfund{{Spec: spec, Fund: fund}}
		}
		k.SetIprpcReward(ctx, iprpcReward)
	}
}
