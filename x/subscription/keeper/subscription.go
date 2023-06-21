package keeper

import (
	"fmt"
	"strconv"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	commontypes "github.com/lavanet/lava/common/types"
	"github.com/lavanet/lava/utils"
	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
	planstypes "github.com/lavanet/lava/x/plans/types"
	projectstypes "github.com/lavanet/lava/x/projects/types"
	"github.com/lavanet/lava/x/subscription/types"
)

const MONTHS_IN_YEAR = 12

// NextMonth returns the date of the same day next month (assumes UTC),
// adjusting for end-of-months differences if needed.
func NextMonth(date time.Time) time.Time {
	// End-of-month days are tricky because months differ in days counts.
	// To avoid this complixity, we trim day-of-month greater than 28 back to
	// day 28, which all months always have (at the cost of the user possibly
	// losing 1 (and up to 3) days of subscription in the first month.

	dayOfMonth := date.Day()
	if dayOfMonth > 28 {
		dayOfMonth = 28
	}

	return time.Date(
		date.Year(),
		date.Month()+1,
		dayOfMonth,
		date.Hour(),
		date.Minute(),
		date.Second(),
		0,
		time.UTC,
	)
}

// ExportSubscriptions exports subscriptions data (for genesis)
func (k Keeper) ExportSubscriptions(ctx sdk.Context) []commontypes.RawMessage {
	return k.subsFS.Export(ctx)
}

// InitSubscriptions imports subscriptions data (from genesis)
func (k Keeper) InitSubscriptions(ctx sdk.Context, data []commontypes.RawMessage) {
	k.subsFS.Init(ctx, data)
}

// GetSubscription returns the subscription of a given consumer
func (k Keeper) GetSubscription(ctx sdk.Context, consumer string) (val types.Subscription, found bool) {
	block := uint64(ctx.BlockHeight())

	var sub types.Subscription
	found = k.subsFS.FindEntry(ctx, consumer, block, &sub)

	return sub, found
}

