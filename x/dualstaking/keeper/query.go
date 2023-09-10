package keeper

import (
	"github.com/lavanet/lava/x/dualstaking/types"
)

var _ types.QueryServer = Keeper{}
