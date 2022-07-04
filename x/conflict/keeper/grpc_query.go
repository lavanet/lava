package keeper

import (
	"github.com/lavanet/lava/x/conflict/types"
)

var _ types.QueryServer = Keeper{}
