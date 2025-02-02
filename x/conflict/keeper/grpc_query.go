package keeper

import (
	"github.com/lavanet/lava/v5/x/conflict/types"
)

var _ types.QueryServer = Keeper{}
