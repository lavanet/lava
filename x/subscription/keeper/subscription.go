package keeper

import (
	"fmt"
	"strconv"
	"strings"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	legacyerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/lavanet/lava/v2/utils"
	"github.com/lavanet/lava/v2/utils/lavaslices"
	planstypes "github.com/lavanet/lava/v2/x/plans/types"
	projectstypes "github.com/lavanet/lava/v2/x/projects/types"
	"github.com/lavanet/lava/v2/x/subscription/types"
)

// GetSubscription returns the subscription of a given consumer
func (k Keeper) GetSubscription(ctx sdk.Context, consumer string) (val types.Subscription, found bool) {
	block := uint64(ctx.BlockHeight())

	var sub types.Subscription
	found = k.subsFS.FindEntry(ctx, consumer, block, &sub)

	return sub, found
}

// GetSubscription returns the subscription of a given consumer in a specific block
func (k Keeper) GetSubscriptionForBlock(ctx sdk.Context, consumer string, block uint64) (val types.Subscription, entryBlock uint64, found bool) {
	var sub types.Subscription
	entryBlock, _, _, found = k.subsFS.FindEntryDetailed(ctx, consumer, block, &sub)
	return sub, entryBlock, found
}

// CreateSubscription creates a subscription for a consumer
func (k Keeper) CreateSubscription(
	ctx sdk.Context,
	creator string,
	consumer string,
	planIndex string,
	duration uint64,
	autoRenewalFlag bool,
) error {
	var err error

	block := uint64(ctx.BlockHeight())
	creatorAcct, plan, err := k.verifySubscriptionBuyInputAndGetPlan(ctx, block, creator, consumer, planIndex)
	if err != nil {
		return err
	}

	var sub types.Subscription
	nextEpoch, err := k.epochstorageKeeper.GetNextEpoch(ctx, block)
	if err != nil {
		return utils.LavaFormatError("Got an error while trying to get next epoch on CreateSubscription", err,
			utils.LogAttr("consumer", sub.Consumer),
			utils.LogAttr("block", block),
		)
	}

	// When we make a subscription upgrade, we append the new subscription entry on the next epoch,
	// 	but, we want to get the most updated subscription here.
	// Hence, in case of user making double upgrade in the same epoch, we take the next epoch.
	found := k.subsFS.FindEntry(ctx, consumer, nextEpoch, &sub)

	// Subscription creation:
	//   When: if not already exists for consumer address)
	//   What: find plan, create default project, set duration, update credit,
	//         charge fees, save subscription.
	//
	// Subscription renewal:
	//   When: if already exists and existing plan is the same as current plans
	//         ("same" means same index and same block of creation)
	//   What: find plan, update duration (total and remaining), update credit,
	//         charge fees, save subscription.
	//
	// Subscription upgrade:
	//   When: if already exists and existing plan, and new plan's price is higher than current plan's
	//		   or if plan index is different from existing
	//   What: find subscription, verify new plan's price, upgrade, set duration, update credit,
	//         charge fees, save subscription.
	// Subscription downgrade: (TBD)

	if !found {
		sub, err = k.createNewSubscription(ctx, &plan, creator, consumer, block, autoRenewalFlag)
		if err != nil {
			return utils.LavaFormatWarning("failed to create subscription", err)
		}
	} else {
		// Allow renewal with the same plan ("same" means both plan index);
		// If the plan index is different - upgrade if the price is higher or equal
		// If the plan index is the same but the plan block is different - advice using the "--advance-purchase" flag
		if plan.Index != sub.PlanIndex {
			if sub.Creator != creator && sub.Consumer != creator {
				return utils.LavaFormatWarning("only the consumer or buyer can overwrite the subscription plan.", nil)
			}
			err := k.upgradeSubscriptionPlan(ctx, &sub, &plan)
			if err != nil {
				return err
			}
		} else if plan.Block != sub.PlanBlock {
			return utils.LavaFormatWarning("requested plan block has changed, can't extend. Please use the --advance-purchase with the same plan index.", nil)
		}

		// The total duration may not exceed MAX_SUBSCRIPTION_DURATION, but allow an
		// extra month to account for renewals before the end of current subscription
		if sub.DurationLeft+duration > types.MAX_SUBSCRIPTION_DURATION+1 {
			str := strconv.FormatInt(types.MAX_SUBSCRIPTION_DURATION, 10)
			return utils.LavaFormatWarning("duration would exceed limit ("+str+" months)",
				fmt.Errorf("subscription renewal failed"),
				utils.Attribute{Key: "duration", Value: sub.DurationLeft},
			)
		}
	}

	// update total (last requested) duration and remaining duration
	sub.DurationBought = duration
	sub.DurationLeft += duration

	if err := sub.ValidateSubscription(); err != nil {
		return utils.LavaFormatWarning("create subscription failed", err)
	}

	// subscription looks good; let's create it and charge the creator
	price := plan.GetPrice()
	price.Amount = price.Amount.MulRaw(int64(duration))
	k.applyPlanDiscountIfEligible(duration, &plan, &price)

	sub.Credit = sub.Credit.AddAmount(price.Amount)

	if !found {
		expiry := uint64(utils.NextMonth(ctx.BlockTime()).UTC().Unix())
		sub.MonthExpiryTime = expiry
		sub.Block = block
		err = k.subsFS.AppendEntry(ctx, consumer, block, &sub)
		if err != nil {
			return err
		}
		k.subsTS.AddTimerByBlockTime(ctx, expiry, []byte(consumer), []byte{})
	} else {
		k.subsFS.ModifyEntry(ctx, consumer, sub.Block, &sub)
	}

	err = k.chargeFromCreatorAccountToModule(ctx, creatorAcct, price)
	if err != nil {
		return err
	}

	return nil
}

