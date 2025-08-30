package keeper

import (
	"github.com/lavanet/lava/v5/x/projects/types"
)

var _ types.QueryServer = Keeper{}
