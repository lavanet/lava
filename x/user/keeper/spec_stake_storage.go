package keeper

import (
	"fmt"

	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/x/user/types"
)

// SetSpecStakeStorage set a specific specStakeStorage in the store from its index
func (k Keeper) SetSpecStakeStorage(ctx sdk.Context, specStakeStorage types.SpecStakeStorage) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.SpecStakeStorageKeyPrefix))
	b := k.cdc.MustMarshal(&specStakeStorage)
	store.Set(types.SpecStakeStorageKey(
		specStakeStorage.Index,
	), b)
}

// GetSpecStakeStorage returns a specStakeStorage from its index
func (k Keeper) GetSpecStakeStorage(
	ctx sdk.Context,
	index string,

) (val types.SpecStakeStorage, found bool) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.SpecStakeStorageKeyPrefix))

	b := store.Get(types.SpecStakeStorageKey(
		index,
	))
	if b == nil {
		return val, false
	}

	k.cdc.MustUnmarshal(b, &val)
	return val, true
}

// RemoveSpecStakeStorage removes a specStakeStorage from the store
func (k Keeper) RemoveSpecStakeStorage(
	ctx sdk.Context,
	index string,

) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.SpecStakeStorageKeyPrefix))
	store.Delete(types.SpecStakeStorageKey(
		index,
	))
}

// GetAllSpecStakeStorage returns all specStakeStorage
func (k Keeper) GetAllSpecStakeStorage(ctx sdk.Context) (list []types.SpecStakeStorage) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.SpecStakeStorageKeyPrefix))
	iterator := sdk.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.SpecStakeStorage
		k.cdc.MustUnmarshal(iterator.Value(), &val)
		list = append(list, val)
	}

	return
}

func (k Keeper) UnstakeUser(ctx sdk.Context, specName types.SpecName, unstakingUser string, deadline types.BlockNum) error {
	specStakeStorage, found := k.GetSpecStakeStorage(ctx, specName.Name)
	if !found {
		// the spec storage is empty
		return fmt.Errorf("can't unstake empty specStakeStorage for spec name: %s", specName.Name)
	}
	stakeStorage := specStakeStorage.StakeStorage
	found_staked_entry := false
	//TODO: improve the finding logic and the way Staked is saved looping a list is slow and bad
	for idx, stakedUser := range stakeStorage.StakedUsers {
		if stakedUser.Index == unstakingUser {
			// found entry
			found_staked_entry = true
			holdBlocks := k.UnstakeHoldBlocks(ctx)
			blockHeight := uint64(ctx.BlockHeight())

			if deadline.Num < blockHeight+holdBlocks {
				// unstaking demands they wait until a cedrftain block height so we can catch frauds before they escape with the money
				stakedUser.Deadline.Num = blockHeight + holdBlocks
			} else {
				stakedUser.Deadline.Num = deadline.Num
			}

			//TODO: store this list sorted by deadline so when we go over it in the timeout, we can do this efficiently
			unstakingUserAllSpecs := types.UnstakingUsersAllSpecs{
				Id:               0,
				Unstaking:        stakedUser,
				SpecStakeStorage: specStakeStorage,
			}
			k.AppendUnstakingUsersAllSpecs(ctx, unstakingUserAllSpecs)
			currentDeadline, found := k.GetBlockDeadlineForCallback(ctx)
			if !found {
				panic("didn't find single variable BlockDeadlineForCallback")
			}
			if currentDeadline.Deadline.Num == 0 || currentDeadline.Deadline.Num > stakedUser.Deadline.Num {
				currentDeadline.Deadline.Num = stakedUser.Deadline.Num
				k.SetBlockDeadlineForCallback(ctx, currentDeadline)
			}
			// effeciently delete stakedUser from stakeStorage.Staked
			stakeStorage.StakedUsers[idx] = stakeStorage.StakedUsers[len(stakeStorage.StakedUsers)-1] // replace the element at delete index with the last one
			stakeStorage.StakedUsers = stakeStorage.StakedUsers[:len(stakeStorage.StakedUsers)-1]     // remove last element
			//should be unique so there's no reason to keep iterating
			break
		}
	}
	if !found_staked_entry {
		return fmt.Errorf("can't unstake User, stake entry not found for address: %s", unstakingUser)
	}
	k.SetSpecStakeStorage(ctx, specStakeStorage)
	return nil
}
