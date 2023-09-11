package keeper

import (
	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/x/dualstaking/types"
	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
)

// SetDelegatorReward set a specific DelegatorReward in the store from its index
func (k Keeper) SetDelegatorReward(ctx sdk.Context, delegatorReward types.DelegatorReward) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.DelegatorRewardKeyPrefix))
	b := k.cdc.MustMarshal(&delegatorReward)
	store.Set(types.DelegatorRewardKey(
		delegatorReward.Index,
	), b)
}

// GetDelegatorReward returns a DelegatorReward from its index
func (k Keeper) GetDelegatorReward(
	ctx sdk.Context,
	index string,
) (val types.DelegatorReward, found bool) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.DelegatorRewardKeyPrefix))

	b := store.Get(types.DelegatorRewardKey(
		index,
	))
	if b == nil {
		return val, false
	}

	k.cdc.MustUnmarshal(b, &val)
	return val, true
}

// RemoveDelegatorReward removes a DelegatorReward from the store
func (k Keeper) RemoveDelegatorReward(
	ctx sdk.Context,
	index string,
) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.DelegatorRewardKeyPrefix))
	store.Delete(types.DelegatorRewardKey(
		index,
	))
}

// GetAllDelegatorReward returns all DelegatorReward
func (k Keeper) GetAllDelegatorReward(ctx sdk.Context) (list []types.DelegatorReward) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.DelegatorRewardKeyPrefix))
	iterator := sdk.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.DelegatorReward
		k.cdc.MustUnmarshal(iterator.Value(), &val)
		list = append(list, val)
	}

	return
}

// CalcProviderReward calculates the provider reward considering delegations
// providerReward = totalReward * ((totalDelegations*commission + providerStake) / delegationsSum)
func (k Keeper) CalcProviderReward(stakeEntry epochstoragetypes.StakeEntry, totalReward math.Int) math.Int {
	providerStake := stakeEntry.Stake.Amount
	delegationCommission := stakeEntry.DelegateCommission
	totalDelegations := stakeEntry.DelegateTotal.Amount
	delegationsSum := k.CalcDelegationsSum(stakeEntry)

	providerRewardPercentage := totalDelegations.MulRaw(int64(delegationCommission / 100)).Add(providerStake).Quo(delegationsSum)
	return providerRewardPercentage.Mul(totalReward)
}

// CalcDelegatorsReward calculates the total amount of rewards for all delegators
// delegatorsReward = totalReward - providerReward
func (k Keeper) CalcDelegatorsReward(stakeEntry epochstoragetypes.StakeEntry, totalReward math.Int) math.Int {
	return totalReward.Sub(k.CalcProviderReward(stakeEntry, totalReward))
}

// CalcDelegationsSum calculates the delegations' sum
// delegations sum = totalDelegations + providerStake
func (k Keeper) CalcDelegationsSum(stakeEntry epochstoragetypes.StakeEntry) math.Int {
	totalDelegations := stakeEntry.DelegateTotal.Amount
	return totalDelegations.Add(stakeEntry.Stake.Amount)
}

// CalcDelegatorReward calculates a single delegator reward according to its delegation
// delegatorReward = delegatorsReward * (delegatorStake / totalDelegations) = (delegatorsReward * delegatorStake) / totalDelegations
func (k Keeper) CalcDelegatorReward(delegatorsReward math.Int, totalDelegations math.Int, delegation types.Delegation) math.Int {
	return delegatorsReward.Mul(delegation.Amount.Amount).Quo(totalDelegations)
}
