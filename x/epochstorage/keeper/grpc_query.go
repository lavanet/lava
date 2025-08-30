package keeper

import (
	"github.com/lavanet/lava/v5/x/epochstorage/types"
)

var _ types.QueryServer = Keeper{}
