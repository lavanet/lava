package keeper

import (
	"github.com/lavanet/lava/v2/x/dualstaking/types"
)

var _ types.QueryServer = Keeper{}
