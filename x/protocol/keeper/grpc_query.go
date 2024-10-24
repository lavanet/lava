package keeper

import (
	"github.com/lavanet/lava/v4/x/protocol/types"
)

var _ types.QueryServer = Keeper{}