// CreateSubscription creates a subscription for a consumer
func (k Keeper) CreateSubscription(
	ctx sdk.Context,
	creator string,
	consumer string,
	planIndex string,
	duration uint64,
) error {
	var err error

	block := uint64(ctx.BlockHeight())

	if _, err = sdk.AccAddressFromBech32(consumer); err != nil {
		return utils.LavaFormatWarning("invalid subscription concumer address", err,
			utils.Attribute{Key: "consumer", Value: consumer},
		)
	}

	creatorAcct, err := sdk.AccAddressFromBech32(creator)
	if err != nil {
		return utils.LavaFormatWarning("invalid subscription creator address", err,
			utils.Attribute{Key: "creator", Value: creator},
		)
	}

	plan, found := k.plansKeeper.GetPlan(ctx, planIndex)
	if !found {
		return utils.LavaFormatWarning("cannot create subcscription with invalid plan", err,
			utils.Attribute{Key: "plan", Value: planIndex},
			utils.Attribute{Key: "block", Value: block},
		)
	}

	var sub types.Subscription
	found = k.subsFS.FindEntry(ctx, consumer, block, &sub)

	// Subscription creation:
	//   When: if not already exists for consumer address)
	//   What: find plan, create default project, set duration, calculate price,
	//         charge fees, save subscription.
	//
	// Subscription renewal:
	//   When: if already exists and existing plan is the same as current plans
	//         ("same" means same index and same block of creation)
	//   What: find plan, update duration (total and remaining), calculate price,
	//         charge fees, save subscription.
	//
	// Subscription upgrade: (TBD)
	//
	// Subscription downgrade: (TBD)

	if !found {
		// creeate new subscription with this plan
		sub = types.Subscription{
			Creator:   creator,
			Consumer:  consumer,
			Block:     block,
			PlanIndex: planIndex,
			PlanBlock: plan.Block,
		}

		sub.MonthCuTotal = plan.PlanPolicy.GetTotalCuLimit()
		sub.MonthCuLeft = plan.PlanPolicy.GetTotalCuLimit()

		// new subscription needs a default project
		err = k.projectsKeeper.CreateAdminProject(ctx, consumer, plan)
		if err != nil {
			return utils.LavaFormatWarning("failed to create default project", err)
		}
	} else {
		// allow renewal with the same plan ("same" means both plan index,block match);
		// otherwise, only one subscription per consumer
		if !(plan.Index == sub.PlanIndex && plan.Block == sub.PlanBlock) {
			return utils.LavaFormatWarning("consumer has existing subscription with a different plan",
				fmt.Errorf("subscription renewal failed"),
				utils.Attribute{Key: "consumer", Value: consumer},
			)
		}

		// The total duration may not exceed MAX_SUBSCRIPTION_DURATION, but allow an
		// extra month to account for renwewals before the end of current subscription
		if sub.DurationLeft+duration > types.MAX_SUBSCRIPTION_DURATION+1 {
			str := strconv.FormatInt(types.MAX_SUBSCRIPTION_DURATION, 10)
			return utils.LavaFormatWarning("duration would exceed limit ("+str+" months)",
				fmt.Errorf("subscription renewal failed"),
				utils.Attribute{Key: "duration", Value: sub.DurationLeft},
			)
		}
	}

	// update total (last requested) duration and remaining duration
	sub.DurationTotal = duration
	sub.DurationLeft += duration

	if err := sub.ValidateSubscription(); err != nil {
		return utils.LavaFormatWarning("create subscription failed", err)
	}

	// subscription looks good; let's charge the creator
	price := plan.GetPrice()
	price.Amount = price.Amount.MulRaw(int64(duration))

	if duration >= MONTHS_IN_YEAR {
		// adjust cost if discount given
		discount := plan.GetAnnualDiscountPercentage()
		if discount > 0 {
			factor := int64(100 - discount)
			price.Amount = price.Amount.MulRaw(factor).QuoRaw(100)
		}
	}

	if k.bankKeeper.GetBalance(ctx, creatorAcct, epochstoragetypes.TokenDenom).IsLT(price) {
		return utils.LavaFormatWarning("create subscription failed", sdkerrors.ErrInsufficientFunds,
			utils.Attribute{Key: "creator", Value: creator},
			utils.Attribute{Key: "price", Value: price},
		)
	}

	err = k.bankKeeper.SendCoinsFromAccountToModule(ctx, creatorAcct, types.ModuleName, []sdk.Coin{price})
	if err != nil {
		return utils.LavaFormatError("create subscription failed. funds transfer failed", err,
			utils.Attribute{Key: "creator", Value: creator},
			utils.Attribute{Key: "price", Value: price},
		)
	}

	if !found {
		expiry := uint64(NextMonth(ctx.BlockTime()).UTC().Unix())
		sub.MonthExpiryTime = expiry
		k.subsTS.AddTimerByBlockTime(ctx, expiry, []byte(consumer), []byte{})
		err = k.subsFS.AppendEntry(ctx, consumer, block, &sub)
	} else {
		k.subsFS.ModifyEntry(ctx, consumer, sub.Block, &sub)
	}

	return err
}

