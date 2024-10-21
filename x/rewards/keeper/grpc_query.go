package keeper

import (
	"github.com/lavanet/lava/v4/x/rewards/types"
)

var _ types.QueryServer = Keeper{}
