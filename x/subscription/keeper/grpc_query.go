package keeper

import (
	"github.com/lavanet/lava/v4/x/subscription/types"
)

var _ types.QueryServer = Keeper{}
