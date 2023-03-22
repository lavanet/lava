package keeper

import (
	"math"
	"strconv"
	"time"

	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/lavanet/lava/utils"
	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
	"github.com/lavanet/lava/x/subscription/types"
)

const MONTHS_IN_YEAR = 12

// SetSubscription sets a subscription (of a consumer) in the store
func (k Keeper) SetSubscription(ctx sdk.Context, sub types.Subscription) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.SubscriptionKeyPrefix))
	b := k.cdc.MustMarshal(&sub)
	store.Set(types.SubscriptionKey(sub.Consumer), b)
}

// GetSubscription returns the subscription of a given consumer
func (k Keeper) GetSubscription(ctx sdk.Context, consumer string) (val types.Subscription, found bool) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.SubscriptionKeyPrefix))

	b := store.Get(types.SubscriptionKey(consumer))
	if b == nil {
		return val, false
	}

	k.cdc.MustUnmarshal(b, &val)
	return val, true
}

// RemoveSubscription removes the subscription (of a consumer) from the store
func (k Keeper) RemoveSubscription(ctx sdk.Context, consumer string) {
	sub, _ := k.GetSubscription(ctx, consumer)

	// (PlanIndex is empty only in testing of RemoveSubscription)
	if sub.PlanIndex != "" {
		k.plansKeeper.PutPlan(ctx, sub.PlanIndex, sub.Block)
	}

	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.SubscriptionKeyPrefix))
	store.Delete(types.SubscriptionKey(consumer))
}

// GetAllSubscription returns all subscriptions that satisfy the condition
func (k Keeper) GetCondSubscription(ctx sdk.Context, cond func(sub types.Subscription) bool) (list []types.Subscription) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.SubscriptionKeyPrefix))
	iterator := sdk.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.Subscription
		k.cdc.MustUnmarshal(iterator.Value(), &val)
		if cond == nil || cond(val) {
			list = append(list, val)
		}
	}

	return
}

// GetAllSubscription returns all subscription (of all consumers)
func (k Keeper) GetAllSubscription(ctx sdk.Context) []types.Subscription {
	return k.GetCondSubscription(ctx, nil)
}

// nextMonth returns the date of the same day next month (assumes UTC),
// adjusting for end-of-months differences if needed.
func nextMonth(date time.Time) time.Time {
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

// CreateSubscription creates a subscription for a consumer
func (k Keeper) CreateSubscription(
	ctx sdk.Context,
	creator string,
	consumer string,
	planIndex string,
	duration uint64,
	vrfpk string,
) error {
	var err error

	logger := k.Logger(ctx)
	block := uint64(ctx.BlockHeight())

	if _, err = sdk.AccAddressFromBech32(consumer); err != nil {
		details := map[string]string{
			"consumer": consumer,
			"error":    err.Error(),
		}
		return utils.LavaError(ctx, logger, "CreateSubscription", details, "invalid consumer")
	}

	creatorAcct, err := sdk.AccAddressFromBech32(creator)
	if err != nil {
		details := map[string]string{
			"creator": creator,
			"error":   err.Error(),
		}
		return utils.LavaError(ctx, logger, "CreateSubscription", details, "invalid creator")
	}

	plan, found := k.plansKeeper.GetPlan(ctx, planIndex)
	if !found {
		details := map[string]string{
			"plan":  planIndex,
			"block": strconv.FormatInt(int64(block), 10),
		}
		return utils.LavaError(ctx, logger, "CreateSubscription", details, "invalid plan")
	}

	sub, found := k.GetSubscription(ctx, consumer)

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

		sub.MonthCuTotal = plan.GetComputeUnits()
		sub.MonthCuLeft = plan.GetComputeUnits()

		// new subscription needs a default project
		err = k.projectsKeeper.CreateAdminProject(ctx, consumer, plan, vrfpk)
		if err != nil {
			details := map[string]string{
				"err": err.Error(),
			}
			return utils.LavaError(ctx, logger, "CreateSubscription", details, "failed to create default project")
		}
	} else {
		// allow renewal with the same plan ("same" means both plan index,block match);
		// otherwise, only one subscription per consumer
		if !(plan.Index == sub.PlanIndex && plan.Block == sub.PlanBlock) {
			details := map[string]string{"consumer": consumer}
			return utils.LavaError(ctx, logger, "CreateSubscription", details, "consumer has existing subscription with a different plan")
		}

		// For now, allow renewal only by the same creator.
		// TODO: after adding fixation, we can allow different creators
		if creator != sub.Creator {
			details := map[string]string{"creator": consumer}
			return utils.LavaError(ctx, logger, "CreateSubscription", details, "existing subscription has different creator")
		}

		// The total duration may not exceed MAX_SUBSCRIPTION_DURATION, but allow an
		// extra month to account for renwewals before the end of current subscription
		if sub.DurationLeft+duration > types.MAX_SUBSCRIPTION_DURATION+1 {
			details := map[string]string{"duration": strconv.FormatInt(int64(sub.DurationLeft), 10)}
			msg := "duration would exceed limit (" + strconv.FormatInt(types.MAX_SUBSCRIPTION_DURATION, 10) + " months)"
			return utils.LavaError(ctx, logger, "CreateSubscription", details, msg)
		}
	}

	// update total (last requested) duration and remaining duration
	sub.DurationTotal = duration
	sub.DurationLeft += duration

	// use current block's timestamp to calculate next month's time
	timestamp := ctx.BlockTime()
	expiry := timestamp

	for i := 0; i < int(duration); i++ {
		expiry = nextMonth(expiry)
	}

	sub.MonthExpiryTime = uint64(expiry.Unix())

	if err := sub.ValidateSubscription(); err != nil {
		return utils.LavaError(ctx, logger, "CreateSub", nil, err.Error())
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
		details := map[string]string{
			"creator": creator,
			"price":   price.String(),
			"error":   sdkerrors.ErrInsufficientFunds.Error(),
		}
		return utils.LavaError(ctx, logger, "CreateSub", details, "insufficient funds")
	}

	err = k.bankKeeper.SendCoinsFromAccountToModule(ctx, creatorAcct, types.ModuleName, []sdk.Coin{price})
	if err != nil {
		details := map[string]string{
			"creator": creator,
			"price":   price.String(),
			"error":   err.Error(),
		}
		return utils.LavaError(ctx, logger, "CreateSubscription", details, "funds transfer failed")
	}

	k.SetSubscription(ctx, sub)

	return nil
}

func (k Keeper) AddProjectToSubscription(ctx sdk.Context, subscriptionOwner string, projectAdmin string, projectName string, enabled bool, projectDescription string, vrfpk string) error {
	sub, found := k.GetSubscription(ctx, subscriptionOwner)
	if !found {
		details := map[string]string{
			"subscriptionOwner": subscriptionOwner,
		}
		return utils.LavaError(ctx, k.Logger(ctx), "AddProjectToSubscription", details, "can't get subscription")
	}

	plan, found := k.plansKeeper.GetPlan(ctx, sub.GetPlanIndex())
	if !found {
		details := map[string]string{
			"subscriptionConsumer": projectAdmin,
			"planIndex":            sub.GetPlanIndex(),
		}
		err := utils.LavaError(ctx, k.Logger(ctx), "AddProjectToSubscription", details, "can't get plan with subscription")
		panic(err)
	}

	return k.projectsKeeper.CreateProject(ctx, subscriptionOwner, projectName, projectAdmin, enabled, projectDescription, plan, math.MaxUint64, vrfpk)
}
