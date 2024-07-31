package keeper

import (
	"github.com/lavanet/lava/v2/x/spec/types"
)

var _ types.QueryServer = Keeper{}
