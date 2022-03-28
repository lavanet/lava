package keeper

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/lavanet/lava/x/user/types"
)

func (k Keeper) EnforceUserCUsUsageInSession(ctx sdk.Context, userStake *types.UserStake, totalCU uint64) error {
	var allowedCU uint64 = 0
	type stakeToCU struct {
		stake sdk.Coin
		cu    uint64
	}
	//TODO: create param dictionary for max CU per session per stake and compare it to totalCU
	// stakeToMaxCUMap := k.GetStakeToMaxCUInSessionMap(ctx)
	stakeToMaxCUMap := []stakeToCU{}
	stakeToMaxCUMap = append(stakeToMaxCUMap, stakeToCU{stake: sdk.Coin{Denom: "stake", Amount: sdk.NewIntFromUint64(0)}, cu: 5000})
	stakeToMaxCUMap = append(stakeToMaxCUMap, stakeToCU{stake: sdk.Coin{Denom: "stake", Amount: sdk.NewIntFromUint64(500)}, cu: 15000})
	stakeToMaxCUMap = append(stakeToMaxCUMap, stakeToCU{stake: sdk.Coin{Denom: "stake", Amount: sdk.NewIntFromUint64(2000)}, cu: 50000})
	stakeToMaxCUMap = append(stakeToMaxCUMap, stakeToCU{stake: sdk.Coin{Denom: "stake", Amount: sdk.NewIntFromUint64(5000)}, cu: 250000})
	stakeToMaxCUMap = append(stakeToMaxCUMap, stakeToCU{stake: sdk.Coin{Denom: "stake", Amount: sdk.NewIntFromUint64(100000)}, cu: 500000})
	stakeToMaxCUMap = append(stakeToMaxCUMap, stakeToCU{stake: sdk.Coin{Denom: "stake", Amount: sdk.NewIntFromUint64(9999900000)}, cu: 9999999999})
	for _, stakeToCU := range stakeToMaxCUMap {
		if userStake.Stake.IsGTE(stakeToCU.stake) {
			allowedCU = stakeToCU.cu
		} else {
			break
		}
	}
	if totalCU > allowedCU {
		k.LimitUserPairingsAndMarkForPenalty(ctx, userStake)
		return fmt.Errorf("user %s bypassed allowed CU %d by using: %d", userStake, allowedCU, totalCU)
	}
	return nil
}

func (k Keeper) LimitUserPairingsAndMarkForPenalty(ctx sdk.Context, userStake *types.UserStake) {
	//TODO: jail user, and count problems
}
