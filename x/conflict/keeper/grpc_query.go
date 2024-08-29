package keeper

import (
	"github.com/lavanet/lava/v2/x/conflict/types"
)

var _ types.QueryServer = Keeper{}
