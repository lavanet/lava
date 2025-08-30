package keeper

import (
	"github.com/lavanet/lava/v5/x/dualstaking/types"
)

var _ types.QueryServer = Keeper{}
