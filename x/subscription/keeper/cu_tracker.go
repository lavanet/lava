package keeper

import (
	"fmt"
	"strconv"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/utils"
	"github.com/lavanet/lava/x/subscription/types"
)

func (k Keeper) GetTrackedCu(ctx sdk.Context, sub string, provider string, chainID string, block uint64) (cu uint64, found bool, key string) {
	cuTrackerKey := types.CuTrackerKey(sub, provider, chainID)
	var trackedCu types.TrackedCu
	found = k.cuTrackerFS.FindEntry(ctx, cuTrackerKey, block, &trackedCu)
	return trackedCu.GetCu(), found, cuTrackerKey
}

func (k Keeper) AddTrackedCu(ctx sdk.Context, sub string, provider string, chainID string, cu uint64, block uint64) error {
	if sub == "" || provider == "" || block > uint64(ctx.BlockHeight()) {
		return utils.LavaFormatError("cannot add tracked CU",
			fmt.Errorf("sub/provider cannot be empty. block cannot be larger than current block"),
			utils.Attribute{Key: "sub", Value: sub},
			utils.Attribute{Key: "provider", Value: provider},
			utils.Attribute{Key: "block", Value: strconv.FormatUint(block, 10)},
		)
	}
	trackedCu, _, key := k.GetTrackedCu(ctx, sub, provider, chainID, block)
	err := k.cuTrackerFS.AppendEntry(ctx, key, block, &types.TrackedCu{Cu: trackedCu + cu})
	if err != nil {
		return utils.LavaFormatError("cannot add tracked CU", err,
			utils.Attribute{Key: "tracked_cu_key", Value: key},
			utils.Attribute{Key: "block", Value: strconv.FormatUint(block, 10)},
			utils.Attribute{Key: "current_cu", Value: trackedCu},
			utils.Attribute{Key: "cu_to_be_added", Value: strconv.FormatUint(cu, 10)})
	}
	return nil
}
