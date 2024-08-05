package keeper

import (
	"github.com/lavanet/lava/v2/x/pairing/types"
)

var _ types.QueryServer = Keeper{}
