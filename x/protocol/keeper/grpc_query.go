package keeper

import (
	"github.com/lavanet/lava/v3/x/protocol/types"
)

var _ types.QueryServer = Keeper{}
