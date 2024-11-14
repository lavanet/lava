package keeper

import (
	"github.com/lavanet/lava/v4/x/plans/types"
)

var _ types.QueryServer = Keeper{}
