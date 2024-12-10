package keeper

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"cosmossdk.io/math"
	abci "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/v4/testutil/sample"
	"github.com/lavanet/lava/v4/utils"
	dualstakingtypes "github.com/lavanet/lava/v4/x/dualstaking/types"
	rewardstypes "github.com/lavanet/lava/v4/x/rewards/types"
	"github.com/lavanet/lava/v4/x/subscription/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) EstimatedProviderRewards(goCtx context.Context, req *types.QueryEstimatedProviderRewardsRequest) (*types.QueryEstimatedRewardsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	oldCtx := sdk.UnwrapSDKContext(goCtx)
	ctx, _ := oldCtx.CacheContext() // verify the original ctx is not changed

	res := types.QueryEstimatedRewardsResponse{}

	details := []utils.Attribute{
		utils.LogAttr("block", ctx.BlockHeight()),
		utils.LogAttr("provider", req.Provider),
	}

	// parse the delegator/delegation optional argument (delegate if needed)
	delegator := req.AmountDelegator
	trackedCuFactor := math.LegacyZeroDec()
	if req.AmountDelegator != "" {
		details = append(details, utils.LogAttr("delegator_amount", req.AmountDelegator))
		delegation, err := sdk.ParseCoinNormalized(req.AmountDelegator)
		if err != nil {
			// arg is not delegation, check if it's an address
			_, err := sdk.AccAddressFromBech32(req.AmountDelegator)
			if err != nil {
				return nil, utils.LavaFormatWarning("cannot estimate rewards for delegator/delegation", fmt.Errorf("delegator/delegation argument is neither a valid coin or a valid bech32 address"), details...)
			}
		} else {
			var err error
			// arg is delegation, assign arbitrary delegator address and delegate the amount
			delegator, err = k.createDummyDelegator(ctx, req.Provider, delegation)
			if err != nil {
				return nil, utils.LavaFormatWarning("cannot estimate rewards for delegator/delegation, cannot get delegator", err, details...)
			}

			totalDelegations := math.ZeroInt()
			md, err := k.epochstorageKeeper.GetMetadata(ctx, req.Provider)
			if err != nil {
				return nil, utils.LavaFormatWarning("cannot estimate rewards for delegator/delegation, cannot get provider metadata", err, details...)
			}
			for _, chain := range md.Chains {
				entry, found := k.epochstorageKeeper.GetStakeEntryCurrent(ctx, chain, req.Provider)
				if found {
					totalDelegations = totalDelegations.Add(entry.TotalStake())
				}
			}

			// calculate tracked CU factor which will increase the tracked CU due to
			// a higher total stake (because of the dummy delegation, which will affect pairing)
			trackedCuFactor = delegation.Amount.ToLegacyDec().QuoInt(totalDelegations)

			// advance ctx by a month and a day to make the delegation count
			ctx = ctx.WithBlockTime(ctx.BlockTime().AddDate(0, 1, 1))
		}
	}

	// get claimable rewards before the rewards distribution
	before, err := k.getClaimableRewards(ctx, req.Provider, delegator)
	if err != nil {
		return nil, utils.LavaFormatWarning("cannot estimate rewards, cannot get claimable rewards before distribution", err, details...)
	}

	// we use events to get the detailed info about the rewards (for the provider only use-case)
	// we reset the context's event manager to have a "clean slate"
	ctx = ctx.WithEventManager(sdk.NewEventManager())

	// trigger subs
	subs := k.GetAllSubscriptionsIndices(ctx)
	for _, sub := range subs {
		if !trackedCuFactor.IsZero() {
			subObj, found := k.GetSubscription(ctx, sub)
			if !found {
				continue
			}
			trackedCuInfos := k.GetSubTrackedCuInfoForProvider(ctx, sub, req.Provider, subObj.Block)
			for _, info := range trackedCuInfos {
				cuToAdd := trackedCuFactor.MulInt64(int64(info.TrackedCu)).TruncateInt64()
				err := k.AddTrackedCu(ctx, sub, req.Provider, info.ChainID, uint64(cuToAdd), subObj.Block)
				if err != nil {
					continue
				}
			}
		}

		k.advanceMonth(ctx, []byte(sub))
	}

	// get all CU tracker timers (used to keep data on subscription rewards) and distribute all subscription rewards
	gs := k.ExportCuTrackerTimers(ctx)
	for _, timer := range gs.BlockEntries {
		k.RewardAndResetCuTracker(ctx, []byte(timer.Key), timer.Data)
	}

	// distribute bonus and IPRPC rewards
	k.rewardsKeeper.DistributeMonthlyBonusRewards(ctx)

	// get claimable rewards after the rewards distribution
	after, err := k.getClaimableRewards(ctx, req.Provider, delegator)
	if err != nil {
		return nil, utils.LavaFormatWarning("cannot estimate rewards, cannot get claimable rewards after distribution", err, details...)
	}

	// calculate the claimable rewards difference
	res.Total = sdk.NewDecCoinsFromCoins(after.Sub(before...)...)

	// get detailed info for providers only
	if req.AmountDelegator == "" {
		info, total := k.getRewardsInfoFromEvents(ctx, req.Provider)
		if !total.IsEqual(res.Total) {
			// total amount sanity check
			return nil, utils.LavaFormatError("cannot estimate rewards, info sanity check failed",
				fmt.Errorf("total rewards from info is different than total claimable rewards difference"),
				utils.LogAttr("total_claimable_rewards", res.Total.String()),
				utils.LogAttr("total_info_rewards", total.String()),
				utils.LogAttr("info", info),
			)
		}

		res.Info = info
	}

	// get the last IPRPC rewards distribution block
	// note: on error we simply leave RecommendedBlock to be zero since until the
	// upcoming IPRPC rewards will be distributed (from the time this query is merged in main)
	// there will be no LastRewardsBlock set
	rewardsDistributionBlock, after24HoursBlock, err := k.rewardsKeeper.GetLastRewardsBlock(ctx)
	if err == nil {
		// if the query was sent within the first 24 hours of the month,
		// make the recommended block be the rewards distribution block - 1
		if uint64(ctx.BlockHeight()) <= after24HoursBlock {
			res.RecommendedBlock = rewardsDistributionBlock - 1
		}
	}

	return &res, nil
}

