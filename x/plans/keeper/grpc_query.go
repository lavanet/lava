package keeper

import (
	"github.com/lavanet/lava/v2/x/plans/types"
)

var _ types.QueryServer = Keeper{}
