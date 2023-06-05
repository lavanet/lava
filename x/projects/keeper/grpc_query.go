package keeper

import (
	"github.com/lavanet/lava/x/projects/types"
)

var _ types.QueryServer = Keeper{}
