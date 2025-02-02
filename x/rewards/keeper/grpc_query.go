package keeper

import (
	"github.com/lavanet/lava/v5/x/rewards/types"
)

var _ types.QueryServer = Keeper{}
