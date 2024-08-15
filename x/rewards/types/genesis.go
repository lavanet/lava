package types

import (
	fmt "fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	commontypes "github.com/lavanet/lava/v2/utils/common/types"
	"github.com/lavanet/lava/v2/x/timerstore/types"
)

// this line is used by starport scaffolding # genesis/types/import

// DefaultIndex is the default capability global index
const DefaultIndex uint64 = 1

// DefaultGenesis returns the default Capability genesis state
func DefaultGenesis() *GenesisState {
	return &GenesisState{
		// this line is used by starport scaffolding # genesis/types/default
		Params:              DefaultParams(),
		RefillRewardsTS:     *types.DefaultGenesis(),
		BasePays:            []BasePayWithIndex{},
		IprpcSubscriptions:  []string{},
		MinIprpcCost:        sdk.NewCoin(commontypes.TokenDenom, sdk.ZeroInt()),
		IprpcRewards:        []IprpcReward{},
		IprpcRewardsCurrent: 0,
	}
}

// Validate performs basic genesis state validation returning an error upon any
// failure.
func (gs GenesisState) Validate() error {
	// this line is used by starport scaffolding # genesis/types/validate
	timeEntries := gs.RefillRewardsTS.GetTimeEntries()
	if len(timeEntries) > 1 {
		return fmt.Errorf(`there should be up to one timer in RefillRewardsTS
			at all times. amount of timers found: %v`, len(timeEntries))
	}

	unique := map[string]struct{}{}
	for _, sub := range gs.IprpcSubscriptions {
		_, err := sdk.AccAddressFromBech32(sub)
		if err != nil {
			return fmt.Errorf("invalid subscription address. err: %s", err.Error())
		}
		_, ok := unique[sub]
		if ok {
			return fmt.Errorf("iprpc subscription address is duplicated. address: %s", sub)
		}
		unique[sub] = struct{}{}
	}

	if gs.MinIprpcCost.Denom != commontypes.TokenDenom {
		return fmt.Errorf("invalid min iprpc cost denom. MinIprpcCost: %s", gs.MinIprpcCost.String())
	}

	if gs.MinIprpcCost.Amount.IsNegative() {
		return fmt.Errorf("negative min iprpc cost. MinIprpcCost: %s", gs.MinIprpcCost.String())
	}

	for _, iprpcReward := range gs.IprpcRewards {
		for _, specFund := range iprpcReward.SpecFunds {
			for _, coin := range specFund.Fund {
				if !coin.IsValid() {
					return fmt.Errorf("invalid iprpc reward fund. invalid coin: %s", coin.String())
				}
			}
		}
	}

	return gs.Params.Validate()
}
