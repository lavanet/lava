package keeper

import (
	"strconv"

	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/utils"
	"github.com/lavanet/lava/utils/slices"
	"github.com/lavanet/lava/x/dualstaking/types"
	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
	spectypes "github.com/lavanet/lava/x/spec/types"
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
func (k Keeper) CalcRewards(stakeEntry epochstoragetypes.StakeEntry, totalReward math.Int, delegations []types.Delegation) (providerReward math.Int, delegatorsReward math.Int) {
	effectiveDelegations, effectiveStake := k.CalcEffectiveDelegationsAndStake(stakeEntry, delegations)

	providerReward = totalReward.Mul(stakeEntry.Stake.Amount).Quo(effectiveStake)
	rawDelegatorsReward := totalReward.Mul(effectiveDelegations).Quo(effectiveStake)
	providerCommission := rawDelegatorsReward.MulRaw(int64(stakeEntry.DelegateCommission)).QuoRaw(100)
	providerReward = providerReward.Add(providerCommission)

	return providerReward, totalReward.Sub(providerReward)
}

// CalcEffectiveDelegationsAndStake calculates the effective stake and effective delegations (for delegator rewards calculations)
// effectiveDelegations = min(totalDelegations, delegateLimit)
// effectiveStake = effectiveDelegations + providerStake
func (k Keeper) CalcEffectiveDelegationsAndStake(stakeEntry epochstoragetypes.StakeEntry, delegations []types.Delegation) (effectiveDelegations math.Int, effectiveStake math.Int) {
	var totalDelegations int64
	for _, d := range delegations {
		totalDelegations += d.Amount.Amount.Int64()
	}

	effectiveDelegationsInt64 := math.Min(totalDelegations, stakeEntry.DelegateLimit.Amount.Int64())
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

		rewardCoins := sdk.Coins{sdk.Coin{Denom: k.stakingKeeper.BondDenom(ctx), Amount: reward.Amount.Amount}}

		// not minting new coins because they're minted when the provider
		// asked for payment (and the delegator reward map was updated)
		err = k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, delegatorAcc, rewardCoins)
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

// RewardProvidersAndDelegators is the main function handling provider rewards with delegations
// it returns the provider reward amount and updates the delegatorReward map with the reward portion for each delegator
func (k Keeper) RewardProvidersAndDelegators(ctx sdk.Context, providerAddr sdk.AccAddress, chainID string, totalReward math.Int, senderModule string, calcOnlyProvider bool, calcOnlyDelegators bool, calcOnlyContributer bool) (providerReward math.Int, claimableRewards math.Int, err error) {
	block := uint64(ctx.BlockHeight())
	epoch, _, err := k.epochstorageKeeper.GetEpochStartForBlock(ctx, block)
	if err != nil {
		return math.ZeroInt(), math.ZeroInt(), utils.LavaFormatError(types.ErrCalculatingProviderReward.Error(), err,
			utils.Attribute{Key: "block", Value: block},
		)
	}
	stakeEntry, err := k.epochstorageKeeper.GetStakeEntryForProviderEpoch(ctx, chainID, providerAddr, epoch)
	if err != nil {
		return math.ZeroInt(), math.ZeroInt(), err
	}

	delegations, err := k.GetProviderDelegators(ctx, providerAddr.String(), epoch)
	if err != nil {
		return math.ZeroInt(), math.ZeroInt(), utils.LavaFormatError("cannot get provider's delegators", err)
	}

	// make sure this is post boost when rewards pool is introduced
	contributorAddresses, contributorPart := k.specKeeper.GetContributorReward(ctx, chainID)
	if contributorPart.GT(math.LegacyZeroDec()) {
		contributorsNum := int64(len(contributorAddresses))
		contributorReward := totalReward.MulRaw(contributorPart.MulInt64(spectypes.ContributorPrecision).RoundInt64()).QuoRaw(spectypes.ContributorPrecision)
		// make sure to round it down for the integers division
		contributorReward = contributorReward.QuoRaw(contributorsNum).MulRaw(contributorsNum)
		claimableRewards = totalReward.Sub(contributorReward)
		if !calcOnlyContributer {
			err = k.PayContributors(ctx, senderModule, contributorAddresses, contributorReward, chainID)
			if err != nil {
				return math.ZeroInt(), math.ZeroInt(), err
			}
		}
	}

	relevantDelegations := slices.Filter(delegations,
		func(d types.Delegation) bool {
			return d.ChainID == chainID && d.IsFirstMonthPassed(ctx.BlockTime().UTC().Unix()) && d.Delegator != d.Provider
		})

	providerReward, delegatorsReward := k.CalcRewards(*stakeEntry, claimableRewards, relevantDelegations)

	leftoverRewards := k.updateDelegatorsReward(ctx, stakeEntry.DelegateTotal.Amount, relevantDelegations, totalReward, delegatorsReward, senderModule, calcOnlyDelegators)
	fullProviderReward := providerReward.Add(leftoverRewards)

	if !calcOnlyProvider {
		if fullProviderReward.GT(math.ZeroInt()) {
			k.rewardDelegator(ctx, types.Delegation{Provider: providerAddr.String(), ChainID: chainID, Delegator: providerAddr.String()}, fullProviderReward, senderModule)
		}
	}

	return fullProviderReward, claimableRewards, nil
}

// updateDelegatorsReward updates the delegator rewards map
func (k Keeper) updateDelegatorsReward(ctx sdk.Context, totalDelegations math.Int, delegations []types.Delegation, totalReward math.Int, delegatorsReward math.Int, senderModule string, calcOnly bool) (leftoverRewards math.Int) {
	usedDelegatorRewards := math.ZeroInt() // the delegator rewards are calculated using int division, so there might be leftovers

	for _, delegation := range delegations {
		delegatorRewardAmount := k.CalcDelegatorReward(delegatorsReward, totalDelegations, delegation)

		if !calcOnly {
			k.rewardDelegator(ctx, delegation, delegatorRewardAmount, senderModule)
		}

		usedDelegatorRewards = usedDelegatorRewards.Add(delegatorRewardAmount)
	}

	return delegatorsReward.Sub(usedDelegatorRewards)
}

func (k Keeper) rewardDelegator(ctx sdk.Context, delegation types.Delegation, amount math.Int, senderModule string) {
	if amount.IsZero() {
		return
	}

	rewardMapKey := types.DelegationKey(delegation.Provider, delegation.Delegator, delegation.ChainID)
	delegatorReward, found := k.GetDelegatorReward(ctx, rewardMapKey)
	if !found {
		delegatorReward.Provider = delegation.Provider
		delegatorReward.Delegator = delegation.Delegator
		delegatorReward.ChainId = delegation.ChainID
		delegatorReward.Amount = sdk.NewCoin(k.stakingKeeper.BondDenom(ctx), amount)
	} else {
		delegatorReward.Amount = delegatorReward.Amount.AddAmount(amount)
	}
	k.SetDelegatorReward(ctx, delegatorReward)
	err := k.bankKeeper.SendCoinsFromModuleToModule(ctx, senderModule, types.ModuleName, sdk.NewCoins(sdk.NewCoin(k.stakingKeeper.BondDenom(ctx), amount)))
	if err != nil {
		utils.LavaFormatError("failed to send rewards to module", err, utils.LogAttr("sender", senderModule), utils.LogAttr("amount", amount.String()))
	}
}

func (k Keeper) PayContributors(ctx sdk.Context, senderModule string, contributorAddresses []sdk.AccAddress, contributorReward math.Int, specId string) error {
	if len(contributorAddresses) == 0 {
		// do not return this error since we don;t want to bail
		utils.LavaFormatError("contributor addresses for pay are empty", nil)
		return nil
	}
	rewardPerContributor := contributorReward.QuoRaw(int64(len(contributorAddresses)))
	rewardCoins := sdk.Coins{sdk.NewCoin(k.stakingKeeper.BondDenom(ctx), rewardPerContributor)}
	details := map[string]string{
		"rewardCoins": rewardCoins.String(),
		"specId":      specId,
	}
	leftRewards := contributorReward
	for i, contributorAddress := range contributorAddresses {
		details["address."+strconv.Itoa(i)] = contributorAddress.String()
		if leftRewards.LT(rewardCoins.AmountOf(k.stakingKeeper.BondDenom(ctx))) {
			return utils.LavaFormatError("trying to pay contributors more than their allowed amount", nil, utils.LogAttr("rewardCoins", rewardCoins.String()), utils.LogAttr("contributorReward", contributorReward.String()), utils.LogAttr("leftRewards", leftRewards.String()))
		}
		leftRewards = leftRewards.Sub(rewardCoins.AmountOf(k.stakingKeeper.BondDenom(ctx)))
		err := k.bankKeeper.SendCoinsFromModuleToAccount(ctx, senderModule, contributorAddress, rewardCoins)
		if err != nil {
			return err
		}
	}
	utils.LogLavaEvent(ctx, k.Logger(ctx), types.ContributorRewardEventName, details, "contributors rewards given")
	if leftRewards.GT(math.ZeroInt()) {
		utils.LavaFormatError("leftover rewards", nil, utils.LogAttr("rewardCoins", rewardCoins.String()), utils.LogAttr("contributorReward", contributorReward.String()), utils.LogAttr("leftRewards", leftRewards.String()))
		// we don;t want to bail on this
		return nil
	}
	return nil
}
