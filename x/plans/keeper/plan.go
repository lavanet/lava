package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/utils"
	"github.com/lavanet/lava/x/plans/types"
	projecttypes "github.com/lavanet/lava/x/projects/types"
)

// AddPlan adds a new plan to the KVStore. The function returns if the added plan is a first version plans
func (k Keeper) AddPlan(ctx sdk.Context, planToAdd types.Plan) error {
	// overwrite the planToAdd's block field with the current block height
	planToAdd.Block = uint64(ctx.BlockHeight())

	// TODO: verify the CU per epoch field

	err := k.plansFs.AppendEntry(ctx, planToAdd.GetIndex(), planToAdd.Block, &planToAdd)
	if err != nil {
		details := map[string]string{"planToAdd": planToAdd.String()}
		return utils.LavaError(ctx, k.Logger(ctx), "AddPlan_add_fixated_entry_failed", details, "could not add new plan fixated entry to storage")
	}

	return nil
}

// GetPlan gets a plan from the KVStore. It increases the plan's refCount by 1
func (k Keeper) GetPlan(ctx sdk.Context, index string) (val types.Plan, found bool) {
	var plan types.Plan
	err, _ := k.plansFs.GetEntry(ctx, index, &plan)
	if err != nil {
		return types.Plan{}, false
	}
	return plan, true
}

// FindPlan gets a plan from the KVStore. It does nothing to the plan's refCount
func (k Keeper) FindPlan(ctx sdk.Context, index string, block uint64) (val types.Plan, found bool) {
	var plan types.Plan
	err, _ := k.plansFs.FindEntry(ctx, index, block, &plan)
	if err != nil {
		return types.Plan{}, false
	}
	return plan, true
}

// PutPlan gets a plan from the KVStore. It decreases the plan's refCount by 1
func (k Keeper) PutPlan(ctx sdk.Context, index string, block uint64) bool {
	var plan types.Plan
	_, found := k.plansFs.PutEntry(ctx, index, block, &plan)
	return found
}

// GetAllPlanIndices gets from the KVStore all the plans' indices
func (k Keeper) GetAllPlanIndices(ctx sdk.Context) (val []string) {
	return k.plansFs.GetAllEntryIndices(ctx)
}

func (k Keeper) ValidateChainPolicies(ctx sdk.Context, policy projecttypes.Policy) error {
	return k.projectsKeeper.ValidateChainPolicies(ctx, policy)
}