func (k Keeper) verifySubscriptionBuyInputAndGetPlan(ctx sdk.Context, block uint64, creator, consumer, planIndex string) (creatorAcct sdk.AccAddress, plan planstypes.Plan, err error) {
	EMPTY_PLAN := planstypes.Plan{}

	if _, err := sdk.AccAddressFromBech32(consumer); err != nil {
		return nil, EMPTY_PLAN, utils.LavaFormatWarning("invalid subscription consumer address", err,
			utils.Attribute{Key: "consumer", Value: consumer},
		)
	}

	creatorAcct, err = sdk.AccAddressFromBech32(creator)
	if err != nil {
		return nil, EMPTY_PLAN, utils.LavaFormatWarning("invalid subscription creator address", err,
			utils.Attribute{Key: "creator", Value: creator},
		)
	}

	plan, found := k.plansKeeper.GetPlan(ctx, planIndex)
	if !found {
		return nil, EMPTY_PLAN, utils.LavaFormatWarning("cannot create subscription with invalid plan", nil,
			utils.Attribute{Key: "creator", Value: creator},
			utils.Attribute{Key: "consumer", Value: consumer},
			utils.Attribute{Key: "plan", Value: planIndex},
			utils.Attribute{Key: "block", Value: block},
		)
	}

	if len(plan.AllowedBuyers) != 0 {
		if !lavaslices.Contains(plan.AllowedBuyers, creator) {
			allowedBuyers := strings.Join(plan.AllowedBuyers, ",")
			return nil, EMPTY_PLAN, utils.LavaFormatWarning("subscription buy input is invalid", fmt.Errorf("creator is not part of the allowed buyers list"),
				utils.LogAttr("creator", creator),
				utils.LogAttr("plan", plan.Index),
				utils.LogAttr("allowed_buyers", allowedBuyers),
			)
		}
	}

	return creatorAcct, plan, nil
}

func (k Keeper) createNewSubscription(ctx sdk.Context, plan *planstypes.Plan, creator, consumer string,
	block uint64, autoRenewalFlag bool,
) (types.Subscription, error) {
	autoRenewalNextPlan := types.AUTO_RENEWAL_PLAN_NONE
	if autoRenewalFlag {
		// On subscription creation, auto renewal is set to the subscription's plan
		autoRenewalNextPlan = plan.Index
	}

	sub := types.Subscription{
		Creator:             creator,
		Consumer:            consumer,
		Block:               block,
		PlanIndex:           plan.Index,
		PlanBlock:           plan.Block,
		DurationTotal:       0,
		AutoRenewalNextPlan: autoRenewalNextPlan,
		Credit:              sdk.NewCoin(k.stakingKeeper.BondDenom(ctx), math.ZeroInt()),
	}

	sub.MonthCuTotal = plan.PlanPolicy.GetTotalCuLimit()
	sub.MonthCuLeft = plan.PlanPolicy.GetTotalCuLimit()
	sub.Cluster = types.GetClusterKey(sub)

	// new subscription needs a default project
	err := k.projectsKeeper.CreateAdminProject(ctx, consumer, *plan)
	if err != nil {
		return types.Subscription{}, utils.LavaFormatWarning("failed to create default project", err)
	}

	return sub, nil
}

