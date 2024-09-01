package keeper

import (
	"github.com/lavanet/lava/v3/x/subscription/types"
)

var _ types.QueryServer = Keeper{}
