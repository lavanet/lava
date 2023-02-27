package keeper

import (
	"strconv"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/utils"
	"github.com/lavanet/lava/x/projects/types"
)

// add a default project to a subscription
func (k Keeper) GetProjectForBlock(ctx sdk.Context, projectID string, BlockHeight uint64) (types.Project, error) {

	var project types.Project

	err := k.projectsFS.FindEntry(ctx, projectID, BlockHeight, &project)
	if err != nil {
		return project, utils.LavaError(ctx, ctx.Logger(), "GetProjectForBlock_not_found", map[string]string{"project": projectID, "blockHeight": strconv.FormatUint(BlockHeight, 10)}, "default project already exist for the current subscription")
	}

	return project, nil
}
