package keeper

import (
	"github.com/lavanet/lava/v4/x/pairing/types"
)

var _ types.QueryServer = Keeper{}
