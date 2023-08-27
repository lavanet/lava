package keeper

import (
	"math"

	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	v3 "github.com/lavanet/lava/x/epochstorage/types/migrations/v3"
	v4 "github.com/lavanet/lava/x/epochstorage/types/migrations/v4"
)

type Migrator struct {
	keeper Keeper
}

func NewMigrator(keeper Keeper) Migrator {
	return Migrator{keeper: keeper}
}

// Migrate2to3 implements store migration from v2 to v3:
// - refund all clients stake
// - migrate providers to a new key
func (m Migrator) Migrate2to3(ctx sdk.Context) error {
	const ClientKey = "client"
	const ProviderKey = "provider"

	storage := m.keeper.GetAllStakeStorage(ctx)
	for _, storage := range storage {
		// handle client keys
		if storage.Index[:len(ClientKey)] == ClientKey {
			m.keeper.RemoveStakeStorage(ctx, storage.Index)
		} else if storage.Index[:len(ProviderKey)] == ProviderKey { // handle provider keys
			if len(storage.Index) > len(ProviderKey) {
				storage.Index = storage.Index[len(ProviderKey):]
				m.keeper.SetStakeStorage(ctx, storage)
			}
		}
	}
	return nil
}

func (m Migrator) Migrate3to4(ctx sdk.Context) error {
	storeKey := sdk.NewKVStoreKey(v3.StoreKey)

	store := prefix.NewStore(ctx.KVStore(storeKey), v3.KeyPrefix(v3.StakeStorageKeyPrefix))
	iterator := sdk.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var stakeStorageV3 v3.StakeStorage
		m.keeper.cdc.MustUnmarshal(iterator.Value(), &stakeStorageV3)

		stakeStorageV4 := v4.StakeStorage{
			Index:          stakeStorageV3.Index,
			EpochBlockHash: stakeStorageV3.EpochBlockHash,
		}

		var stakeEntriesV4 []v4.StakeEntry
		for _, stakeEntryV3 := range stakeStorageV3.StakeEntries {
			stakeEntryV4 := v4.StakeEntry{
				Stake:             stakeEntryV3.Stake,
				Address:           stakeEntryV3.Address,
				StakeAppliedBlock: stakeEntryV3.StakeAppliedBlock,
				Chain:             stakeEntryV3.Chain,
				Moniker:           stakeEntryV3.Moniker,
			}

			var geoInt32 int32
			if stakeEntryV3.Geolocation <= math.MaxInt32 {
				geoInt32 = int32(stakeEntryV3.Geolocation)
			} else {
				geoInt32 = math.MaxInt32
			}

			stakeEntryV4.Geolocation = geoInt32

			var endpointsV4 []v4.Endpoint
			for _, endpointV3 := range stakeEntryV3.Endpoints {
				endpointV4 := v4.Endpoint{
					IPPORT:        endpointV3.IPPORT,
					Addons:        endpointV3.Addons,
					ApiInterfaces: endpointV3.ApiInterfaces,
					Extensions:    endpointV3.Extensions,
				}

				var geoEndpInt32 int32
				if stakeEntryV3.Geolocation <= math.MaxInt32 {
					geoEndpInt32 = int32(stakeEntryV3.Geolocation)
				} else {
					geoEndpInt32 = math.MaxInt32
				}

				endpointV4.Geolocation = geoEndpInt32
				endpointsV4 = append(endpointsV4, endpointV4)
			}

			stakeEntryV4.Endpoints = endpointsV4
			stakeEntriesV4 = append(stakeEntriesV4, stakeEntryV4)
		}
		stakeStorageV4.StakeEntries = stakeEntriesV4

		store.Delete(iterator.Key())
		store.Set(iterator.Key(), m.keeper.cdc.MustMarshal(&stakeStorageV4))
	}

	return nil
}
