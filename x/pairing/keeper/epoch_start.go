package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/utils"
)

// Function that calls all the functions that are supposed to run in epoch start
func (k Keeper) EpochStart(ctx sdk.Context, epochsNumToCheckCuForUnresponsiveProvider uint64, epochsNumToCheckForComplainers uint64) {
	logOnErr := func(err error, failingFunc string) {
		if err != nil {
			utils.LavaFormatError("failing func: "+failingFunc, err)
		}
	}
	// on session start we need to do:
	// 1. remove old session payments
	// 2. unstake any unstaking providers
	// 3. unstake any unstaking users
	// 4. unstake/jail unresponsive providers

	// 1.
	err := k.RemoveOldEpochPayment(ctx)
	logOnErr(err, "RemoveOldEpochPayment")

	// 2+3.
	err = k.CheckUnstakingForCommit(ctx)
	logOnErr(err, "CheckUnstakingForCommit")

	// 4. unstake unresponsive providers
	err = k.UnstakeUnresponsiveProviders(ctx, epochsNumToCheckCuForUnresponsiveProvider, epochsNumToCheckForComplainers)
	logOnErr(err, "UnstakeUnresponsiveProviders")
}
