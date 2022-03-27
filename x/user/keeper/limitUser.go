package keeper

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/lavanet/lava/x/user/types"
)

func (k Keeper) EnforceUserCUsUsageInSession(ctx sdk.Context, userStake *types.UserStake, totalCU uint64) error {
	//TODO: create param dictionary for max CU per session per stake and compare it to totalCU
	userStakeAmount := userStake.Stake.Amount.Uint64()
	// stakeToMaxCUMap := k.GetStakeToMaxCUInSessionMap(ctx)
	var allowedCU uint64 = 0
	stakeToMaxCUMap := map[uint64]uint64{0: 5000, 500: 20000, 1000: 50000, 999999: 1000000000}
	for stakeStep, allowedCUPerSession := range stakeToMaxCUMap {
		if userStakeAmount >= stakeStep {
			allowedCU = allowedCUPerSession
		} else {
			break
		}
	}
	if totalCU > allowedCU {
		k.LimitUserPairingsAndMarkForPenalty(ctx, userStake)
		return fmt.Errorf("user %s bypassed allowed CU %d by using: %s", userStake, allowedCU, totalCU)
	}
	return nil
}

func (k Keeper) LimitUserPairingsAndMarkForPenalty(ctx sdk.Context, userStake *types.UserStake) {
	//TODO: jail user, and count problems
}