func (k Keeper) upgradeSubscriptionPlan(ctx sdk.Context, sub *types.Subscription, newPlan *planstypes.Plan) error {
	block := uint64(ctx.BlockHeight())

	nextEpoch, err := k.epochstorageKeeper.GetNextEpoch(ctx, block)
	if err != nil {
		return utils.LavaFormatError("Trying to upgrade subscription, but got an error while trying to get next epoch", err,
			utils.LogAttr("consumer", sub.Consumer))
	}

	if sub.Block == nextEpoch {
		// The subscription block is the next epoch, meaning, the user already tried to upgrade the subscription in this epoch
		return utils.LavaFormatWarning("can't upgrade the same subscription more than once in the same epoch", fmt.Errorf("subscription block is equal to next epoch"),
			utils.LogAttr("consumer", sub.Consumer),
			utils.LogAttr("block", block),
			utils.LogAttr("nextEpoch", nextEpoch),
		)
	}

	// We want to get the most up-to-date subscription (including next-epoch upgrade)
	currentPlan, err := k.GetPlanFromSubscription(ctx, sub.Consumer, nextEpoch)
	if err != nil {
		return utils.LavaFormatError("failed to find plan for current subscription", err,
			utils.LogAttr("consumer", sub.Consumer),
			utils.LogAttr("block", block),
			utils.LogAttr("nextEpoch", nextEpoch),
		)
	}

	if newPlan.Price.Amount.LT(currentPlan.Price.Amount) {
		return utils.LavaFormatError("New plan's price must be higher than the old plan", nil,
			utils.LogAttr("consumer", sub.Consumer),
			utils.LogAttr("currentPlan", currentPlan),
			utils.LogAttr("newPlan", newPlan),
			utils.LogAttr("block", block),
			utils.LogAttr("nextEpoch", nextEpoch),
		)
	}

	// Track CU for the previous subscription
	k.addCuTrackerTimerForSubscription(ctx, block, sub)

	if sub.DurationLeft == 0 {
		// Subscription was already expired. Can't upgrade.
		k.handleZeroDurationLeftForSubscription(ctx, block, sub)
		return utils.LavaFormatError("Trying to upgrade subscription, but subscription was already expired", nil,
			utils.LogAttr("consumer", sub.Consumer))
	}

	sub.DurationTotal = 0

	// The "old" subscription's duration is now expired
	// If called from CreateSubscription, the duration will reset to the duration bought
	sub.DurationLeft = 0
	sub.PlanIndex = newPlan.Index
	sub.PlanBlock = newPlan.Block
	sub.MonthCuTotal = newPlan.PlanPolicy.TotalCuLimit
	sub.Credit.Amount = math.ZeroInt()

	err = k.resetSubscriptionDetailsAndAppendEntry(ctx, sub, nextEpoch, true)
	if err != nil {
		return utils.LavaFormatError("upgrade subscription failed, reset subscription failed", err,
			utils.Attribute{Key: "consumer", Value: sub.Consumer},
			utils.Attribute{Key: "block", Value: strconv.FormatUint(nextEpoch, 10)},
		)
	}

	details := map[string]string{
		"consumer": sub.Consumer,
		"oldPlan":  sub.PlanIndex,
		"newPlan":  newPlan.Index,
	}
	utils.LogLavaEvent(ctx, k.Logger(ctx), types.UpgradeSubscriptionEventName, details, "subscription upgraded")
	return nil
}