// helper function that returns a map of provider->rewards
func (k Keeper) getClaimableRewards(ctx sdk.Context, provider string, delegator string) (rewards sdk.Coins, err error) {
	var qRes *dualstakingtypes.QueryDelegatorRewardsResponse
	goCtx := sdk.WrapSDKContext(ctx)

	// get delegator rewards
	if delegator == "" {
		// self delegation
		md, err := k.epochstorageKeeper.GetMetadata(ctx, provider)
		if err != nil {
			return nil, err
		}
		qRes, err = k.dualstakingKeeper.DelegatorRewards(goCtx, &dualstakingtypes.QueryDelegatorRewardsRequest{
			Delegator: md.Vault,
			Provider:  md.Provider,
		})
		if err != nil {
			return nil, err
		}
	} else {
		qRes, err = k.dualstakingKeeper.DelegatorRewards(goCtx, &dualstakingtypes.QueryDelegatorRewardsRequest{
			Delegator: delegator,
			Provider:  provider,
		})
		if err != nil {
			return nil, err
		}
	}

	if len(qRes.Rewards) == 0 {
		return sdk.NewCoins(), nil
	}

	// we have one reward from the specific provider
	return qRes.Rewards[0].Amount, nil
}

// getRewardsInfoFromEvents build the estimated reward info array by checking events details
// it also returns the total reward amount for validation
func (k Keeper) getRewardsInfoFromEvents(ctx sdk.Context, provider string) (infos []types.EstimatedRewardInfo, total sdk.DecCoins) {
	events := ctx.EventManager().ABCIEvents()
	rewardsInfo := map[string]sdk.DecCoins{}

	for _, event := range events {
		// get reward info from event (!ok == event not relevant)
		eventRewardsInfo, ok := k.parseEvent(event, provider)
		if !ok {
			continue
		}

		// append the event map to rewardsInfo
		for source, reward := range eventRewardsInfo {
			savedReward, ok := rewardsInfo[source]
			if ok {
				rewardsInfo[source] = savedReward.Add(reward...)
			} else {
				rewardsInfo[source] = reward
			}
		}
	}

	// sort the rewardsInfo keys
	sortedKeys := []string{}
	for key := range rewardsInfo {
		sortedKeys = append(sortedKeys, key)
	}
	sort.Strings(sortedKeys)

	// construct the info list and calculate the total rewards
	for _, key := range sortedKeys {
		infos = append(infos, types.EstimatedRewardInfo{
			Source: key,
			Amount: rewardsInfo[key],
		})
		total = total.Add(rewardsInfo[key]...)
	}

	return infos, total
}

