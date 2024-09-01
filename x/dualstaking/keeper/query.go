package keeper

import (
	"github.com/lavanet/lava/v3/x/dualstaking/types"
)

var _ types.QueryServer = Keeper{}