func (k Keeper) renewSubscription(ctx sdk.Context, sub *types.Subscription) error {
	block := ctx.BlockHeight()
	planIndex := sub.AutoRenewalNextPlan

	plan, found := k.plansKeeper.FindPlan(ctx, planIndex, uint64(block))
	if !found {
		return utils.LavaFormatError("failed to find existing subscription plan", legacyerrors.ErrKeyNotFound,
			utils.Attribute{Key: "consumer", Value: sub.Creator},
			utils.Attribute{Key: "planIndex", Value: sub.PlanIndex},
			utils.Attribute{Key: "block", Value: block},
		)
	}

	if len(plan.AllowedBuyers) != 0 {
		if !lavaslices.Contains(plan.AllowedBuyers, sub.Creator) {
			allowedBuyers := strings.Join(plan.AllowedBuyers, ",")
			return utils.LavaFormatWarning("cannot auto-renew subscription", fmt.Errorf("creator is not part of the allowed buyers list"),
				utils.LogAttr("creator", sub.Creator),
				utils.LogAttr("plan", plan.Index),
				utils.LogAttr("allowed_buyers", allowedBuyers),
			)
		}
	}

	sub.PlanIndex = plan.Index
	sub.PlanBlock = plan.Block
	sub.DurationBought += 1
	sub.DurationLeft = 1
	sub.Block = uint64(block)

	// Charge creator for 1 extra month
	price := plan.GetPrice()

	creatorAcct, err := sdk.AccAddressFromBech32(sub.Creator)
	if err != nil {
		return utils.LavaFormatWarning("invalid subscription consumer address", err,
			utils.Attribute{Key: "consumer", Value: sub.Creator},
		)
	}

	if k.bankKeeper.GetBalance(ctx, creatorAcct, k.stakingKeeper.BondDenom(ctx)).IsLT(price) {
		return utils.LavaFormatWarning("renew subscription failed", legacyerrors.ErrInsufficientFunds,
			utils.LogAttr("creator", sub.Creator),
			utils.LogAttr("price", price),
		)
	}

	err = k.bankKeeper.SendCoinsFromAccountToModule(ctx, creatorAcct, types.ModuleName, []sdk.Coin{price})
	if err != nil {
		return utils.LavaFormatError("renew subscription failed. funds transfer failed", err,
			utils.LogAttr("creator", sub.Creator),
			utils.LogAttr("price", price),
		)
	}

	sub.Credit = sub.Credit.AddAmount(price.Amount)

	err = k.resetSubscriptionDetailsAndAppendEntry(ctx, sub, sub.Block, false)
	if err != nil {
		return utils.LavaFormatWarning("renew subscription failed", err,
			utils.LogAttr("creator", sub.Creator),
			utils.LogAttr("price", price),
		)
	}

	if plan.Index != sub.PlanIndex || plan.Block != sub.PlanBlock {
		// Different plan: decrease refcount for old plan, increase for new plan
		k.plansKeeper.PutPlan(ctx, sub.PlanIndex, sub.PlanBlock)
		k.plansKeeper.GetPlan(ctx, plan.Index)
	}

	return nil
}

