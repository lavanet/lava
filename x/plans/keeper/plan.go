package keeper

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	commonTypes "github.com/lavanet/lava/common/types"
	"github.com/lavanet/lava/utils"
	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
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

	err = k.ValidatePlanFields(ctx, &planToAdd)
	if err != nil {
		return err
	}
	return nil
}

// DelPlan deletes a plan, so it is not visible/gettable for new subscriptions
// (however, existing referenced versions remain intact until not used anymore)
func (k Keeper) DelPlan(ctx sdk.Context, index string) error {
	// Deletions should take place at the end of epoch (beginning of next epoch).
	nextEpoch, err := k.epochstorageKeeper.GetNextEpoch(ctx, uint64(ctx.BlockHeight()))
	if err != nil {
		return utils.LavaFormatError("DelPlan: failed to get NextEpoch", err,
			utils.Attribute{Key: "index", Value: index},
		)
	}

	return k.plansFS.DelEntry(ctx, index, nextEpoch)
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

// Export all plans from the KVStore
func (k Keeper) ExportPlans(ctx sdk.Context) []commonTypes.RawMessage {
	return k.plansFS.Export(ctx)
}

// Init all plans in the KVStore
func (k Keeper) InitPlans(ctx sdk.Context, data []commonTypes.RawMessage) {
	k.plansFS.Init(ctx, data)
}

func (k Keeper) ValidatePlanFields(ctx sdk.Context, planToAdd *types.Plan) error {
	for _, chainPolicy := range planToAdd.PlanPolicy.ChainPolicies {
		specID := chainPolicy.ChainId
		if specID == types.WILDCARD_CHAIN_POLICY && len(chainPolicy.Apis) == 0 && len(chainPolicy.Collections) == 0 {
			continue // this is allowed
		}
		expectedInterfaces, err := k.specKeeper.GetExpectedInterfacesForSpec(ctx, specID, false)
		if err != nil {
			return err
		}
		for _, collection := range chainPolicy.Collections {
			addon := collection.AddOn
			if addon == collection.ApiInterface {
				addon = ""
			}
			if _, ok := expectedInterfaces[epochstoragetypes.EndpointService{
				ApiInterface: collection.ApiInterface,
				Addon:        addon,
			}]; !ok {
				return fmt.Errorf("policy chain policy collection %#v was not found on spec %s", collection, specID)
			}
		}
	}
	return nil
}
