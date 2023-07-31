package keeper

import (
	"github.com/lavanet/lava/x/protocol/types"
)

var _ types.QueryServer = Keeper{}
