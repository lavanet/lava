package keeper

import (
	"github.com/lavanet/lava/v2/x/rewards/types"
)

var _ types.QueryServer = Keeper{}
