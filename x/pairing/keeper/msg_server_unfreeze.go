package keeper

import (
	"context"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/utils"
	"github.com/lavanet/lava/x/pairing/types"
)

func (k msgServer) UnfreezeProvider(goCtx context.Context, msg *types.MsgUnfreezeProvider) (*types.MsgUnfreezeProviderResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	unfreezeBlock := k.epochStorageKeeper.GetCurrentNextEpoch(ctx) + 1
	unfrozen_chains := []string{}
	for _, chainId := range msg.GetChainIds() {
		stakeEntry, found := k.epochStorageKeeper.GetStakeEntryCurrent(ctx, chainId, msg.Creator)
		if !found {
			return nil, utils.LavaFormatWarning("Unfreeze_cant_get_stake_entry", types.FreezeStakeEntryNotFoundError, []utils.Attribute{{Key: "chainID", Value: chainId}, {Key: "providerAddress", Value: msg.GetCreator()}}...)
		}

		// the provider is not frozen (active or jailed), continue to other chainIDs
		if !stakeEntry.IsFrozen() {
			continue
		}

		minStake := k.Keeper.specKeeper.GetMinStake(ctx, chainId)
		if stakeEntry.EffectiveStake().LT(minStake.Amount) {
			return nil, utils.LavaFormatWarning("Unfreeze_insufficient_stake", types.UnFreezeInsufficientStakeError,
				[]utils.Attribute{
					{Key: "chainID", Value: chainId},
					{Key: "providerAddress", Value: msg.GetCreator()},
					{Key: "stake", Value: stakeEntry.Stake},
					{Key: "minStake", Value: minStake},
				}...)
		}

		if stakeEntry.IsJailed(ctx.BlockTime().UTC().Unix()) {
			return nil, utils.LavaFormatWarning("Unfreeze_jailed_provider", types.UnFreezeJailedStakeError,
				[]utils.Attribute{
					{Key: "chainID", Value: chainId},
					{Key: "providerAddress", Value: msg.GetCreator()},
					{Key: "jailEnd", Value: stakeEntry.JailEndTime},
				}...)
		}

		// unfreeze the provider by making the StakeAppliedBlock the current block. This will let the provider be added to the pairing list in the next epoch, when current entries becomes the front of epochStorage
		stakeEntry.UnFreeze(unfreezeBlock)
		stakeEntry.JailEndTime = 0
		stakeEntry.Jails = 0
		k.epochStorageKeeper.SetStakeEntryCurrent(ctx, stakeEntry)
		unfrozen_chains = append(unfrozen_chains, chainId)
	}
	utils.LogLavaEvent(ctx, ctx.Logger(), "unfreeze_provider", map[string]string{"providerAddress": msg.GetCreator(), "chainIDs": strings.Join(unfrozen_chains, ",")}, "Provider Unfreeze")
	return &types.MsgUnfreezeProviderResponse{}, nil
}
