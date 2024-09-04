package keeper

import (
	"github.com/lavanet/lava/v3/x/epochstorage/types"
)

var _ types.QueryServer = Keeper{}
