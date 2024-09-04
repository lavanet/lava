package keeper

import (
	"github.com/lavanet/lava/v3/x/plans/types"
)

var _ types.QueryServer = Keeper{}
