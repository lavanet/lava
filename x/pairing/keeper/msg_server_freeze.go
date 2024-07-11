package keeper

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/utils"
	commontypes "github.com/lavanet/lava/utils/common/types"
	"github.com/lavanet/lava/x/pairing/types"
)

func (k msgServer) FreezeProvider(goCtx context.Context, msg *types.MsgFreezeProvider) (*types.MsgFreezeProviderResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	err := k.Keeper.FreezeProvider(ctx, msg.GetCreator(), msg.GetChainIds(), msg.Reason)

	return &types.MsgFreezeProviderResponse{}, err
}

func (k Keeper) FreezeProvider(ctx sdk.Context, provider string, chainIDs []string, reason string) error {
	if !utils.IsBech32Address(provider) {
		return utils.LavaFormatWarning("Freeze_get_provider_address", fmt.Errorf("invalid address"),
			utils.Attribute{Key: "providerAddress", Value: provider},
		)
	}

	if !commontypes.ValidateString(reason, commontypes.DESCRIPTION_RESTRICTIONS, nil) {
		return utils.LavaFormatWarning("Freeze_invalid_reason", fmt.Errorf("invalid string"),
			utils.LogAttr("reason", reason),
		)
	}

	for _, chainId := range chainIDs {
		stakeEntry, found := k.epochStorageKeeper.GetStakeEntryCurrent(ctx, chainId, provider)
		if !found {
			return utils.LavaFormatWarning("Freeze_cant_get_stake_entry", types.FreezeStakeEntryNotFoundError, []utils.Attribute{{Key: "chainID", Value: chainId}, {Key: "providerAddress", Value: provider}}...)
		}

		// freeze the provider by making the StakeAppliedBlock be max. This will remove the provider from the pairing list in the next epoch
		stakeEntry.Freeze()
		k.epochStorageKeeper.SetStakeEntryCurrent(ctx, stakeEntry)
	}

	utils.LogLavaEvent(ctx, ctx.Logger(), "freeze_provider", map[string]string{"providerAddress": provider, "chainIDs": strings.Join(chainIDs, ","), "freezeRequestBlock": strconv.FormatInt(ctx.BlockHeight(), 10), "freezeReason": reason}, "Provider Freeze")

	return nil
}
