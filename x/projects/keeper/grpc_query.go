package keeper

import (
	"github.com/lavanet/lava/v2/x/projects/types"
)

var _ types.QueryServer = Keeper{}
