package keeper

import (
	"github.com/lavanet/lava/v4/x/epochstorage/types"
)

var _ types.QueryServer = Keeper{}
