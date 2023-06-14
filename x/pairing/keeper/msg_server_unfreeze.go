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

	providerAddr, err := sdk.AccAddressFromBech32(msg.GetCreator())
	if err != nil {
		return nil, utils.LavaFormatError("Unfreeze_get_provider_address", err, utils.Attribute{Key: "providerAddress", Value: msg.GetCreator()})
	}
	current_block := uint64(ctx.BlockHeight())
	unfrozen_chains := []string{}
	for _, chainId := range msg.GetChainIds() {
		stakeEntry, found, index := k.epochStorageKeeper.GetStakeEntryByAddressCurrent(ctx, chainId, providerAddr)
		if !found {
			return nil, utils.LavaFormatError("Unfreeze_cant_get_stake_entry", types.FreezeStakeEntryNotFoundError, []utils.Attribute{{Key: "chainID", Value: chainId}, {Key: "providerAddress", Value: msg.GetCreator()}}...)
		}

		if stakeEntry.StakeAppliedBlock > current_block {
			// unfreeze the provider by making the StakeAppliedBlock the current block. This will let the provider be added to the pairing list in the next epoch, when current entries becomes the front of epochStorage
			stakeEntry.StakeAppliedBlock = current_block
			k.epochStorageKeeper.ModifyStakeEntryCurrent(ctx, chainId, stakeEntry, index)
			unfrozen_chains = append(unfrozen_chains, chainId)
		}
		// else case does not throw an error because we don't want to fail unfreezing other chains
	}
	utils.LogLavaEvent(ctx, ctx.Logger(), "unfreeze_provider", map[string]string{"providerAddress": msg.GetCreator(), "chainIDs": strings.Join(unfrozen_chains, ",")}, "Provider Unfreeze")
	return &types.MsgUnfreezeProviderResponse{}, nil
}