func (k Keeper) advanceMonth(ctx sdk.Context, subkey []byte) {
	date := ctx.BlockTime()
	block := uint64(ctx.BlockHeight())
	consumer := string(subkey)

	var sub types.Subscription
	if found := k.subsFS.FindEntry(ctx, consumer, block, &sub); !found {
		// subscription (monthly) timer has expired for an unknown subscription:
		// either the timer was set wrongly, or the subscription was incorrectly
		// removed; and we cannot even return an error about it.
		utils.LavaFormatError("critical: month expirty for unknown subscription, skipping",
			fmt.Errorf("subscription not found"),
			utils.Attribute{Key: "consumer", Value: consumer},
			utils.Attribute{Key: "block", Value: block},
		)
		return
	}

	if sub.DurationLeft == 0 {
		utils.LavaFormatError("critical: negative duration for subscription, extend by 1 month",
			fmt.Errorf("negative duration in AdvanceMonth()"),
			utils.Attribute{Key: "consumer", Value: consumer},
			utils.Attribute{Key: "plan", Value: sub.PlanIndex},
			utils.Attribute{Key: "block", Value: block},
		)
		// normally would panic! but can "recover" by auto-extending by 1 month
		// (don't bother to modfy sub.MonthExpiryTime to minimize state changes)
		expiry := uint64(NextMonth(date).UTC().Unix())
		k.subsTS.AddTimerByBlockTime(ctx, expiry, []byte(consumer), []byte{})
		return
	}

	sub.DurationLeft -= 1

	if sub.DurationLeft > 0 {
		// reset projects CU allowance for this coming month
		k.projectsKeeper.SnapshotSubscriptionProjects(ctx, sub.Consumer)

		// reset subscription CU allowance for this coming month
		sub.MonthCuLeft = sub.MonthCuTotal
		sub.Block = block

		// restart timer and append new (fixated) version of this subscription
		expiry := uint64(NextMonth(date).UTC().Unix())
		sub.MonthExpiryTime = expiry
		k.subsTS.AddTimerByBlockTime(ctx, expiry, []byte(consumer), []byte{})
		err := k.subsFS.AppendEntry(ctx, consumer, block, &sub)
		if err != nil {
			// normally would panic! but ok to ignore - the subscription remains
			// as is with same remaining duration (but not renewed CU)
			utils.LavaFormatError("critical: failed to recharge subscription", err,
				utils.Attribute{Key: "consumer", Value: consumer},
				utils.Attribute{Key: "plan", Value: sub.PlanIndex},
				utils.Attribute{Key: "block", Value: block},
			)
		}
	} else {
		// delete all projects before deleting
		k.delAllProjectsFromSubscription(ctx, consumer)

		// delete subscription effective now (don't wait for end of epoch)
		k.subsFS.DelEntry(ctx, sub.Consumer, block)
	}
}

func (k Keeper) GetPlanFromSubscription(ctx sdk.Context, consumer string) (planstypes.Plan, error) {
	var sub types.Subscription
	if found := k.subsFS.FindEntry(ctx, consumer, uint64(ctx.BlockHeight()), &sub); !found {
		return planstypes.Plan{}, utils.LavaFormatWarning("can't find subscription with consumer address", sdkerrors.ErrKeyNotFound,
			utils.Attribute{Key: "consumer", Value: consumer},
		)
	}

	plan, found := k.plansKeeper.FindPlan(ctx, sub.PlanIndex, sub.PlanBlock)
	if !found {
		err := utils.LavaFormatError("can't find plan from subscription with consumer address", sdkerrors.ErrKeyNotFound,
			utils.Attribute{Key: "consumer", Value: consumer},
			utils.Attribute{Key: "plan", Value: sub.PlanIndex},
		)
		panic(err)
	}

	return plan, nil
}

func (k Keeper) AddProjectToSubscription(ctx sdk.Context, consumer string, projectData projectstypes.ProjectData) error {
	var sub types.Subscription
	if found := k.subsFS.FindEntry(ctx, consumer, uint64(ctx.BlockHeight()), &sub); !found {
		return sdkerrors.ErrKeyNotFound.Wrapf("AddProjectToSubscription_can't_get_subscription_of_%s", consumer)
	}

	plan, found := k.plansKeeper.FindPlan(ctx, sub.PlanIndex, sub.PlanBlock)
	if !found {
		err := utils.LavaFormatError("can't get plan with subscription", sdkerrors.ErrKeyNotFound,
			utils.Attribute{Key: "subscription", Value: sub.Creator},
			utils.Attribute{Key: "plan", Value: sub.PlanIndex},
		)
		panic(err)
	}

	return k.projectsKeeper.CreateProject(ctx, consumer, projectData, plan)
}

func (k Keeper) DelProjectFromSubscription(ctx sdk.Context, consumer string, name string) error {
	var sub types.Subscription
	if found := k.subsFS.FindEntry(ctx, consumer, uint64(ctx.BlockHeight()), &sub); !found {
		return sdkerrors.ErrKeyNotFound.Wrapf("AddProjectToSubscription_can't_get_subscription_of_%s", consumer)
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

func (k Keeper) ChargeComputeUnitsToSubscription(ctx sdk.Context, consumer string, block uint64, cuAmount uint64) error {
	var sub types.Subscription
	if found := k.subsFS.FindEntry(ctx, consumer, block, &sub); !found {
		return utils.LavaFormatError("can't charge cu to subscription",
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

	k.subsFS.ModifyEntry(ctx, consumer, sub.Block, &sub)
	return nil
}
