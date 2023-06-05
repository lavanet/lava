package keeper

import (
	"strconv"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/x/subscription/types"
)

// EpochStart runs the functions that are supposed to run in epoch start
func (k Keeper) EpochStart(ctx sdk.Context) {
	// On epoch start we need to iterate through all the subscriptions and
	// check those whose current month just expired:
	//
	// - record the actual epoch of expiry (for rewards validation)
	// - save the month's remaining CU in prev (for rewards validation)
	// - reset the month's remaining CU to the plan's allowance
	// - reduce remaining duration, and delete if it reaches zero
	//
	// Note that actual deletion is deferred by EpochsToSave parameter
	// (in Epochstorage) to allow payments for the last month of the
	// subscription. (During this period, remaining CU of the new month
	// is zero).

	date := ctx.BlockTime().UTC()
	block := uint64(ctx.BlockHeight())

	blocksToSave, err := k.epochstorageKeeper.BlocksToSave(ctx, block)
	if err != nil {
		// critical: no recovery from this
		panic("Subscription: EpochStart: failed to obtain BlocksToSave at block " + strconv.Itoa(int(block)))
	}

	// not the end of memory yet, do nothing
	if block < blocksToSave {
		return
	}

	subExpired := k.GetCondSubscription(ctx, func(sub types.Subscription) bool {
		return sub.IsMonthExpired(date) || sub.IsStale(block-blocksToSave)
	})

	for _, sub := range subExpired {
		// subscription has been dead for EpochsToSave epochs: delete
		// TODO: disable all projects registered in this subscription
		// TODO: THIS WILL BE HANDLED AUTOMATICALLY BY FIXATION-STORE
		// (WHICH WILL BE ADDED IN SUBSEQUENT PR)
		if sub.IsStale(block - blocksToSave) {
			k.RemoveSubscription(ctx, sub.Consumer)
			continue
		}

		sub.PrevExpiryBlock = block
		sub.PrevCuLeft = sub.MonthCuLeft

		if sub.DurationLeft == 0 {
			panic("Subscription: EpochStart: negative DurationLeft for consumer " + sub.Consumer)
		}

		sub.DurationLeft -= 1

		if sub.DurationLeft > 0 {
			date = nextMonth(date)
			sub.MonthExpiryTime = uint64(date.Unix())

			// reset subscription CU allowance for this coming month
			sub.MonthCuLeft = sub.MonthCuTotal

			// reset projects' CU allowance for this coming month
			k.projectsKeeper.SnapshotSubscriptionProjects(ctx, sub.Consumer)
		} else {
			// duration ended, but don't delete yet - keep around for another
			// EpochsToSave epochs before removing, to allow for payments for
			// the months the ends now to be validated.

			// set expiry timeout far far in the future, so the test
			// for sub.IsStale() above would kick in first.
			date = date.Add(8760 * time.Hour) // 1 yr
			sub.MonthExpiryTime = uint64(date.Unix())

			// zero CU allowance for this coming month
			sub.MonthCuLeft = 0
		}

		k.SetSubscription(ctx, sub)
	}
}
