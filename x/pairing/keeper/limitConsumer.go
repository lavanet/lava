package keeper

import (
	"fmt"
	"strconv"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/lavanet/lava/utils"
	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
)

func (k Keeper) GetAllowedCUForBlock(ctx sdk.Context, blockHeight uint64, entry *epochstoragetypes.StakeEntry) (uint64, error) {
	var allowedCU uint64 = 0
	stakeToMaxCUMap, err := k.StakeToMaxCUList(ctx, blockHeight)
	if err != nil {
		return 0, err
	}

	for _, stakeToCU := range stakeToMaxCUMap.List {
		if entry.Stake.IsGTE(stakeToCU.StakeThreshold) {
			allowedCU = stakeToCU.MaxComputeUnits
		} else {
			break
		}
	}
	return allowedCU, nil
}

func (k Keeper) EnforceClientCUsUsageInEpoch(ctx sdk.Context, chainID string, cuSum uint64, blockHeight int64, allowedCU uint64, clientAddr sdk.AccAddress, totalCUInEpochForUserProvider uint64, providerAddr sdk.AccAddress, epochStart uint64) (ammountToPay uint64, err error) {
	if allowedCU == 0 {
		return 0, fmt.Errorf("user %s, no allowedCU were found epoch: %d", clientAddr, epochStart)
	}
	servicersToPairCount, err := k.ServicersToPairCount(ctx, uint64(blockHeight))
	if err != nil {
		return 0, err
	}
	allowedCUConsumer := allowedCU / servicersToPairCount
	if totalCUInEpochForUserProvider > allowedCUConsumer {
		// if cu limit reached we return an error.
		return 0, utils.LavaFormatError("total cu in epoch for consumer exceeded the allowed amount", fmt.Errorf("consumer CU limit exceeded"), &map[string]string{"totalCUInEpochForUserProvider": strconv.FormatUint(totalCUInEpochForUserProvider, 10), "allowedCUProvider": strconv.FormatUint(allowedCUConsumer, 10)})
	}
	return cuSum, nil
}

func (k Keeper) ClientMaxCUProviderForBlock(ctx sdk.Context, blockHeight uint64, clientEntry *epochstoragetypes.StakeEntry) (uint64, error) {
	allowedCU, err := k.GetAllowedCUForBlock(ctx, blockHeight, clientEntry)
	if err != nil {
		return 0, fmt.Errorf("user %s, MaxCU was not found for stake of: %d", clientEntry, clientEntry.Stake.Amount.Int64())
	}
	servicersToPairCount, err := k.ServicersToPairCount(ctx, blockHeight)
	if err != nil {
		return 0, err
	}
	allowedCU /= servicersToPairCount

	return allowedCU, nil
}
