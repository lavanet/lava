package keeper

import (
	"github.com/lavanet/lava/v3/x/pairing/types"
)

var _ types.QueryServer = Keeper{}
