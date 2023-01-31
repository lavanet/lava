package keeper

import (
	"github.com/lavanet/lava/x/packagemanager/types"
)

var _ types.QueryServer = Keeper{}
