package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/utils"
)

func (k Keeper) EpochStart(ctx sdk.Context) {
	logger := k.Logger(ctx)
	logOnErr := func(err error, failingFunc string) {
		if err != nil {
			attrs := map[string]string{"error": err.Error()}
			utils.LavaError(ctx, logger, "new_epoch", attrs, failingFunc)
		}
	}

	// on epoch start we need to do:
	// 1. delete stale packages (package was edited and there are no subs for it. Also, currentEpoch > packageEpoch + packageDuration)
	// get packagesToDeleteMap
	err := k.deleteOldPackages(ctx)
	logOnErr(err, "deleteOldPackages")
}
