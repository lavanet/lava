package keeper

import (
	"github.com/lavanet/lava/v3/x/conflict/types"
)

var _ types.QueryServer = Keeper{}
