package upgrades

import (
	"fmt"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/cosmos/cosmos-sdk/x/bank/types"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"
	"github.com/lavanet/lava/v2/app/keepers"
	dualstakingtypes "github.com/lavanet/lava/v2/x/dualstaking/types"
	rewardstypes "github.com/lavanet/lava/v2/x/rewards/types"
)

func v0_32_0_UpgradeHandler(
	m *module.Manager,
	c module.Configurator,
	bapm BaseAppParamManager,
	lk *keepers.LavaKeepers,
) upgradetypes.UpgradeHandler {
	return func(ctx sdk.Context, plan upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
		// rewards pool
		totalSupply, err := lk.BankKeeper.TotalSupply(ctx, &types.QueryTotalSupplyRequest{})
		if err != nil {
			return nil, err
		}
		allocate := totalSupply.Supply.QuoInt(sdk.NewIntFromUint64(33))
		err = lk.BankKeeper.MintCoins(ctx, string(rewardstypes.ValidatorsRewardsAllocationPoolName), allocate)
		if err != nil {
			return nil, err
		}
		err = lk.BankKeeper.MintCoins(ctx, string(rewardstypes.ProvidersRewardsAllocationPool), allocate)
		if err != nil {
			return nil, err
		}

		rewards := lk.DualstakingKeeper.GetAllDelegatorReward(ctx)
		for _, reward := range rewards {
			lk.DualstakingKeeper.RemoveDelegatorReward(ctx, dualstakingtypes.DelegationKey(reward.Provider, reward.Delegator, reward.ChainId))
		}
		rewards = lk.DualstakingKeeper.GetAllDelegatorReward(ctx)
		if len(rewards) != 0 {
			return nil, fmt.Errorf("v0.32.0 UpgradeHandler: did not delete all rewards")
		}

		// expedited proposal
		// deposit
		params := lk.GovKeeper.GetParams(ctx)
		params.ExpeditedMinDeposit = append(params.ExpeditedMinDeposit, params.MinDeposit[0].AddAmount(sdk.NewIntFromUint64(1000)))

		// tally
		params.ExpeditedThreshold = "0.75"

		// voting
		seconds := params.VotingPeriod.Nanoseconds()
		duration := time.Duration(seconds/10) * time.Nanosecond
		params.ExpeditedVotingPeriod = &duration

		if err = params.ValidateBasic(); err != nil {
			return nil, err
		}

		if err := lk.GovKeeper.SetParams(ctx, params); err != nil {
			return nil, err
		}
		return m.RunMigrations(ctx, c, vm)
	}
}
