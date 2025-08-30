package keeper

import (
	"github.com/lavanet/lava/v5/x/pairing/types"
)

var _ types.QueryServer = Keeper{}
