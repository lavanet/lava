package keeper

import (
	"github.com/lavanet/lava/x/epochstorage/types"
)

var _ types.QueryServer = Keeper{}
