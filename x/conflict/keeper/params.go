package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/v2/x/conflict/types"
)

// GetParams get all parameters as types.Params
func (k Keeper) GetParams(ctx sdk.Context) types.Params {
	return types.NewParams(
		k.MajorityPercent(ctx),
		k.VoteStartSpan(ctx),
		k.VotePeriod(ctx),
		k.Rewards(ctx),
	)
}

// SetParams set the params
func (k Keeper) SetParams(ctx sdk.Context, params types.Params) {
	k.paramstore.SetParamSet(ctx, &params)
}

// MajorityPercent returns the MajorityPercent param
func (k Keeper) MajorityPercent(ctx sdk.Context) (res sdk.Dec) {
	k.paramstore.Get(ctx, types.KeyMajorityPercent, &res)
	return
}

func (k Keeper) VoteStartSpan(ctx sdk.Context) (res uint64) {
	k.paramstore.Get(ctx, types.KeyVoteStartSpan, &res)
	return
}

func (k Keeper) VotePeriod(ctx sdk.Context) (res uint64) {
	k.paramstore.Get(ctx, types.KeyVotePeriod, &res)
	return
}

func (k Keeper) Rewards(ctx sdk.Context) (res types.Rewards) {
	k.paramstore.Get(ctx, types.KeyRewards, &res)
	return
}