// parseEvent parses an event. If it's one of the payment events and contains the provider address
// it builds the "source" string that is required for the estimated rewards info. Also it returns the
// rewards amount of the provider for a specific payment type and a specific chain
// if it returns ok == false, the event is not relevant and can be skipped
func (k Keeper) parseEvent(event abci.Event, provider string) (eventRewardsInfo map[string]sdk.DecCoins, ok bool) {
	subEventName := utils.EventPrefix + types.SubscriptionPayoutEventName
	boostEventName := utils.EventPrefix + rewardstypes.ProvidersBonusRewardsEventName
	iprpcEventName := utils.EventPrefix + rewardstypes.IprpcPoolEmissionEventName

	// check event type and call appropriate handler
	switch event.Type {
	case subEventName:
		return extractInfoFromSubscriptionEvent(event, provider)
	case boostEventName:
		return extractInfoFromBoostEvent(event, provider)
	case iprpcEventName:
		return extractInfoFromIprpcEvent(event, provider)
	}

	return nil, false
}

// extractInfoFromSubscriptionEvent extract a map of source string keys (source = "Subscription: <chain_id>")
// that hold rewards amount. The subscription payout event holds multiple keys of each reward for each chain ID.
// All of the rewards keys are prefixed with provider addresses
func extractInfoFromSubscriptionEvent(event abci.Event, provider string) (eventRewardsInfo map[string]sdk.DecCoins, ok bool) {
	eventRewardsInfo = map[string]sdk.DecCoins{}

	for _, atr := range event.Attributes {
		if strings.HasPrefix(atr.Key, provider) {
			// extract chain ID
			parts := strings.Split(atr.Key, " ")
			if len(parts) != 2 {
				// should never happen (as long as the event key is not changed)
				utils.LavaFormatWarning("cannot extract chain ID from attribute",
					fmt.Errorf("attribute key is not in expected format of '<provider> <chain_id>'"),
					utils.LogAttr("attribute_key", atr.Key),
					utils.LogAttr("provider", provider),
				)
				return nil, false
			}
			source := "Subscription: " + parts[1]

			// extract provider reward
			parts = strings.Split(atr.Value, " ")
			if len(parts) != 4 {
				// should never happen (as long as the event value is not changed)
				utils.LavaFormatWarning("cannot extract provider reward from attribute",
					fmt.Errorf("attribute value is not in expected format of 'cu: <cu> reward: <reward_string>'"),
					utils.LogAttr("attribute_key", atr.Key),
					utils.LogAttr("attribute_value", atr.Value),
					utils.LogAttr("provider", provider),
				)
				return nil, false
			}
			reward, err := sdk.ParseDecCoins(parts[3])
			if err != nil {
				// should never happen
				utils.LavaFormatError("cannot parse DecCoins from reward string", err,
					utils.LogAttr("provider", provider),
					utils.LogAttr("chain_id", parts[1]),
					utils.LogAttr("amount", parts[3]),
				)
				return nil, false
			}

			// put source and reward in result map
			savedReward, ok := eventRewardsInfo[source]
			if ok {
				eventRewardsInfo[source] = savedReward.Add(reward...)
			} else {
				eventRewardsInfo[source] = reward
			}
		}
	}

	if len(eventRewardsInfo) == 0 {
		// the provider address was not found in the event
		return nil, false
	}

	return eventRewardsInfo, true
}

