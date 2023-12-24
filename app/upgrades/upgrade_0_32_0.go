package upgrades

import (
	"encoding/json"
	"fmt"
	"time"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/cosmos/cosmos-sdk/x/bank/types"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"
	"github.com/lavanet/lava/app/keepers"
	dualstakingtypes "github.com/lavanet/lava/x/dualstaking/types"
	rewardstypes "github.com/lavanet/lava/x/rewards/types"
)

type Upgrade_0_33_0_Info struct {
	ExpeditedMinDeposit   sdk.Coins
	ExpeditedThreshold    math.LegacyDec
	ExpeditedVotingPeriod time.Duration
}

func (u *Upgrade_0_33_0_Info) FromPlan(plan upgradetypes.Plan) error {
	info := plan.Info
	intermediaryInfo := struct {
		ExpeditedMinDeposit   string `json:"expedited_min_deposit"`
		ExpeditedThreshold    string `json:"expedited_threshold"`
		ExpeditedVotingPeriod string `json:"expedited_voting_period"`
	}{}
	err := json.Unmarshal([]byte(info), &intermediaryInfo)
	if err != nil {
		return err
	}

	u.ExpeditedMinDeposit, err = sdk.ParseCoinsNormalized(intermediaryInfo.ExpeditedMinDeposit)
	if err != nil {
		return err
	}
	u.ExpeditedThreshold, err = math.LegacyNewDecFromStr(intermediaryInfo.ExpeditedThreshold)
	if err != nil {
		return err
	}
	u.ExpeditedVotingPeriod, err = time.ParseDuration(intermediaryInfo.ExpeditedVotingPeriod)
	if err != nil {
		return err
	}
	return nil
}

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
		var info Upgrade_0_33_0_Info
		err = info.FromPlan(plan)
		if err != nil {
			return nil, err
		}
		// deposit
		params := lk.GovKeeper.GetParams(ctx)
		params.ExpeditedMinDeposit = info.ExpeditedMinDeposit

		// tally
		params.ExpeditedThreshold = info.ExpeditedThreshold.String()

		// voting
		params.ExpeditedVotingPeriod = &info.ExpeditedVotingPeriod

		if err = params.ValidateBasic(); err != nil {
			return nil, err
		}

		if err := lk.GovKeeper.SetParams(ctx, params); err != nil {
			return nil, err
		}
		return m.RunMigrations(ctx, c, vm)
	}
}
