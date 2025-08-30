package keeper

import (
	"github.com/lavanet/lava/v5/x/spec/types"
)

var _ types.QueryServer = Keeper{}
