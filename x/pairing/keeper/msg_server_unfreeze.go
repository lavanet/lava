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

func (k msgServer) UnfreezeProvider(goCtx context.Context, msg *types.MsgUnfreezeProvider) (*types.MsgUnfreezeProviderResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	providerAddr, err := sdk.AccAddressFromBech32(msg.GetCreator())
	if err != nil {
		return nil, utils.LavaFormatError("Unfreeze_get_provider_address", err, &map[string]string{"providerAddress": msg.GetCreator()})
	}

	for _, chainId := range msg.GetChainIds() {
		stakeEntry, found, index := k.epochStorageKeeper.GetStakeEntryByAddressCurrent(ctx, epochstoragetypes.ProviderKey, chainId, providerAddr)
		if !found {
			return nil, utils.LavaFormatError("Unfreeze_cant_get_stake_entry", types.FreezeStakeEntryNotFoundError, &map[string]string{"chainID": chainId, "providerAddress": msg.GetCreator()})
		}

		nextEpoch, err := k.epochStorageKeeper.GetNextEpoch(ctx, uint64(ctx.BlockHeight()))
		if err != nil {
			return nil, utils.LavaFormatError("Unfreeze_get_next_epoch", err, &map[string]string{"block": strconv.FormatInt(ctx.BlockHeight(), 10)})
		}

		// unfreeze the provider by making the StakeAppliedBlock be the next epoch. This will let the provider be added to the pairing list in the next epoch
		stakeEntry.StakeAppliedBlock = nextEpoch
		k.epochStorageKeeper.ModifyStakeEntryCurrent(ctx, epochstoragetypes.ProviderKey, chainId, stakeEntry, index)
	}

	utils.LogLavaEvent(ctx, ctx.Logger(), "unfreeze_provider", map[string]string{"providerAddress": msg.GetCreator(), "chainIDs": strings.Join(msg.GetChainIds(), ","), "unfreezeRequestBlock": strconv.FormatInt(ctx.BlockHeight(), 10)}, "Provider Unfreeze")

	return &types.MsgUnfreezeProviderResponse{}, nil
}
