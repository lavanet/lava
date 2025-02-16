package keeper

import (
	"github.com/lavanet/lava/v5/x/protocol/types"
)

var _ types.QueryServer = Keeper{}
