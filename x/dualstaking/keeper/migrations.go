package keeper

import (
	_ "embed"

	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	dualstakingv4 "github.com/lavanet/lava/x/dualstaking/migrations/v4"
	dualstakingtypes "github.com/lavanet/lava/x/dualstaking/types"
	"github.com/lavanet/lava/x/pairing/types"
)

type Migrator struct {
	keeper Keeper
}

func NewMigrator(keeper Keeper) Migrator {
	return Migrator{keeper: keeper}
}

// MigrateVersion4To5 change the DelegatorReward object amount's field from sdk.Coin to sdk.Coins
func (m Migrator) MigrateVersion4To5(ctx sdk.Context) error {
	delegatorRewards := m.GetAllDelegatorRewardV4(ctx)
	for _, dr := range delegatorRewards {
		delegatorReward := dualstakingtypes.DelegatorReward{
			Delegator: dr.Delegator,
			Provider:  dr.Provider,
			ChainId:   dr.ChainId,
			Amount:    sdk.NewCoins(dr.Amount),
		}
		m.keeper.RemoveDelegatorReward(ctx, dualstakingtypes.DelegationKey(dr.Provider, dr.Delegator, dr.ChainId))
		m.keeper.SetDelegatorReward(ctx, delegatorReward)
	}

	params := dualstakingtypes.DefaultParams()
	m.keeper.SetParams(ctx, params)
	return nil
}

func (m Migrator) GetAllDelegatorRewardV4(ctx sdk.Context) (list []dualstakingv4.DelegatorRewardv4) {
	store := prefix.NewStore(ctx.KVStore(m.keeper.storeKey), types.KeyPrefix(dualstakingtypes.DelegatorRewardKeyPrefix))
	iterator := sdk.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val dualstakingv4.DelegatorRewardv4
		m.keeper.cdc.MustUnmarshal(iterator.Value(), &val)
		list = append(list, val)
	}

	return
}
