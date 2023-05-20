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

	err := k.plansFS.AppendEntry(ctx, planToAdd.GetIndex(), planToAdd.Block, &planToAdd)
	if err != nil {
		return utils.LavaFormatError("failed adding plan to planFS", err,
			utils.Attribute{Key: "planToAdd", Value: planToAdd},
		)
	}

	return nil
}

// DelPlan deletes a plan, so it is not visible/gettable for new subscriptions
// (however, existing referenced versions remain intact until not used anymore)
func (k Keeper) DelPlan(ctx sdk.Context, index string) error {
	// Deletions should take place at the end of epoch (beginning of next epoch).
	// However, because deletion of plans occurs through gov proposal, we must be
	// already at the beginning of that "next" epoch (after having waited for the
	// already at the beginning of that "next" epoch (after having waited for the
	// the voting). Hence, we delete at current block height.

	return k.plansFS.DelEntry(ctx, index, uint64(ctx.BlockHeight()))
}

// GetPlan gets the latest plan from the KVStore and increments its refcount
func (k Keeper) GetPlan(ctx sdk.Context, index string) (val types.Plan, found bool) {
	var plan types.Plan
	if found := k.plansFS.GetEntry(ctx, index, &plan); !found {
		return types.Plan{}, false
	}
	return plan, true
}

// FindPlan gets a plan with nearest-smaller block (without changing its refcount)
func (k Keeper) FindPlan(ctx sdk.Context, index string, block uint64) (val types.Plan, found bool) {
	var plan types.Plan
	if found := k.plansFS.FindEntry(ctx, index, block, &plan); !found {
		return types.Plan{}, false
	}
	return plan, true
}

// PutPlan finds a plan with nearest-smaller block and decrements its refcount
func (k Keeper) PutPlan(ctx sdk.Context, index string, block uint64) {
	k.plansFS.PutEntry(ctx, index, block)
}

// GetAllPlanIndices gets from the KVStore all the plans' indices
func (k Keeper) GetAllPlanIndices(ctx sdk.Context) (val []string) {
	return k.plansFS.GetAllEntryIndices(ctx)
}
