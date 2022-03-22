package keeper

import (
	"github.com/lavanet/lava/x/user/types"
)

var _ types.QueryServer = Keeper{}