func (k Keeper) advanceMonth(ctx sdk.Context, subkey []byte) {
	block := k.epochstorageKeeper.GetCurrentNextEpoch(ctx)
	consumer := string(subkey)

	var sub types.Subscription
	if !k.verifySubExists(ctx, consumer, block, &sub) {
		return
	}

	k.addCuTrackerTimerForSubscription(ctx, block, &sub)

	if sub.DurationLeft == 0 {
		k.handleZeroDurationLeftForSubscription(ctx, block, &sub)
		return
	}

	sub.DurationLeft -= 1

	if sub.DurationLeft > 0 {
		sub.DurationTotal += 1
		err := k.resetSubscriptionDetailsAndAppendEntry(ctx, &sub, block, false)
		if err != nil {
			utils.LavaFormatError("failed subscription reset in advance month", err,
				utils.Attribute{Key: "consumer", Value: sub.Consumer},
				utils.Attribute{Key: "block", Value: strconv.FormatUint(block, 10)},
			)
			return
		}
	} else {
		if sub.FutureSubscription != nil {
			// Consumer made advance purchase. Now we activate it.
			newSubInfo := sub.FutureSubscription

			plan, found := k.plansKeeper.FindPlan(ctx, newSubInfo.PlanIndex, newSubInfo.PlanBlock)
			if !found {
				utils.LavaFormatWarning("subscription advance purchase failed: could not find plan. removing subscription", nil,
					utils.Attribute{Key: "consumer", Value: sub.Consumer},
					utils.Attribute{Key: "planIndex", Value: newSubInfo.PlanIndex},
				)
				k.RemoveExpiredSubscription(ctx, consumer, block, sub.PlanIndex, sub.PlanBlock)
				return
			}

			sub.Creator = newSubInfo.Creator
			sub.PlanIndex = newSubInfo.PlanIndex
			sub.PlanBlock = newSubInfo.PlanBlock
			sub.DurationBought = newSubInfo.DurationBought
			sub.DurationLeft = newSubInfo.DurationBought
			sub.DurationTotal = 0
			sub.FutureSubscription = nil
			sub.MonthCuTotal = plan.PlanPolicy.TotalCuLimit
			sub.Credit = newSubInfo.Credit

			err := k.resetSubscriptionDetailsAndAppendEntry(ctx, &sub, block, false)
			if err != nil {
				utils.LavaFormatError("failed subscription reset in advance month", err,
					utils.Attribute{Key: "consumer", Value: sub.Consumer},
					utils.Attribute{Key: "block", Value: strconv.FormatUint(block, 10)},
				)
				return
			}
		} else if sub.IsAutoRenewalOn() {
			// apply the DurationLeft decrease to 0 and buy an extra month
			k.subsFS.ModifyEntry(ctx, sub.Consumer, sub.Block, &sub)
			err := k.renewSubscription(ctx, &sub)
			if err != nil {
				utils.LavaFormatWarning("subscription auto renewal failed. removing subscription", err,
					utils.Attribute{Key: "consumer", Value: sub.Consumer},
				)
				k.RemoveExpiredSubscription(ctx, consumer, block, sub.PlanIndex, sub.PlanBlock)
			}
		} else {
			k.RemoveExpiredSubscription(ctx, consumer, block, sub.PlanIndex, sub.PlanBlock)
		}
	}
}

func (k Keeper) verifySubExists(ctx sdk.Context, consumer string, block uint64, sub *types.Subscription) bool {
	if found := k.subsFS.FindEntry(ctx, consumer, block, sub); !found {
		// subscription (monthly) timer has expired for an unknown subscription:
		// either the timer was set wrongly, or the subscription was incorrectly
		// removed; and we cannot even return an error about it.
		utils.LavaFormatError("critical: month expiry for unknown subscription, skipping",
			fmt.Errorf("subscription not found"),
			utils.Attribute{Key: "consumer", Value: consumer},
			utils.Attribute{Key: "block", Value: block},
		)
		return false
	}
	return true
}

func (k Keeper) addCuTrackerTimerForSubscription(ctx sdk.Context, block uint64, sub *types.Subscription) {
	blocksToSave, err := k.epochstorageKeeper.BlocksToSave(ctx, block)
	if err != nil {
		utils.LavaFormatError("critical: failed assigning CU tracker callback, skipping", err,
			utils.Attribute{Key: "block", Value: block},
		)
	} else {
		creditReward := sub.Credit.Amount.QuoRaw(int64(sub.DurationLeft))
		sub.Credit = sub.Credit.SubAmount(creditReward)

		timerData := types.CuTrackerTimerData{
			Block:  sub.Block,
			Credit: sdk.NewCoin(k.stakingKeeper.BondDenom(ctx), creditReward),
		}
		marshaledTimerData, err := k.cdc.Marshal(&timerData)
		if err != nil {
			utils.LavaFormatError("critical: failed assigning CU tracker callback. can't marshal cu tracker timer data, skipping", err,
				utils.Attribute{Key: "block", Value: block},
			)
			return
		}
		k.cuTrackerTS.AddTimerByBlockHeight(ctx, block+blocksToSave-1, []byte(sub.Consumer), marshaledTimerData)
	}
}

func (k Keeper) handleZeroDurationLeftForSubscription(ctx sdk.Context, block uint64, sub *types.Subscription) {
	// subscription duration has already reached zero before and should have
	// been removed before. Extend duration by another month (without adding
	// CUs) to allow smooth operation.
	utils.LavaFormatError("critical: negative duration for subscription, extend by 1 month",
		fmt.Errorf("negative duration in AdvanceMonth()"),
		utils.Attribute{Key: "consumer", Value: sub.Consumer},
		utils.Attribute{Key: "plan", Value: sub.PlanIndex},
		utils.Attribute{Key: "block", Value: block},
	)
	// normally would panic! but can "recover" by auto-extending by 1 month
	// (don't bother to modify sub.MonthExpiryTime to minimize state changes)
	expiry := uint64(utils.NextMonth(ctx.BlockTime()).UTC().Unix())
	k.subsTS.AddTimerByBlockTime(ctx, expiry, []byte(sub.Consumer), []byte{})
}

