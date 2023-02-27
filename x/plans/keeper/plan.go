package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/utils"
	"github.com/lavanet/lava/x/plans/types"
)

// AddPlan adds a new plan to the KVStore. The function returns if the added plan is a first version plans
func (k Keeper) AddPlan(ctx sdk.Context, planToAdd types.Plan) error {
	// overwrite the planToAdd's block field with the current block height
	planToAdd.Block = uint64(ctx.BlockHeight())

	// TODO: verify the CU per epoch field

	err := k.plansFs.AppendEntry(ctx, planToAdd.GetIndex(), planToAdd.GetBlock(), &planToAdd)
	if err != nil {
		return utils.LavaError(ctx, k.Logger(ctx), "AddPlan_add_fixated_entry_failed", map[string]string{"planToAdd": planToAdd.String()}, "could not add new plan fixated entry to storage")
	}

	return nil
}
