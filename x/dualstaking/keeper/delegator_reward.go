package keeper

import (
	"strconv"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/v5/utils"
	"github.com/lavanet/lava/v5/x/dualstaking/types"
	spectypes "github.com/lavanet/lava/v5/x/spec/types"
)

// SetDelegatorReward set a specific DelegatorReward in the store from its index
func (k Keeper) SetDelegatorReward(ctx sdk.Context, delegatorReward types.DelegatorReward) {
	err := k.rewards.Set(ctx, types.DelegationKey(delegatorReward.Provider, delegatorReward.Delegator), delegatorReward)
	if err != nil {
		panic(err)
	}
}

// GetDelegatorReward returns a DelegatorReward from its index
func (k Keeper) GetDelegatorReward(ctx sdk.Context, provider, delegator string) (val types.DelegatorReward, found bool) {
	val, err := k.rewards.Get(ctx, types.DelegationKey(provider, delegator))
	return val, err == nil
}

// RemoveDelegatorReward removes a DelegatorReward from the store
func (k Keeper) RemoveDelegatorReward(
	ctx sdk.Context,
	provider, delegator string,
) {
	err := k.rewards.Remove(ctx, types.DelegationKey(provider, delegator))
	if err != nil {
		panic(err)
	}
}

// GetAllDelegatorReward returns all DelegatorReward
func (k Keeper) GetAllDelegatorReward(ctx sdk.Context) (list []types.DelegatorReward) {
	iter, err := k.rewards.Iterate(ctx, nil)
	if err != nil {
		panic(err)
	}

	list, err = iter.Values()
	if err != nil {
		panic(err)
	}

	return list
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
func (k Keeper) CalcRewards(ctx sdk.Context, totalReward sdk.Coins, totalDelegations math.Int, selfDelegation types.Delegation, commission uint64) (providerReward sdk.Coins, delegatorsReward sdk.Coins) {
	zeroCoins := sdk.NewCoins()
	totalDelegationsWithSelf := totalDelegations.Add(selfDelegation.Amount.Amount)

	// Sanity check - effectiveStake != 0
	if totalDelegationsWithSelf.IsZero() {
		return zeroCoins, zeroCoins
	}

	if commission == 100 {
		return totalReward, zeroCoins
	}

	providerReward = totalReward.MulInt(selfDelegation.Amount.Amount).QuoInt(totalDelegationsWithSelf)
	if !totalDelegations.IsZero() && commission != 0 {
		rawDelegatorsReward := totalReward.MulInt(totalDelegations).QuoInt(totalDelegationsWithSelf)
		providerCommission := rawDelegatorsReward.MulInt(sdk.NewIntFromUint64(commission)).QuoInt(sdk.NewInt(100))
		providerReward = providerReward.Add(providerCommission...)
	}

	return providerReward, totalReward.Sub(providerReward...)
}

// CalcDelegatorReward calculates a single delegator reward according to its delegation
// delegatorReward = delegatorsReward * (delegatorStake / totalDelegations) = (delegatorsReward * delegatorStake) / totalDelegations
func (k Keeper) CalcDelegatorReward(ctx sdk.Context, delegatorsReward sdk.Coins, totalDelegations math.Int, delegation types.Delegation) sdk.Coins {
	// Sanity check - totalDelegations != 0
	if totalDelegations.IsZero() {
		return sdk.NewCoins()
	}
	return delegatorsReward.MulInt(delegation.Amount.Amount).QuoInt(totalDelegations)
}

func (k Keeper) ClaimRewards(ctx sdk.Context, delegator string, provider string) (sdk.Coins, error) {
	goCtx := sdk.WrapSDKContext(ctx)

	delegatorAcc, err := sdk.AccAddressFromBech32(delegator)
	if err != nil {
		return sdk.Coins{}, utils.LavaFormatError("critical: could not claim delegator reward from provider", err,
			utils.Attribute{Key: "delegator", Value: delegator},
			utils.Attribute{Key: "provider", Value: provider},
		)
	}

	res, err := k.DelegatorRewards(goCtx, &types.QueryDelegatorRewardsRequest{Delegator: delegator, Provider: provider})
	if err != nil {
		return sdk.Coins{}, utils.LavaFormatWarning("could not claim delegator rewards", err,
			utils.Attribute{Key: "delegator", Value: delegator},
			utils.Attribute{Key: "provider", Value: provider},
		)
	}

	if len(res.Rewards) == 0 {
		return sdk.Coins{}, nil
	}

	rewardCoins := sdk.NewCoins()
	for _, reward := range res.Rewards {
		rewardCoins = rewardCoins.Add(reward.Amount...)
		k.RemoveDelegatorReward(ctx, reward.Provider, delegator)
	}

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

	return rewardCoins, nil
}

// RewardProvidersAndDelegators is the main function handling provider rewards with delegations
// it returns the provider reward amount and updates the delegatorReward map with the reward portion for each delegator
// since this function does not actually send rewards to the providers and delegator (but only allocates rewards to be claimed)
func (k Keeper) RewardProvidersAndDelegators(ctx sdk.Context, provider string, chainID string, totalReward sdk.Coins, senderModule string, calcOnlyProvider bool, calcOnlyDelegators bool, calcOnlyContributor bool) (providerReward sdk.Coins, err error) {
	block := uint64(ctx.BlockHeight())
	zeroCoins := sdk.NewCoins()

	metadata, err := k.epochstorageKeeper.GetMetadata(ctx, provider)
	if err != nil {
		return zeroCoins, utils.LavaFormatError(types.ErrCalculatingProviderReward.Error(), err,
			utils.Attribute{Key: "block", Value: block},
		)
	}

	delegations, err := k.GetProviderDelegators(ctx, provider)
	if err != nil {
		return zeroCoins, utils.LavaFormatError("cannot get provider's delegators", err)
	}
	// make sure this is post boost when rewards pool is introduced
	contributorAddresses, contributorPart := k.specKeeper.GetContributorReward(ctx, chainID)
	contributorsNum := sdk.NewInt(int64(len(contributorAddresses)))
	contributorReward := zeroCoins
	if !contributorsNum.IsZero() && contributorPart.GT(math.LegacyZeroDec()) {
		contributorReward = totalReward.MulInt(contributorPart.MulInt64(spectypes.ContributorPrecision).RoundInt()).QuoInt(sdk.NewInt(spectypes.ContributorPrecision))
		// make sure to round it down for the integers division
		contributorReward = contributorReward.QuoInt(contributorsNum).MulInt(contributorsNum)
		if !calcOnlyContributor {
			err = k.PayContributors(ctx, senderModule, contributorAddresses, contributorReward, chainID)
			if err != nil {
				return zeroCoins, err
			}
		}
	}

	relevantDelegations := []types.Delegation{}
	totalDelegations := sdk.ZeroInt()
	var selfDelegation types.Delegation
	// fetch relevant delegations (those who are passed the first week of delegation), self delegation and sum the total delegations
	for _, delegation := range delegations {
		if delegation.Delegator == metadata.Vault {
			selfDelegation = delegation
			// we are normalizing all delegations according to the time they were staked,
			// if the provider is staked less than a month that would handicap them so we need to adjust the provider stake as well
			credit := k.CalculateMonthlyCredit(ctx, selfDelegation)
			if credit.IsZero() {
				// should never happen
				continue
			}
			selfDelegation.Amount = credit
		} else {
			credit := k.CalculateMonthlyCredit(ctx, delegation)
			if credit.IsZero() {
				continue
			}
			// modify the delegation for reward calculation based on the time it was staked
			delegation.Amount = credit
			relevantDelegations = append(relevantDelegations, delegation)
			totalDelegations = totalDelegations.Add(delegation.Amount.Amount)
		}
	}

	providerReward, delegatorsReward := k.CalcRewards(ctx, totalReward.Sub(contributorReward...), totalDelegations, selfDelegation, metadata.DelegateCommission)

	leftoverRewards := k.updateDelegatorsReward(ctx, totalDelegations, relevantDelegations, delegatorsReward, senderModule, calcOnlyDelegators)
	fullProviderReward := providerReward.Add(leftoverRewards...)

	if !calcOnlyProvider {
		// reward provider's vault
		k.rewardDelegator(ctx, types.Delegation{Provider: metadata.Provider, Delegator: metadata.Vault}, fullProviderReward, senderModule)
	}

	return fullProviderReward, nil
}

// updateDelegatorsReward updates the delegator rewards map
func (k Keeper) updateDelegatorsReward(ctx sdk.Context, totalDelegations math.Int, delegations []types.Delegation, delegatorsReward sdk.Coins, senderModule string, calcOnly bool) (leftoverRewards sdk.Coins) {
	usedDelegatorRewards := sdk.NewCoins(sdk.NewCoin(k.stakingKeeper.BondDenom(ctx), math.ZeroInt())) // the delegator rewards are calculated using int division, so there might be leftovers

	for _, delegation := range delegations {
		delegatorReward := k.CalcDelegatorReward(ctx, delegatorsReward, totalDelegations, delegation)
		if !calcOnly {
			k.rewardDelegator(ctx, delegation, delegatorReward, senderModule)
		}
		usedDelegatorRewards = usedDelegatorRewards.Add(delegatorReward...)
	}

	return delegatorsReward.Sub(usedDelegatorRewards...)
}

func (k Keeper) rewardDelegator(ctx sdk.Context, delegation types.Delegation, amount sdk.Coins, senderModule string) {
	if amount.IsZero() {
		return
	}

	delegatorReward, found := k.GetDelegatorReward(ctx, delegation.Provider, delegation.Delegator)
	if !found {
		delegatorReward.Provider = delegation.Provider
		delegatorReward.Delegator = delegation.Delegator
		delegatorReward.Amount = amount
	} else {
		delegatorReward.Amount = delegatorReward.Amount.Add(amount...)
	}
	k.SetDelegatorReward(ctx, delegatorReward)
	err := k.bankKeeper.SendCoinsFromModuleToModule(ctx, senderModule, types.ModuleName, amount)
	if err != nil {
		utils.LavaFormatError("failed to send rewards to module", err, utils.LogAttr("sender", senderModule), utils.LogAttr("amount", amount.String()))
	}
}

func (k Keeper) PayContributors(ctx sdk.Context, senderModule string, contributorAddresses []sdk.AccAddress, contributorReward sdk.Coins, specId string) error {
	if len(contributorAddresses) == 0 {
		// do not return this error since we don;t want to bail
		utils.LavaFormatError("contributor addresses for pay are empty", nil)
		return nil
	}
	rewardCoins := contributorReward.QuoInt(sdk.NewInt(int64(len(contributorAddresses))))
	details := map[string]string{
		"total_reward_coins":           contributorReward.String(),
		"reward_coins_per_contributor": rewardCoins.String(),
		"chain_id":                     specId,
	}
	leftRewards := contributorReward
	for i, contributorAddress := range contributorAddresses {
		details["address."+strconv.Itoa(i)] = contributorAddress.String()
		if !leftRewards.IsAnyGTE(rewardCoins) {
			return utils.LavaFormatError("trying to pay contributors more than their allowed amount", nil, utils.LogAttr("rewardCoins", rewardCoins.String()), utils.LogAttr("contributorReward", contributorReward.String()), utils.LogAttr("leftRewards", leftRewards.String()))
		}
		leftRewards = leftRewards.Sub(rewardCoins...)
		err := k.bankKeeper.SendCoinsFromModuleToAccount(ctx, senderModule, contributorAddress, rewardCoins)
		if err != nil {
			return err
		}
	}
	utils.LogLavaEventDebug(ctx, k.Logger(ctx), types.ContributorRewardEventName, details, "All contributors rewarded successfully")
	if !leftRewards.IsZero() {
		utils.LavaFormatError("leftover rewards", nil, utils.LogAttr("rewardCoins", rewardCoins.String()), utils.LogAttr("contributorReward", contributorReward.String()), utils.LogAttr("leftRewards", leftRewards.String()))
		// we don;t want to bail on this
		return nil
	}
	return nil
}
