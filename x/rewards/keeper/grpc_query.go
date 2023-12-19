package keeper

import (
	"github.com/lavanet/lava/x/rewards/types"
)

var _ types.QueryServer = Keeper{}
