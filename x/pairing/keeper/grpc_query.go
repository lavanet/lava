package keeper

import (
	"github.com/lavanet/lava/x/pairing/types"
)

var _ types.QueryServer = Keeper{}
