package keeper

import (
	"github.com/lavanet/lava/v3/x/projects/types"
)

var _ types.QueryServer = Keeper{}
