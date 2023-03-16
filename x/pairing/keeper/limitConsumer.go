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

func (k Keeper) EnforceClientCUsUsageInEpoch(ctx sdk.Context, cuSum uint64, allowedCU uint64, totalCUInEpochForUserProvider uint64, epochStart uint64) error {
	if totalCUInEpochForUserProvider > allowedCU {
		// if cu limit reached we return an error.
		return utils.LavaFormatError("total cu in epoch for consumer exceeded the allowed amount", fmt.Errorf("consumer CU limit exceeded"), &map[string]string{"totalCUInEpochForUserProvider": strconv.FormatUint(totalCUInEpochForUserProvider, 10), "allowedCUProvider": strconv.FormatUint(allowedCU, 10)})
	}
	return nil
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
