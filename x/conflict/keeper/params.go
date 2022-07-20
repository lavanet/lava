package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/x/conflict/types"
)

// GetParams get all parameters as types.Params
func (k Keeper) GetParams(ctx sdk.Context) types.Params {
	return types.NewParams(
		k.MajorityPrecent(ctx),
		k.VoteStartSpan(ctx),
		k.VotePeriod(ctx),
		k.WinnerRewardPrecent(ctx),
		k.ClientRewardPrecent(ctx),
		k.VotersRewardPrecent(ctx),
	)
}

// SetParams set the params
func (k Keeper) SetParams(ctx sdk.Context, params types.Params) {
	k.paramstore.SetParamSet(ctx, &params)
}

// MajorityPrecent returns the MajorityPrecent param
func (k Keeper) MajorityPrecent(ctx sdk.Context) (res sdk.Dec) {
	k.paramstore.Get(ctx, types.KeyMajorityPrecent, &res)
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

func (k Keeper) WinnerRewardPrecent(ctx sdk.Context) (res sdk.Dec) {
	k.paramstore.Get(ctx, types.KeyWinnerRewardPrecent, &res)
	return
}

func (k Keeper) ClientRewardPrecent(ctx sdk.Context) (res sdk.Dec) {
	k.paramstore.Get(ctx, types.KeyClientRewardPrecent, &res)
	return
}

func (k Keeper) VotersRewardPrecent(ctx sdk.Context) (res sdk.Dec) {
	k.paramstore.Get(ctx, types.KeyVotersRewardPrecent, &res)
	return
}
