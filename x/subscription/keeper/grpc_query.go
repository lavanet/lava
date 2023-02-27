package keeper

import (
	"github.com/lavanet/lava/x/subscription/types"
)

var _ types.QueryServer = Keeper{}
