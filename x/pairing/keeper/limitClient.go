package keeper

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
)

func (k Keeper) EnforceClientCUsUsageInEpoch(ctx sdk.Context, clientEntry *epochstoragetypes.StakeEntry, totalCU uint64) error {
	var allowedCU uint64 = 0
	stakeToMaxCUMap := k.StakeToMaxCUList(ctx).List

	for _, stakeToCU := range stakeToMaxCUMap {
		if stakeToCU.StakeThreshold <= clientEntry.Stake.Amount.Uint64() {
			allowedCU = stakeToCU.MaxComputeUnits
			break
		}
	}
	if allowedCU == 0 {
		return fmt.Errorf("user %s, MaxCU was not found for stake of: %d", clientEntry, clientEntry.Stake.Amount.Int64())
	}
	if totalCU > allowedCU {
		k.LimitClientPairingsAndMarkForPenalty(ctx, clientEntry)
		return fmt.Errorf("user %s bypassed allowed CU %d by using: %d", clientEntry, allowedCU, totalCU)
	}

	return nil
}

func (k Keeper) LimitClientPairingsAndMarkForPenalty(ctx sdk.Context, clientEntry *epochstoragetypes.StakeEntry) {
	//TODO: jail user, and count problems
}
