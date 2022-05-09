package keeper

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
)

func (k Keeper) EnforceClientCUsUsageInEpoch(ctx sdk.Context, clientEntry *epochstoragetypes.StakeEntry, totalCUInEpochForUserProvider uint64) error {
	var allowedCU uint64 = 0
	stakeToMaxCUMap := k.StakeToMaxCUList(ctx).List

	for _, stakeToCU := range stakeToMaxCUMap {
		if stakeToCU.StakeThreshold.IsGTE(clientEntry.Stake) {
			allowedCU = stakeToCU.MaxComputeUnits
		} else {
			break
		}
	}

	if allowedCU == 0 {
		panic(fmt.Sprintf("user %s, MaxCU was not found for stake of: %d", clientEntry, clientEntry.Stake.Amount.Int64()))
	}
	allowedCU = allowedCU / k.ServicersToPairCount(ctx)
	if totalCUInEpochForUserProvider > allowedCU {
		k.LimitClientPairingsAndMarkForPenalty(ctx, clientEntry)
		return fmt.Errorf("user %s bypassed allowed CU %d by using: %d", clientEntry, allowedCU, totalCUInEpochForUserProvider)
	}

	return nil
}

func (k Keeper) LimitClientPairingsAndMarkForPenalty(ctx sdk.Context, clientEntry *epochstoragetypes.StakeEntry) {
	//TODO: jail user, and count problems
}
