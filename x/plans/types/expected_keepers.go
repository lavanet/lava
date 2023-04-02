package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/x/projects/types"
)

type ProjectsKeeper interface {
	ValidateChainPolicies(ctx sdk.Context, policy types.Policy) error
}
