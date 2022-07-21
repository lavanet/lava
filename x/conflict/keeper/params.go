package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/x/conflict/types"
)

// GetParams get all parameters as types.Params
func (k Keeper) GetParams(ctx sdk.Context) types.Params {
	return types.NewParams(
		k.MajorityPercent(ctx),
		k.VoteStartSpan(ctx),
		k.VotePeriod(ctx),
		k.WinnerRewardPercent(ctx),
		k.ClientRewardPercent(ctx),
		k.VotersRewardPercent(ctx),
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

func (k Keeper) WinnerRewardPercent(ctx sdk.Context) (res sdk.Dec) {
	k.paramstore.Get(ctx, types.KeyWinnerRewardPercent, &res)
	return
}

func (k Keeper) ClientRewardPercent(ctx sdk.Context) (res sdk.Dec) {
	k.paramstore.Get(ctx, types.KeyClientRewardPercent, &res)
	return
}

func (k Keeper) VotersRewardPercent(ctx sdk.Context) (res sdk.Dec) {
	k.paramstore.Get(ctx, types.KeyVotersRewardPercent, &res)
	return
}