func (k Keeper) resetSubscriptionDetailsAndAppendEntry(ctx sdk.Context, sub *types.Subscription, block uint64, deleteOldTimer bool) error {
	// reset subscription CU allowance for this coming month
	sub.MonthCuLeft = sub.MonthCuTotal
	sub.Block = block

	// restart timer and append new (fixated) version of this subscription
	// the timer will expire in exactly one month from now
	expiry := uint64(utils.NextMonth(ctx.BlockTime()).UTC().Unix())

	tsKey := []byte(sub.Consumer)

	if deleteOldTimer {
		if k.subsTS.HasTimerByBlockTime(ctx, sub.MonthExpiryTime, tsKey) {
			k.subsTS.DelTimerByBlockTime(ctx, sub.MonthExpiryTime, tsKey)
		} else {
			utils.LavaFormatError("Delete timer failed: timer key was not found", nil,
				utils.LogAttr("expiryTime", sub.MonthExpiryTime),
				utils.LogAttr("consumer", sub.Consumer),
				utils.LogAttr("block", block),
			)
		}
	}

	sub.MonthExpiryTime = expiry

	// since the total duration increases, the cluster might change
	sub.Cluster = types.GetClusterKey(*sub)
	k.subsTS.AddTimerByBlockTime(ctx, expiry, tsKey, []byte{})

	err := k.subsFS.AppendEntry(ctx, sub.Consumer, block, sub)
	if err != nil {
		// normally would panic! but ok to ignore - the subscription remains
		// as is with same remaining duration (but not renewed CU)
		return utils.LavaFormatError("critical: failed to recharge subscription", err,
			utils.Attribute{Key: "consumer", Value: sub.Consumer},
			utils.Attribute{Key: "plan", Value: sub.PlanIndex},
			utils.Attribute{Key: "block", Value: block},
		)
	}

	// reset projects CU allowance for this coming month
	k.projectsKeeper.SnapshotSubscriptionProjects(ctx, sub.Consumer, block)

	return nil
}

func (k Keeper) applyPlanDiscountIfEligible(duration uint64, plan *planstypes.Plan, price *sdk.Coin) {
	if duration >= utils.MONTHS_IN_YEAR {
		// adjust cost if discount given
		discount := plan.GetAnnualDiscountPercentage()
		if discount > 0 {
			factor := int64(100 - discount)
			price.Amount = price.Amount.MulRaw(factor).QuoRaw(100)
		}
	}
}

func (k Keeper) chargeFromCreatorAccountToModule(ctx sdk.Context, creator sdk.AccAddress, price sdk.Coin) error {
	if k.bankKeeper.GetBalance(ctx, creator, k.stakingKeeper.BondDenom(ctx)).IsLT(price) {
		return utils.LavaFormatWarning("create subscription failed", legacyerrors.ErrInsufficientFunds,
			utils.LogAttr("creator", creator),
			utils.LogAttr("price", price),
		)
	}

	err := k.bankKeeper.SendCoinsFromAccountToModule(ctx, creator, types.ModuleName, []sdk.Coin{price})
	if err != nil {
		return utils.LavaFormatError("create subscription failed. funds transfer failed", err,
			utils.LogAttr("creator", creator),
			utils.LogAttr("price", price),
		)
	}

	return nil
}

