package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/utils"
	"github.com/lavanet/lava/x/plans/types"
)

// AddPlan adds a new plan to the KVStore
func (k Keeper) AddPlan(ctx sdk.Context, planToAdd types.Plan) error {
	// overwrite the planToAdd's block field with the current block height
	planToAdd.Block = uint64(ctx.BlockHeight())

	// TODO: verify the CU per epoch field

	err := k.plansFs.AppendEntry(ctx, planToAdd.GetIndex(), planToAdd.Block, &planToAdd)
	if err != nil {
		details := map[string]string{"planToAdd": planToAdd.String()}
		return utils.LavaError(ctx, k.Logger(ctx), "AddPlan_add_fixated_entry_failed", details, err.Error())
	}

	return nil
}

// GetPlan gets the latest plan from the KVStore and increments its refcount
func (k Keeper) GetPlan(ctx sdk.Context, index string) (val types.Plan, found bool) {
	var plan types.Plan
	if found := k.plansFs.GetEntry(ctx, index, &plan); !found {
		return types.Plan{}, false
	}
	return plan, true
}

// FindPlan gets a plan with nearest-smaller block (without changing its refcount)
func (k Keeper) FindPlan(ctx sdk.Context, index string, block uint64) (val types.Plan, found bool) {
	var plan types.Plan
	if found := k.plansFs.FindEntry(ctx, index, block, &plan); !found {
		return types.Plan{}, false
	}
	return plan, true
}

// PutPlan finds a plan with nearest-smaller block and decrements its refcount
func (k Keeper) PutPlan(ctx sdk.Context, index string, block uint64) {
	k.plansFs.PutEntry(ctx, index, block)
}

// GetAllPlanIndices gets from the KVStore all the plans' indices
func (k Keeper) GetAllPlanIndices(ctx sdk.Context) (val []string) {
	return k.plansFs.GetAllEntryIndices(ctx)
}
