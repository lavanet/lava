package keeper

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/v2/utils"
	epochstoragetypes "github.com/lavanet/lava/v2/x/epochstorage/types"
	"github.com/lavanet/lava/v2/x/plans/types"
)

// AddPlan adds a new plan to the KVStore
func (k Keeper) AddPlan(ctx sdk.Context, planToAdd types.Plan, modify bool) error {
	err := k.ValidatePlanFields(ctx, &planToAdd)
	if err != nil {
		return err
	}

	if modify {
		var planFromStore types.Plan
		block, _, _, found := k.plansFS.FindEntryDetailed(ctx, planToAdd.GetIndex(), planToAdd.Block, &planFromStore)
		if found {
			if !planFromStore.Price.Amount.Equal(planToAdd.Price.Amount) {
				return utils.LavaFormatError("failed modifying plan in planFS", fmt.Errorf("plan price cannot be modified"),
					utils.Attribute{Key: "planToAdd", Value: planToAdd},
					utils.Attribute{Key: "originalPlan", Value: planFromStore},
				)
			}
			if planToAdd.Block != block {
				return utils.LavaFormatError("failed modifying plan in planFS", fmt.Errorf("plan to modify does not exist"),
					utils.Attribute{Key: "planToAdd", Value: planToAdd},
					utils.Attribute{Key: "originalPlan", Value: planFromStore},
					utils.Attribute{Key: "requestedBlock", Value: planToAdd.Block},
				)
			}
			k.plansFS.ModifyEntry(ctx, planToAdd.GetIndex(), planToAdd.Block, &planToAdd)
		} else {
			return utils.LavaFormatError("failed modifying plan in planFS", fmt.Errorf("plan to modify does not exist"),
				utils.Attribute{Key: "planToAdd", Value: planToAdd},
				utils.Attribute{Key: "originalPlan", Value: planFromStore},
				utils.Attribute{Key: "requestedBlock", Value: planToAdd.Block},
			)
		}
	} else {
		planToAdd.Block = uint64(ctx.BlockHeight())
		// TODO: verify the CU per epoch field
		err = k.plansFS.AppendEntry(ctx, planToAdd.GetIndex(), planToAdd.Block, &planToAdd)
		if err != nil {
			return utils.LavaFormatError("failed adding plan to planFS", err,
				utils.Attribute{Key: "planToAdd", Value: planToAdd},
			)
		}
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

func (k Keeper) ValidatePlanFields(ctx sdk.Context, planToAdd *types.Plan) error {
	if err := utils.ValidateCoins(ctx, k.stakingKeeper.BondDenom(ctx), planToAdd.Price, false); err != nil {
		return utils.LavaFormatError("plan price is invalid", err)
	}

	for _, chainPolicy := range planToAdd.PlanPolicy.ChainPolicies {
		specID := chainPolicy.ChainId
		if specID == types.WILDCARD_CHAIN_POLICY && len(chainPolicy.Apis) == 0 && len(chainPolicy.Requirements) == 0 {
			continue // this is allowed
		}
		expectedInterfaces, err := k.specKeeper.GetExpectedServicesForSpec(ctx, specID, false)
		if err != nil {
			return err
		}
		for _, requirement := range chainPolicy.Requirements {
			collection := requirement.Collection
			addon := collection.AddOn
			// an addon is the same as apiInterface for optional apiInterfaces
			if addon == collection.ApiInterface {
				addon = ""
			}
			if _, ok := expectedInterfaces[epochstoragetypes.EndpointService{
				ApiInterface: collection.ApiInterface,
				Addon:        addon,
				Extension:    "",
			}]; !ok {
				return fmt.Errorf("policy chain policy collection %#v was not found on spec %s", collection, specID)
			}

			for _, extension := range requirement.Extensions {
				if _, ok := expectedInterfaces[epochstoragetypes.EndpointService{
					ApiInterface: collection.ApiInterface,
					Addon:        addon,
					Extension:    extension,
				}]; !ok {
					return fmt.Errorf("policy chain policy requirement %#v extensions were not found on spec %s", requirement, specID)
				}
			}
		}
	}
	return nil
}
