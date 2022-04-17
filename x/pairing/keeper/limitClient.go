package keeper

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
)

func (k Keeper) EnforceClientCUsUsageInEpoch(ctx sdk.Context, clientEntry *epochstoragetypes.StakeEntry, totalCU uint64) error {
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
		if clientEntry.Stake.IsGTE(stakeToCU.stake) {
			allowedCU = stakeToCU.cu
		} else {
			break
		}
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