// extractInfoFromIprpcEvent extract a map of source string keys that hold rewards amount from
// a boost payout event
func extractInfoFromBoostEvent(event abci.Event, provider string) (eventRewardsInfo map[string]sdk.DecCoins, ok bool) {
	return extractInfoFromIprpcAndBoostEvent(event, provider, "Boost: ")
}

// extractInfoFromIprpcEvent extract a map of source string keys that hold rewards amount from
// an IPRPC payout event
func extractInfoFromIprpcEvent(event abci.Event, provider string) (eventRewardsInfo map[string]sdk.DecCoins, ok bool) {
	return extractInfoFromIprpcAndBoostEvent(event, provider, "Pools: ")
}

// extractInfoFromIprpcAndBoostEvent extract a map of source string keys (source = "Pools: <chain_id>")
// that hold rewards amount from either IPRPC and boost rewards. There's an event for each chain ID.
// All keys are prefixed with provider addresses. Since the events are emitted per chain ID,
// we expect a single reward key for a specific provider. Also, the reward string in the event
// is of type sdk.Coins.
func extractInfoFromIprpcAndBoostEvent(event abci.Event, provider string, sourcePrefix string) (eventRewardsInfo map[string]sdk.DecCoins, ok bool) {
	var rewardStr, chainID string
	eventRewardsInfo = map[string]sdk.DecCoins{}

	for _, atr := range event.Attributes {
		if strings.HasPrefix(atr.Key, provider) {
			// extract provider reward
			parts := strings.Split(atr.Value, " ")
			if len(parts) != 4 {
				// should never happen (as long as the event value is not changed)
				utils.LavaFormatWarning("cannot extract provider reward from attribute",
					fmt.Errorf("attribute value is not in expected format of 'cu: <cu> reward: <reward_string>'"),
					utils.LogAttr("attribute_key", atr.Key),
					utils.LogAttr("attribute_value", atr.Value),
					utils.LogAttr("provider", provider),
				)
				return nil, false
			}
			rewardStr = parts[3]
		}

		if atr.Key == "chainid" {
			chainID = atr.Value
		}
	}

	if rewardStr == "" || chainID == "" {
		// the provider address was not found in the event
		return nil, false
	}

	reward, err := sdk.ParseCoinsNormalized(rewardStr)
	if err != nil {
		// should never happen
		utils.LavaFormatError("cannot parse DecCoins from reward string", err,
			utils.LogAttr("provider", provider),
			utils.LogAttr("chain_id", chainID),
			utils.LogAttr("reward", rewardStr),
		)
		return nil, false
	}

	// put source and reward in result map
	eventRewardsInfo[sourcePrefix+chainID] = sdk.NewDecCoinsFromCoins(reward...)
	return eventRewardsInfo, true
}

// createDummyDelegator is a helper function to create a dummy delegator that delegates to the provider
func (k Keeper) createDummyDelegator(ctx sdk.Context, provider string, delegation sdk.Coin) (string, error) {
	delegator := sample.AccAddress()
	return delegator, k.dualstakingKeeper.Delegate(ctx, delegator, provider, delegation, false)
}

// Deprecated: backwards compatibility support for the old deprecated version of the EstimatedProviderRewards query
func (k Keeper) EstimatedRewards(goCtx context.Context, req *types.QueryEstimatedRewardsRequest) (*types.QueryEstimatedRewardsResponse, error) {
	newReq := types.QueryEstimatedProviderRewardsRequest{Provider: req.Provider, AmountDelegator: req.AmountDelegator}
	return k.EstimatedProviderRewards(goCtx, &newReq)
}
