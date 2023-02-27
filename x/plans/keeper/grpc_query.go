package keeper

import (
	"github.com/lavanet/lava/x/plans/types"
)

var _ types.QueryServer = Keeper{}
