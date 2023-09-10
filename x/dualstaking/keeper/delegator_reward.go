package keeper

import (
	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/utils"
	"github.com/lavanet/lava/x/dualstaking/types"
	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
	pairingtypes "github.com/lavanet/lava/x/pairing/types"
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
// providerReward = totalReward * ((effectiveDelegations*commission + providerStake) / effective stake)
func (k Keeper) CalcProviderReward(stakeEntry epochstoragetypes.StakeEntry, totalReward math.Int) math.Int {
	providerStake := stakeEntry.Stake.Amount
	delegationCommission := stakeEntry.DelegateCommission
	effectiveDelegations, effectiveStake := k.CalcEffectiveDelegationsAndStake(stakeEntry)

	providerRewardPercentage := effectiveDelegations.MulRaw(int64(delegationCommission / 100)).Add(providerStake).Quo(effectiveStake)
	return providerRewardPercentage.Mul(totalReward)
}

// CalcDelegatorsReward calculates the total amount of rewards for all delegators
// delegatorsReward = totalReward - providerReward
func (k Keeper) CalcDelegatorsReward(stakeEntry epochstoragetypes.StakeEntry, totalReward math.Int) math.Int {
	return totalReward.Sub(k.CalcProviderReward(stakeEntry, totalReward))
}

// CalcEffectiveDelegationsAndStake calculates the effective stake and effective delegations (for delegator rewards calculations)
// effectiveDelegations = totalDelegations
// effective stake = effectiveDelegations + providerStake
func (k Keeper) CalcEffectiveDelegationsAndStake(stakeEntry epochstoragetypes.StakeEntry) (effectiveDelegations math.Int, effectiveStake math.Int) {
	totalDelegations := stakeEntry.DelegateTotal.Amount
	return totalDelegations, totalDelegations.Add(stakeEntry.Stake.Amount)
}

// CalcDelegatorReward calculates a single delegator reward according to its delegation
// delegatorReward = (delegatorsReward * delegatorStake) / totalDelegations
func (k Keeper) CalcDelegatorReward(stakeEntry epochstoragetypes.StakeEntry, totalReward math.Int, delegation types.Delegation) math.Int {
	totalDelegations := stakeEntry.DelegateTotal.Amount

	delegatorsReward := k.CalcDelegatorsReward(stakeEntry, totalReward)
	return delegatorsReward.Mul(delegation.Amount.Amount).Quo(totalDelegations)
}

func (k Keeper) ClaimRewards(ctx sdk.Context, delegator string, provider string) error {
	goCtx := sdk.WrapSDKContext(ctx)
	res, err := k.DelegatorRewards(goCtx, &types.QueryDelegatorRewardsRequest{Delegator: delegator, Provider: provider})
	if err != nil {
		return utils.LavaFormatWarning("could not claim delegator rewards", err,
			utils.Attribute{Key: "delegator", Value: delegator},
		)
	}

	for _, reward := range res.Rewards {
		delegatorAcc, err := sdk.AccAddressFromBech32(delegator)
		if err != nil {
			utils.LavaFormatError("could not claim delegator reward from provider", err,
				utils.Attribute{Key: "delegator", Value: delegator},
				utils.Attribute{Key: "provider", Value: provider},
			)
			continue
		}

		rewardCoins := sdk.Coins{sdk.Coin{Denom: epochstoragetypes.TokenDenom, Amount: reward.Amount.Amount}}

		// not minting new coins because they're minted when the provider
		// asked for payment (and the delegator reward map was updated)
		err = k.bankKeeper.SendCoinsFromModuleToAccount(ctx, pairingtypes.ModuleName, delegatorAcc, rewardCoins)
		if err != nil {
			// panic:ok: reward transfer should never fail
			utils.LavaFormatPanic("critical: failed to send reward to delegator for provider", err,
				utils.Attribute{Key: "provider", Value: provider},
				utils.Attribute{Key: "delegator", Value: delegator},
				utils.Attribute{Key: "reward", Value: rewardCoins},
			)
		}

		ind := types.DelegationKey(delegator, reward.Provider, reward.ChainId)
		k.RemoveDelegatorReward(ctx, ind)
	}

	return nil
}
