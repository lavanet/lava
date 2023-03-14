package keeper

import (
	"context"
	"math"
	"strconv"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/utils"
	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
	"github.com/lavanet/lava/x/pairing/types"
)

func (k msgServer) FreezeProvider(goCtx context.Context, msg *types.MsgFreezeProvider) (*types.MsgFreezeProviderResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	providerAddr, err := sdk.AccAddressFromBech32(msg.GetCreator())
	if err != nil {
		return nil, utils.LavaFormatError("Freeze_get_provider_address", err, &map[string]string{"providerAddress": msg.GetCreator()})
	}

	for _, chainId := range msg.GetChainIds() {
		stakeEntry, found, index := k.epochStorageKeeper.GetStakeEntryByAddressCurrent(ctx, epochstoragetypes.ProviderKey, chainId, providerAddr)
		if !found {
			return nil, utils.LavaFormatError("Freeze_cant_get_stake_entry", types.FreezeStakeEntryNotFoundError, &map[string]string{"chainID": chainId, "providerAddress": msg.GetCreator()})
		}

		// freeze the provider by making the StakeAppliedBlock be max. This will remove the provider from the pairing list in the next epoch
		stakeEntry.StakeAppliedBlock = math.MaxUint64
		k.epochStorageKeeper.ModifyStakeEntryCurrent(ctx, epochstoragetypes.ProviderKey, chainId, stakeEntry, index)
	}

	utils.LogLavaEvent(ctx, ctx.Logger(), "freeze_provider", map[string]string{"providerAddress": msg.GetCreator(), "chainIDs": strings.Join(msg.GetChainIds(), ","), "freezeRequestBlock": strconv.FormatInt(ctx.BlockHeight(), 10), "freezeReason": msg.GetReason()}, "Provider Freeze")

	return &types.MsgFreezeProviderResponse{}, nil
}