func (k Keeper) CreateFutureSubscription(ctx sdk.Context,
	creator string,
	consumer string,
	planIndex string,
	duration uint64,
) error {
	var err error

	block := uint64(ctx.BlockHeight())
	creatorAcct, plan, err := k.verifySubscriptionBuyInputAndGetPlan(ctx, block, creator, consumer, planIndex)
	if err != nil {
		return err
	}

	if duration > types.MAX_SUBSCRIPTION_DURATION {
		str := strconv.FormatInt(types.MAX_SUBSCRIPTION_DURATION, 10)
		return utils.LavaFormatWarning("duration cannot exceed limit ("+str+" months)",
			fmt.Errorf("future subscription failed"),
			utils.Attribute{Key: "duration", Value: duration},
		)
	}

	var sub types.Subscription
	nextEpoch, err := k.epochstorageKeeper.GetNextEpoch(ctx, block)
	if err != nil {
		return utils.LavaFormatError("Got an error while trying to get next epoch on CreateSubscription", err,
			utils.LogAttr("consumer", sub.Consumer),
			utils.LogAttr("block", block),
		)
	}

	// When we make a subscription upgrade, we append the new subscription entry on the next epoch,
	// 	but, we want to get the most updated subscription here.
	// Hence, in case of user making double upgrade in the same epoch, we take the next epoch.
	found := k.subsFS.FindEntry(ctx, consumer, nextEpoch, &sub)

	if !found {
		return utils.LavaFormatWarning("could not find active subscription. advanced purchase is only available for an active subscription", nil,
			utils.LogAttr("consumer", consumer),
			utils.LogAttr("block", block),
		)
	}

	newPlanPrice := plan.GetPrice()
	newPlanPrice.Amount = newPlanPrice.Amount.MulRaw(int64(duration))
	k.applyPlanDiscountIfEligible(duration, &plan, &newPlanPrice)
	chargePrice := newPlanPrice

	if sub.FutureSubscription != nil {
		// Consumer already has a future subscription
		// If the new plan's price > current future subscription's plan - change and charge the diff
		currentPlan, found := k.plansKeeper.FindPlan(ctx, sub.FutureSubscription.PlanIndex, sub.FutureSubscription.PlanBlock)
		if !found {
			return utils.LavaFormatError("panic: could not future subscription's plan. aborting", err,
				utils.Attribute{Key: "creator", Value: creator},
				utils.Attribute{Key: "planIndex", Value: sub.FutureSubscription.PlanIndex},
			)
		}

		if newPlanPrice.Amount.GT(sub.FutureSubscription.Credit.Amount) {
			chargePrice.Amount = newPlanPrice.Amount.Sub(sub.FutureSubscription.Credit.Amount)

			details := map[string]string{
				"creator":      creator,
				"consumer":     consumer,
				"duration":     strconv.FormatUint(duration, 10),
				"oldPlanIndex": currentPlan.Index,
				"oldPlanBlock": strconv.FormatUint(currentPlan.Block, 10),
				"newPlanIndex": plan.Index,
				"newPlanBlock": strconv.FormatUint(plan.Block, 10),
			}
			utils.LogLavaEvent(ctx, k.Logger(ctx), types.AdvancedBuyUpgradeSubscriptionEventName, details, "advanced subscription upgraded")
		} else {
			return utils.LavaFormatWarning("can't purchase another plan in advanced with a lower price", nil)
		}
	}

	err = k.chargeFromCreatorAccountToModule(ctx, creatorAcct, chargePrice)
	if err != nil {
		return err
	}

	sub.FutureSubscription = &types.FutureSubscription{
		Creator:        creator,
		PlanIndex:      plan.Index,
		PlanBlock:      plan.Block,
		DurationBought: duration,
		Credit:         newPlanPrice,
	}

	k.subsFS.ModifyEntry(ctx, consumer, sub.Block, &sub)
	return nil
}

func (k Keeper) RemoveExpiredSubscription(ctx sdk.Context, consumer string, block uint64, planIndex string, planBlock uint64) {
	// delete all projects before deleting
	k.delAllProjectsFromSubscription(ctx, consumer)

	err := k.subsFS.DelEntry(ctx, consumer, block) // minus 1 to avoid deleting an upgraded subscription
	if err != nil {
		utils.LavaFormatError("deleting expired subscription failed", err,
			utils.Attribute{Key: "consumer", Value: consumer},
			utils.Attribute{Key: "block", Value: strconv.FormatUint(block, 10)},
		)
		return
	}

	// decrease plan ref count
	k.plansKeeper.PutPlan(ctx, planIndex, planBlock)

	details := map[string]string{"consumer": consumer}
	utils.LogLavaEvent(ctx, k.Logger(ctx), types.ExpireSubscriptionEventName, details, "subscription expired")
}

