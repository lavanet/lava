package keeper

import (
	"github.com/lavanet/lava/v4/x/spec/types"
)

var _ types.QueryServer = Keeper{}
