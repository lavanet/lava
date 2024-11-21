package keeper

import (
	"github.com/lavanet/lava/v4/x/conflict/types"
)

var _ types.QueryServer = Keeper{}
