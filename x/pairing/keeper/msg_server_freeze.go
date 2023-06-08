package keeper

import (
	"context"
	"strconv"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/utils"
	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
	"github.com/lavanet/lava/x/pairing/types"
)

func (k msgServer) FreezeProvider(goCtx context.Context, msg *types.MsgFreezeProvider) (*types.MsgFreezeProviderResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	err := k.Keeper.FreezeProvider(ctx, msg.GetCreator(), msg.GetChainIds(), msg.Reason)

	return &types.MsgFreezeProviderResponse{}, err
}

func (k Keeper) FreezeProvider(ctx sdk.Context, provider string, chainIDs []string, reason string) error {
	providerAddr, err := sdk.AccAddressFromBech32(provider)
	if err != nil {
		return utils.LavaFormatError("Freeze_get_provider_address", err, utils.Attribute{Key: "providerAddress", Value: provider})
	}

	for _, chainId := range chainIDs {
		stakeEntry, found, index := k.epochStorageKeeper.GetStakeEntryByAddressCurrent(ctx, epochstoragetypes.ProviderKey, chainId, providerAddr)
		if !found {
			return utils.LavaFormatError("Freeze_cant_get_stake_entry", types.FreezeStakeEntryNotFoundError, []utils.Attribute{{Key: "chainID", Value: chainId}, {Key: "providerAddress", Value: provider}}...)
		}

		// freeze the provider by making the StakeAppliedBlock be max. This will remove the provider from the pairing list in the next epoch
		stakeEntry.StakeAppliedBlock = types.FROZEN_BLOCK
		k.epochStorageKeeper.ModifyStakeEntryCurrent(ctx, epochstoragetypes.ProviderKey, chainId, stakeEntry, index)
	}

	utils.LogLavaEvent(ctx, ctx.Logger(), "freeze_provider", map[string]string{"providerAddress": providerAddr.String(), "chainIDs": strings.Join(chainIDs, ","), "freezeRequestBlock": strconv.FormatInt(ctx.BlockHeight(), 10), "freezeReason": reason}, "Provider Freeze")

	return nil
}
