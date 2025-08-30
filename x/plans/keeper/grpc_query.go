package keeper

import (
	"github.com/lavanet/lava/v5/x/plans/types"
)

var _ types.QueryServer = Keeper{}
