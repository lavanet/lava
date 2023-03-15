package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/x/subscription/types"
)

// EpochStart runs the functions that are supposed to run in epoch start
func (k Keeper) EpochStart(ctx sdk.Context) {
	// On epoch start we need to iterate through all the subscriptions and
	// check if they just expired. Those that expire get deleted.

	date := ctx.BlockTime().UTC()
	subExpired := k.GetCondSubscription(ctx, func(sub types.Subscription) bool {
		return sub.IsExpired(date)
	})

	// First collect, then remove (because removing may affect the iterator?)
	for _, sub := range subExpired {
		k.RemoveSubscription(ctx, sub.Consumer)
	}
}
