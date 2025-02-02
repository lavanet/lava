package keeper

import (
	"github.com/lavanet/lava/v5/x/subscription/types"
)

var _ types.QueryServer = Keeper{}
