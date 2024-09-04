package keeper

import (
	"github.com/lavanet/lava/v3/x/rewards/types"
)

var _ types.QueryServer = Keeper{}
