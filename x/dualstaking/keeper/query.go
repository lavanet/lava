package keeper

import (
	"github.com/lavanet/lava/v4/x/dualstaking/types"
)

var _ types.QueryServer = Keeper{}
