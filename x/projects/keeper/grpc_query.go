package keeper

import (
	"github.com/lavanet/lava/v4/x/projects/types"
)

var _ types.QueryServer = Keeper{}
