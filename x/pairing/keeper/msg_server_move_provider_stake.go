package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/v3/utils"
	"github.com/lavanet/lava/v3/x/pairing/types"
)

func (k msgServer) MoveProviderStake(goCtx context.Context, msg *types.MsgMoveProviderStake) (*types.MsgMoveProviderStakeResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	err := k.Keeper.MoveProviderStake(ctx, msg.Creator, msg.SrcChain, msg.DstChain, msg.Amount)
	return &types.MsgMoveProviderStakeResponse{}, err
}

func (k Keeper) MoveProviderStake(ctx sdk.Context, creator, srcChain, dstChain string, amount sdk.Coin) error {
	srcEntry, found := k.epochStorageKeeper.GetStakeEntryCurrent(ctx, srcChain, creator)
	if !found {
		return utils.LavaFormatError("provider is not staked in the source chain", nil,
			utils.LogAttr("provider", creator),
			utils.LogAttr("chain", srcChain),
		)
	}

	dstEntry, found := k.epochStorageKeeper.GetStakeEntryCurrent(ctx, dstChain, creator)
	if !found {
		return utils.LavaFormatError("provider is not staked in the destination chain", nil,
			utils.LogAttr("provider", creator),
			utils.LogAttr("chain", srcChain),
		)
	}

	srcEntry.Stake = srcEntry.Stake.Sub(amount)
	dstEntry.Stake = dstEntry.Stake.Add(amount)

	minSelfDelegation := k.dualstakingKeeper.MinSelfDelegation(ctx)
	if srcEntry.Stake.IsLT(minSelfDelegation) {
		return utils.LavaFormatError("provider will be below min stake on source chain", nil,
			utils.LogAttr("provider", creator),
			utils.LogAttr("chain", srcChain),
		)
	}

	k.epochStorageKeeper.SetStakeEntryCurrent(ctx, srcEntry)
	k.epochStorageKeeper.SetStakeEntryCurrent(ctx, dstEntry)
	return k.dualstakingKeeper.AfterDelegationModified(ctx, creator, creator, sdk.NewCoin(k.stakingKeeper.BondDenom(ctx), sdk.ZeroInt()), false, true)
}
