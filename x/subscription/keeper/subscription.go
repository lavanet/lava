package keeper

import (
	"strconv"
	"time"

	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/lavanet/lava/utils"
	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
	"github.com/lavanet/lava/x/subscription/types"
)

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

// endOfMonth returns the date of the last day of the month (assume UTC)
// (https://yourbasic.org/golang/last-day-month-date/)
func endOfMonth(date time.Time) time.Time {
	return date.AddDate(0, 1, -date.Day())
}

// isLeapYear returns true if the year in the date is a leap year (assume UTC)
// (https://stackoverflow.com/questions/725098/leap-year-calculation)
func isLeapYear(date time.Time) bool {
	year := date.Year()
	// Leap year occurs every 4 years; but every 100 years it is skipped;
	// except that every every 400 years it is kept.
	return year%400 == 0 || (year%4 == 0 && year%100 != 0)
}

// nextMonth returns the date of the same day next month (assumes UTC),
// adjusting for end-of-months differences if needed.
func nextMonth(date time.Time) time.Time {
	next := date.AddDate(0, 1, 0)

	// https://golang.org/pkg/time/#Time.AddDate:
	//   AddDate normalizes its result in the same way that Date does, so, for
	//   example, adding one month to October 31 yields December 1, the normalized
	//   form for November 31.
	//
	// If we are at end of this month, then "manually" select end of next month;
	// This properly handles transitions from short to longer months.

	if date.Day() == endOfMonth(date).Day() {
		next = time.Date(
			date.Year(),
			date.Month()+1,
			1,
			date.Hour(),
			date.Minute(),
			date.Second(),
			date.Nanosecond(),
			time.UTC)
		return endOfMonth(next)
	}

	// If we are reaching end of January, stll need to "manually" select end of
	// next month, otherwise will overrun into March.

	if date.Month() == 1 && date.Day() >= 29 {
		next = time.Date(
			date.Year(),
			time.February,
			1,
			date.Hour(),
			date.Minute(),
			date.Second(),
			date.Nanosecond(),
			time.UTC)
		return endOfMonth(next)
	}

	return next
}

// nextYear returns the date of the same day next year (assumes UTC),
// properly handling leap year variations on February.
func nextYear(date time.Time) time.Time {
	next := date.AddDate(1, 0, 0)

	if date.Month() == 2 {
		if date.Day() == 29 {
			next = date.AddDate(1, 0, -1)
		}
		if date.Day() == 28 && isLeapYear(next) {
			next = date.AddDate(1, 0, 1)
		}
	}

	return next
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

	logger := k.Logger(ctx)
	block := uint64(ctx.BlockHeight())

	_, err = sdk.AccAddressFromBech32(consumer)
	if err != nil {
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

	// only one subscription per consumer
	if _, found := k.GetSubscription(ctx, consumer); found {
		details := map[string]string{"consumer": consumer}
		return utils.LavaError(ctx, logger, "CreateSubscription", details, "consumer has existing subscription")
	}

	plan, found := k.plansKeeper.GetPlan(ctx, planIndex)
	if !found {
		details := map[string]string{
			"plan":  planIndex,
			"block": strconv.FormatInt(ctx.BlockHeight(), 10),
		}
		return utils.LavaError(ctx, logger, "CreateSubscription", details, "invalid plan")
	}

	err = k.projectsKeeper.CreateDefaultProject(ctx, consumer)
	if err != nil {
		details := map[string]string{
			"err": err.Error(),
		}
		return utils.LavaError(ctx, logger, "CreateSubscription", details, "failed to create default project")
	}

	price := plan.GetPrice()

	// use current block's timestamp for subscription start-time
	timestamp := ctx.BlockTime()
	expiry := timestamp

	for i := 0; i < int(duration); i++ {
		expiry = nextMonth(expiry)
	}

	if duration == 12 {
		// extend the duration to 1 year, and price pro-rata
		expiry = nextYear(timestamp)
		price.Amount = price.Amount.MulRaw(12)

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

	sub := types.Subscription{
		Creator:     creator,
		Consumer:    consumer,
		Block:       block,
		PlanIndex:   planIndex,
		PlanBlock:   plan.Block,
		Duration:    duration,
		ExpiryTime:  uint64(expiry.Unix()),
		RemainingCU: plan.GetComputeUnits(),
		UsedCU:      0,
	}

	k.SetSubscription(ctx, sub)

	return nil
}