func (k Keeper) GetPlanFromSubscription(ctx sdk.Context, consumer string, block uint64) (planstypes.Plan, error) {
	var sub types.Subscription
	if found := k.subsFS.FindEntry(ctx, consumer, block, &sub); !found {
		return planstypes.Plan{}, fmt.Errorf("can't find subscription with consumer address: %s", consumer)
	}

	plan, found := k.plansKeeper.FindPlan(ctx, sub.PlanIndex, sub.PlanBlock)
	if !found {
		// normally would panic! but can "recover" by ignoring and returning error
		err := utils.LavaFormatError("critical: failed to find existing subscription plan", legacyerrors.ErrKeyNotFound,
			utils.Attribute{Key: "consumer", Value: consumer},
			utils.Attribute{Key: "planIndex", Value: sub.PlanIndex},
			utils.Attribute{Key: "planBlock", Value: sub.PlanBlock},
		)
		return plan, err
	}

	return plan, nil
}

func (k Keeper) AddProjectToSubscription(ctx sdk.Context, consumer string, projectData projectstypes.ProjectData) error {
	var sub types.Subscription
	if found := k.subsFS.FindEntry(ctx, consumer, uint64(ctx.BlockHeight()), &sub); !found {
		return legacyerrors.ErrKeyNotFound.Wrapf("AddProjectToSubscription_can't_get_subscription_of_%s", consumer)
	}

	plan, found := k.plansKeeper.FindPlan(ctx, sub.PlanIndex, sub.PlanBlock)
	if !found {
		err := utils.LavaFormatError("critical: failed to find existing subscription plan", legacyerrors.ErrKeyNotFound,
			utils.Attribute{Key: "consumer", Value: sub.Creator},
			utils.Attribute{Key: "planIndex", Value: sub.PlanIndex},
			utils.Attribute{Key: "planBlock", Value: sub.PlanBlock},
		)
		return err
	}

	return k.projectsKeeper.CreateProject(ctx, consumer, projectData, plan)
}

func (k Keeper) DelProjectFromSubscription(ctx sdk.Context, consumer, name string) error {
	var sub types.Subscription
	if found := k.subsFS.FindEntry(ctx, consumer, uint64(ctx.BlockHeight()), &sub); !found {
		return legacyerrors.ErrKeyNotFound.Wrapf("AddProjectToSubscription_can't_get_subscription_of_%s", consumer)
	}

	projectID := projectstypes.ProjectIndex(consumer, name)
	return k.projectsKeeper.DeleteProject(ctx, consumer, projectID)
}

func (k Keeper) delAllProjectsFromSubscription(ctx sdk.Context, consumer string) {
	allProjectsIDs := k.projectsKeeper.GetAllProjectsForSubscription(ctx, consumer)
	for _, projectID := range allProjectsIDs {
		err := k.projectsKeeper.DeleteProject(ctx, consumer, projectID)
		if err != nil {
			// TODO: should panic (should never fail because these are exactly the
			// subscription's current projects)
			utils.LavaFormatError("critical: delete project failed at subscription removal", err,
				utils.Attribute{Key: "subscription", Value: consumer},
				utils.Attribute{Key: "project", Value: projectID},
				utils.Attribute{Key: "block", Value: uint64(ctx.BlockHeight())},
			)
		}
	}
}

func (k Keeper) ChargeComputeUnitsToSubscription(ctx sdk.Context, consumer string, block, cuAmount uint64) (types.Subscription, error) {
	var sub types.Subscription
	if found := k.subsFS.FindEntry(ctx, consumer, block, &sub); !found {
		return sub, utils.LavaFormatError("can't charge cu to subscription",
			fmt.Errorf("subscription not found"),
			utils.Attribute{Key: "subscription", Value: consumer},
			utils.Attribute{Key: "block", Value: block},
		)
	}

	if sub.MonthCuLeft < cuAmount {
		sub.MonthCuLeft = 0
	} else {
		sub.MonthCuLeft -= cuAmount
	}

	utils.LavaFormatDebug("charging sub for cu amonut",
		utils.LogAttr("sub", consumer),
		utils.LogAttr("sub_block", sub.Block),
		utils.LogAttr("charge_cu", cuAmount),
		utils.LogAttr("month_cu_left", sub.MonthCuLeft))
	k.subsFS.ModifyEntry(ctx, consumer, sub.Block, &sub)
	return sub, nil
}
