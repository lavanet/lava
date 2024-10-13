package keeper

import (
	_ "embed"

	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/v3/x/dualstaking/types"
	fixationtypes "github.com/lavanet/lava/v3/x/fixationstore/types"
	timerstoretypes "github.com/lavanet/lava/v3/x/timerstore/types"
)

type Migrator struct {
	keeper Keeper
}

func NewMigrator(keeper Keeper) Migrator {
	return Migrator{keeper: keeper}
}

func (m Migrator) MigrateVersion5To6(ctx sdk.Context) error {
	nextEpoch := m.keeper.epochstorageKeeper.GetCurrentNextEpoch(ctx)

	// prefix for the delegations fixation store
	const (
		DelegationPrefix         = "delegation-fs"
		DelegatorPrefix          = "delegator-fs"
		UnbondingPrefix          = "unbonding-ts"
		DelegatorRewardKeyPrefix = "DelegatorReward/value/"
	)

	// set delegations
	ts := timerstoretypes.NewTimerStore(m.keeper.storeKey, m.keeper.cdc, DelegationPrefix)
	delegationFS := fixationtypes.NewFixationStore(m.keeper.storeKey, m.keeper.cdc, DelegationPrefix, ts, nil)
	incisec := delegationFS.GetAllEntryIndices(ctx)
	for _, index := range incisec {
		var oldDelegation types.Delegation
		found := delegationFS.FindEntry(ctx, index, nextEpoch, &oldDelegation)
		if found {
			delegation, found := m.keeper.GetDelegation(ctx, oldDelegation.Provider, oldDelegation.Delegator)
			if found {
				delegation.Amount = delegation.Amount.Add(oldDelegation.Amount)
			} else {
				delegation = oldDelegation
			}
			delegation.Timestamp = ctx.BlockTime().UTC().Unix()
			m.keeper.SetDelegation(ctx, delegation)
		}
	}

	// set rewards
	store := prefix.NewStore(ctx.KVStore(m.keeper.storeKey), types.KeyPrefix(DelegatorRewardKeyPrefix))
	iterator := sdk.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.DelegatorReward
		m.keeper.cdc.MustUnmarshal(iterator.Value(), &val)
		reward, found := m.keeper.GetDelegatorReward(ctx, val.Delegator, val.Delegator)
		if found {
			reward.Amount = reward.Amount.Add(val.Amount...)
		} else {
			reward = val
		}
		m.keeper.SetDelegatorReward(ctx, reward)
	}

	// now delete the stores
	deleteStore := func(prefixString string) {
		store := prefix.NewStore(ctx.KVStore(m.keeper.storeKey), types.KeyPrefix(prefixString))
		iterator := sdk.KVStorePrefixIterator(store, []byte{})

		defer iterator.Close()

		for ; iterator.Valid(); iterator.Next() {
			store.Delete(iterator.Key())
		}
	}

	deleteStore(DelegationPrefix)
	deleteStore(DelegatorPrefix)
	deleteStore(UnbondingPrefix)
	deleteStore(DelegatorRewardKeyPrefix)

	return nil
}
