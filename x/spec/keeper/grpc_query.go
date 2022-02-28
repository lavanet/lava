package keeper

import (
	"github.com/lavanet/lava/x/spec/types"
)

var _ types.QueryServer = Keeper{}
