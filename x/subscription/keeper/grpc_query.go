package keeper

import (
	"github.com/lavanet/lava/v2/x/subscription/types"
)

var _ types.QueryServer = Keeper{}
