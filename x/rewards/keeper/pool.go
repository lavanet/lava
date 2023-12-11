package keeper

import (
	"fmt"
	"strconv"

	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/utils"
	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
	"github.com/lavanet/lava/x/rewards/types"
)

// TotalPoolTokens gets the total tokens supply from a pool
func (k Keeper) TotalPoolTokens(ctx sdk.Context, pool types.Pool) math.Int {
	poolAddr := k.accountKeeper.GetModuleAddress(string(pool))
	return k.bankKeeper.GetBalance(ctx, poolAddr, epochstoragetypes.TokenDenom).Amount
}

// BurnPoolTokens removes coins from a pool module account
func (k Keeper) BurnPoolTokens(ctx sdk.Context, pool types.Pool, amt math.Int) error {
	if !amt.IsPositive() {
		// skip as no coins need to be burned
		return nil
	}

	coins := sdk.NewCoins(sdk.NewCoin(epochstoragetypes.TokenDenom, amt))

	return k.bankKeeper.BurnCoins(ctx, string(pool), coins)
}

func (k Keeper) DistributeBlockReward(ctx sdk.Context) error {
	// get params for validator rewards calculation
	bondedTargetFactor := k.BondedTargetFactor(ctx)
	blocksToNextTimerExpiry := k.BlocksToNextTimerExpiry(ctx)

	// get validator block pool balance
	distributionPoolBalance := k.TotalPoolTokens(ctx, types.ValidatorsRewardsDistributionPoolName)

	// validators bonus rewards = (distributionPoolBalance * bondedTargetFactor) / blocksToNextTimerExpiry
	validatorsRewards := bondedTargetFactor.MulInt(distributionPoolBalance).TruncateInt().QuoRaw(blocksToNextTimerExpiry)
	if validatorsRewards.IsZero() {
		// no rewards is not necessarily an error -> print warning and return
		utils.LavaFormatWarning("validators block rewards is zero", fmt.Errorf(""),
			utils.Attribute{Key: "bonded_target_factor", Value: bondedTargetFactor.String()},
			utils.Attribute{Key: "distribution_pool_balance", Value: distributionPoolBalance.String()},
			utils.Attribute{Key: "blocks_to_next_timer_expiry", Value: strconv.FormatInt(blocksToNextTimerExpiry, 10)},
		)
		return nil
	}

	coins := sdk.NewCoins(sdk.NewCoin(epochstoragetypes.TokenDenom, validatorsRewards))

	// distribute rewards to validators (same as Cosmos mint module)
	err := k.AddCollectedFees(ctx, coins)
	if err != nil {
		return err
	}

	return nil
}
