package keeper

import (
	"github.com/lavanet/lava/x/servicer/types"
)

var _ types.QueryServer = Keeper{}
