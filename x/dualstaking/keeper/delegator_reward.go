package keeper

import (
	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/utils"
	"github.com/lavanet/lava/utils/slices"
	"github.com/lavanet/lava/x/dualstaking/types"
	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
	pairingtypes "github.com/lavanet/lava/x/pairing/types"
)

// SetDelegatorReward set a specific DelegatorReward in the store from its index
func (k Keeper) SetDelegatorReward(ctx sdk.Context, delegatorReward types.DelegatorReward) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.DelegatorRewardKeyPrefix))
	index := types.DelegationKey(delegatorReward.Provider, delegatorReward.Delegator, delegatorReward.ChainId)
	b := k.cdc.MustMarshal(&delegatorReward)
	store.Set(types.DelegatorRewardKey(
		index,
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

// The reward for servicing a consumer is divided between the provider and its delegators.
// The following terms are used when calculating the reward distribution between them:
//   1. TotalReward: The total reward before it's divided to the provider and delegators
//   2. ProviderReward: The provider's part in the reward
//   3. DelegatorsReward: The total reward for all delegators
//   4. DelegatorReward: The reward for a specific delegator
//   5. TotalDelegations: The total sum of delegations of a specific provider
//   6. effectiveStake: The provider's stake + TotalDelegations ("effective stake")
//   7. DelegationLimit: The provider's delegation limit (e.g. the max value of delegations the provider uses)
//   8. effectiveDelegations: min(TotalDelegations, DelegationLimit)

// CalcRewards calculates the provider reward and the total reward for delegators
// providerReward = totalReward * ((effectiveDelegations*commission + providerStake) / effectiveStake)
// delegatorsReward = totalReward - providerReward
func (k Keeper) CalcRewards(stakeEntry epochstoragetypes.StakeEntry, totalReward math.Int) (providerReward math.Int, delegatorsReward math.Int) {
	effectiveDelegations, effectiveStake := k.CalcEffectiveDelegationsAndStake(stakeEntry)

	providerReward = totalReward.Mul(stakeEntry.Stake.Amount).Quo(effectiveStake)
	rawDelegatorsReward := totalReward.Mul(effectiveDelegations).Quo(effectiveStake)
	providerCommission := rawDelegatorsReward.MulRaw(int64(stakeEntry.DelegateCommission)).QuoRaw(100)
	providerReward = providerReward.Add(providerCommission)

	return providerReward, totalReward.Sub(providerReward)
}

// CalcEffectiveDelegationsAndStake calculates the effective stake and effective delegations (for delegator rewards calculations)
// effectiveDelegations = min(totalDelegations, delegateLimit)
// effectiveStake = effectiveDelegations + providerStake
func (k Keeper) CalcEffectiveDelegationsAndStake(stakeEntry epochstoragetypes.StakeEntry) (effectiveDelegations math.Int, effectiveStake math.Int) {
	effectiveDelegationsInt64 := math.Min(stakeEntry.DelegateTotal.Amount.Int64(), stakeEntry.DelegateLimit.Amount.Int64())
	effectiveDelegations = math.NewInt(effectiveDelegationsInt64)
	return effectiveDelegations, effectiveDelegations.Add(stakeEntry.Stake.Amount)
}

// CalcDelegatorReward calculates a single delegator reward according to its delegation
// delegatorReward = delegatorsReward * (delegatorStake / totalDelegations) = (delegatorsReward * delegatorStake) / totalDelegations
func (k Keeper) CalcDelegatorReward(delegatorsReward math.Int, totalDelegations math.Int, delegation types.Delegation) math.Int {
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
			utils.LavaFormatError("critical: could not claim delegator reward from provider", err,
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

		ind := types.DelegationKey(reward.Provider, delegator, reward.ChainId)
		k.RemoveDelegatorReward(ctx, ind)
	}

	return nil
}

// CalcProviderRewardWithDelegations is the main function handling provider rewards with delegations
// it returns the provider reward amount and updates the delegatorReward map with the reward portion for each delegator
func (k Keeper) CalcProviderRewardWithDelegations(ctx sdk.Context, providerAddr sdk.AccAddress, chainID string, block uint64, totalReward math.Int) (providerReward math.Int, err error) {
	epoch, _, err := k.epochstorageKeeper.GetEpochStartForBlock(ctx, block)
	if err != nil {
		return math.ZeroInt(), utils.LavaFormatError(types.ErrCalculatingProviderReward.Error(), err,
			utils.Attribute{Key: "block", Value: block},
		)
	}
	stakeEntry, err := k.epochstorageKeeper.GetStakeEntryForProviderEpoch(ctx, chainID, providerAddr, epoch)
	if err != nil {
		return math.ZeroInt(), utils.LavaFormatError(types.ErrCalculatingProviderReward.Error(), err,
			utils.Attribute{Key: "provider", Value: providerAddr},
			utils.Attribute{Key: "chainID", Value: chainID},
			utils.Attribute{Key: "epoch", Value: epoch},
			utils.Attribute{Key: "totalRewards", Value: totalReward},
		)
	}

	delegations, err := k.GetProviderDelegators(ctx, providerAddr.String(), epoch)
	if err != nil {
		return math.ZeroInt(), utils.LavaFormatError("cannot get provider's delegators", err)
	}

	relevantDelegations := slices.Filter(delegations,
		func(d types.Delegation) bool { return d.ChainID == chainID })

	providerReward, delegatorsReward := k.CalcRewards(*stakeEntry, totalReward)

	leftoverRewards := k.updateDelegatorsReward(ctx, stakeEntry.DelegateTotal.Amount, relevantDelegations, totalReward, delegatorsReward)

	return providerReward.Add(leftoverRewards), nil
}

// updateDelegatorsReward updates the delegator rewards map
func (k Keeper) updateDelegatorsReward(ctx sdk.Context, totalDelegations math.Int, delegations []types.Delegation, totalReward math.Int, delegatorsReward math.Int) (leftoverRewards math.Int) {
	usedDelegatorRewards := math.ZeroInt() // the delegator rewards are calculated using int division, so there might be leftovers

	for _, delegation := range delegations {
		delegatorRewardAmount := k.CalcDelegatorReward(delegatorsReward, totalDelegations, delegation)
		rewardMapKey := types.DelegationKey(delegation.Provider, delegation.Delegator, delegation.ChainID)

		delegatorReward, found := k.GetDelegatorReward(ctx, rewardMapKey)
		if !found {
			delegatorReward.Provider = delegation.Provider
			delegatorReward.Delegator = delegation.Delegator
			delegatorReward.ChainId = delegation.ChainID
			delegatorReward.Amount = sdk.NewCoin(epochstoragetypes.TokenDenom, delegatorRewardAmount)
		} else {
			delegatorReward.Amount = delegatorReward.Amount.AddAmount(delegatorRewardAmount)
		}
		k.SetDelegatorReward(ctx, delegatorReward)
		usedDelegatorRewards = usedDelegatorRewards.Add(delegatorRewardAmount)
	}

	return delegatorsReward.Sub(usedDelegatorRewards)
}
