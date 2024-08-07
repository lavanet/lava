package keeper

import (
	"github.com/lavanet/lava/v2/x/epochstorage/types"
)

var _ types.QueryServer = Keeper{}
