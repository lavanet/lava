package keeper

import (
	"context"
	"strconv"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/utils"
	"github.com/lavanet/lava/x/projects/types"
)

func (k msgServer) SetProjectPolicy(goCtx context.Context, msg *types.MsgSetProjectPolicy) (*types.MsgSetProjectPolicyResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	projectID := msg.Project
	adminKey := msg.Creator
	policy := msg.Policy

	err := k.validateChainPolicies(ctx, projectID, *policy)
	if err != nil {
		return nil, err
	}

	project, err := k.GetProjectForBlock(ctx, projectID, uint64(ctx.BlockHeight()))
	if err != nil {
		return nil, utils.LavaError(ctx, ctx.Logger(), "SetProjectPolicy_project_not_found", map[string]string{"project": projectID}, "project id not found")
	}

	// check if the admin key is valid
	if !project.IsAdminKey(adminKey) {
		return nil, utils.LavaError(ctx, ctx.Logger(), "SetProjectPolicy_not_admin", map[string]string{"project": projectID}, "the requesting key is not admin key")
	}

	project.Policy = *policy

	if project.UsedCu > project.Policy.TotalCuLimit {
		project.Policy.TotalCuLimit = project.UsedCu
	}

	nextEpoch, err := k.epochStorageKeeper.GetNextEpoch(ctx, uint64(ctx.BlockHeight()))
	if err != nil {
		return nil, utils.LavaError(ctx, k.Logger(ctx), "SetProjectPolicy_cant_get_next_epoch", map[string]string{"block": strconv.FormatUint(uint64(ctx.BlockHeight()), 10)}, "can't get next epoch")
	}
	err = k.projectsFS.AppendEntry(ctx, projectID, nextEpoch, &project)

	if err != nil {
		return nil, err
	}

	return &types.MsgSetProjectPolicyResponse{}, nil
}

func (k msgServer) validateChainPolicies(ctx sdk.Context, projectId string, policy types.Policy) error {
	// validate chainPolicies
	for _, chainPolicy := range policy.GetChainPolicies() {
		// get spec and make sure it's enabled
		spec, found := k.specKeeper.GetSpec(ctx, chainPolicy.GetChainId())
		if !found {
			return utils.LavaError(ctx, k.Logger(ctx), "validateChainPolicies_spec_not_found", map[string]string{"specIndex": spec.GetIndex()}, "policy's spec not found")
		}
		if !spec.GetEnabled() {
			return utils.LavaError(ctx, k.Logger(ctx), "validateChainPolicies_spec_not_enabled", map[string]string{"specIndex": spec.GetIndex()}, "policy's spec not enabled")
		}

		// go over the chain policy's APIs and make sure that they are part of the spec
		for _, policyApi := range chainPolicy.GetApis() {
			foundApi := false
			for _, api := range spec.GetApis() {
				if api.GetName() == policyApi {
					foundApi = true
				}
			}
			if !foundApi {
				details := map[string]string{
					"specIndex": spec.GetIndex(),
					"API":       policyApi,
				}
				return utils.LavaError(ctx, k.Logger(ctx), "validateChainPolicies_chain_policy_api_not_found", details, "policy's spec's API not found")
			}
		}
	}

	return nil
}
