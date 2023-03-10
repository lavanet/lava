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

	sub := types.Subscription{
		Creator:       creator,
		Consumer:      consumer,
		Block:         block,
		PlanIndex:     planIndex,
		PlanBlock:     plan.Block,
		DurationTotal: duration,
		MonthCuTotal:  plan.GetComputeUnits(),
	}

	// use current block's timestamp for subscription start-time
	timestamp := ctx.BlockTime()
	expiry := timestamp

	for i := 0; i < int(duration); i++ {
		expiry = nextMonth(expiry)
	}

	sub.MonthExpiryTime = uint64(expiry.Unix())

	sub.MonthCuLeft = plan.GetComputeUnits()
	sub.DurationLeft = duration

	if err := sub.ValidateSubscription(); err != nil {
		return utils.LavaError(ctx, logger, "CreateSub", nil, err.Error())
	}

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
