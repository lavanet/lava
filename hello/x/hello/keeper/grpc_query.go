package keeper

import (
	"github.com/username/hello/x/hello/types"
)

var _ types.QueryServer = Keeper{}
