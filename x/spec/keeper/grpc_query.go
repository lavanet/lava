package keeper

import (
	"github.com/lavanet/lava/v3/x/spec/types"
)

var _ types.QueryServer = Keeper{}
