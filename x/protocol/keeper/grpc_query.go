package keeper

import (
	"github.com/lavanet/lava/v2/x/protocol/types"
)

var _ types.QueryServer = Keeper{}
