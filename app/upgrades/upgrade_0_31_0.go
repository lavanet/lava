package upgrades

import (
	"encoding/json"
	"time"

	"cosmossdk.io/math"
	store "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"
	"github.com/lavanet/lava/app/keepers"
)

type Upgrade_0_31_0_Info struct {
	ExpeditedMinDeposit   sdk.Coins
	ExpeditedThreshold    math.LegacyDec
	ExpeditedVotingPeriod time.Duration
}

func (u *Upgrade_0_31_0_Info) FromPlan(plan upgradetypes.Plan) error {
	info := plan.Info
	var intermediaryInfo = struct {
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

var Upgrade_0_31_0 = Upgrade{
	UpgradeName: "v0.31.0",
	CreateUpgradeHandler: func(mm *module.Manager, configurator module.Configurator, _ BaseAppParamManager, keepers *keepers.LavaKeepers) upgradetypes.UpgradeHandler {
		return func(ctx sdk.Context, plan upgradetypes.Plan, fromVM module.VersionMap) (module.VersionMap, error) {
			var info Upgrade_0_31_0_Info
			err := info.FromPlan(plan)
			if err != nil {
				return nil, err
			}
			// deposit
			params := keepers.GovKeeper.GetParams(ctx)
			params.ExpeditedMinDeposit = info.ExpeditedMinDeposit

			// tally
			params.ExpeditedThreshold = info.ExpeditedThreshold.String()

			// voting
			params.ExpeditedVotingPeriod = &info.ExpeditedVotingPeriod

			if err := keepers.GovKeeper.SetParams(ctx, params); err != nil {
				return nil, err
			}
			return mm.RunMigrations(ctx, configurator, fromVM)
		}
	},
	StoreUpgrades: store.StoreUpgrades{},
}
