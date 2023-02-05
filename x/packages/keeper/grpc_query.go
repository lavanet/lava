package keeper

import (
	"github.com/lavanet/lava/x/packages/types"
)

var _ types.QueryServer = Keeper{}
