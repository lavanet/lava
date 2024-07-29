package keeper

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"unicode"

	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/utils"
	"github.com/lavanet/lava/x/epochstorage/types"
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
	// const ClientKey = "client"
	// const ProviderKey = "provider"

	// storage := m.keeper.GetAllStakeStorage(ctx)
	// for _, storage := range storage {
	// 	// handle client keys
	// 	if storage.Index[:len(ClientKey)] == ClientKey {
	// 		m.keeper.RemoveStakeStorage(ctx, storage.Index)
	// 	} else if storage.Index[:len(ProviderKey)] == ProviderKey { // handle provider keys
	// 		if len(storage.Index) > len(ProviderKey) {
	// 			storage.Index = storage.Index[len(ProviderKey):]
	// 			m.keeper.SetStakeStorage(ctx, storage)
	// 		}
	// 	}
	// }
	return nil
}

// Migrate3to4 implements store migration from v3 to v4:
// set geolocation to int32
func (m Migrator) Migrate3to4(ctx sdk.Context) error {
	utils.LavaFormatDebug("migrate: epochstorage change geolocation from uint64 to int32")

	store := prefix.NewStore(ctx.KVStore(m.keeper.storeKey), v3.KeyPrefix(v3.StakeStorageKeyPrefix))
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

// Migrate4to5 implements store migration from v4 to v5:
// - initialize DelegateTotal, DelegateLimit, DelegateCommission
func (m Migrator) Migrate4to5(ctx sdk.Context) error {
	// utils.LavaFormatDebug("migrate: epochstorage to include delegations")

	// StakeStorages := m.keeper.GetAllStakeStorage(ctx)
	// for st := range StakeStorages {
	// 	for s := range StakeStorages[st].StakeEntries {
	// 		StakeStorages[st].StakeEntries[s].DelegateTotal = sdk.NewCoin(m.keeper.stakingKeeper.BondDenom(ctx), sdk.ZeroInt())
	// 		StakeStorages[st].StakeEntries[s].DelegateLimit = sdk.NewCoin(m.keeper.stakingKeeper.BondDenom(ctx), sdk.ZeroInt())
	// 		StakeStorages[st].StakeEntries[s].DelegateCommission = 100
	// 	}
	// 	m.keeper.SetStakeStorage(ctx, StakeStorages[st])
	// }

	return nil
}

// Migrate5to6 goes over all existing stake entries and populates the new vault address field with the stake entry address
func (m Migrator) Migrate5to6(ctx sdk.Context) error {
	// utils.LavaFormatDebug("migrate: epochstorage to include provider and vault addresses")

	// store := prefix.NewStore(ctx.KVStore(m.keeper.storeKey), types.KeyPrefix(string(types.StakeEntriesPrefix)))
	// iterator := sdk.KVStorePrefixIterator(store, []byte{})

	// defer iterator.Close()

	// for ; iterator.Valid(); iterator.Next() {
	// 	var stakeStorageV6 v6.StakeStorage
	// 	m.keeper.cdc.MustUnmarshal(iterator.Value(), &stakeStorageV6)

	// 	for i := range stakeStorageV6.StakeEntries {
	// 		stakeStorageV6.StakeEntries[i].Vault = stakeStorageV6.StakeEntries[i].Address
	// 	}

	// 	store.Set(iterator.Key(), m.keeper.cdc.MustMarshal(&stakeStorageV6))
	// }

	return nil
}

// Migrate6to7 goes over all existing stake entries and populates the new description field with current moniker
func (m Migrator) Migrate6to7(ctx sdk.Context) error {
	// utils.LavaFormatDebug("migrate: epochstorage to include detailed description")

	// store := prefix.NewStore(ctx.KVStore(m.keeper.storeKey), types.KeyPrefix(types.StakeEntriesPrefix))
	// iterator := sdk.KVStorePrefixIterator(store, []byte{})

	// defer iterator.Close()

	// for ; iterator.Valid(); iterator.Next() {
	// 	var stakeStorageV7 types.StakeStorage
	// 	m.keeper.cdc.MustUnmarshal(iterator.Value(), &stakeStorageV7)

	// 	for i := range stakeStorageV7.StakeEntries {
	// 		stakeStorageV7.StakeEntries[i].Description.Moniker = stakeStorageV7.StakeEntries[i].Moniker
	// 		stakeStorageV7.StakeEntries[i].Moniker = ""
	// 	}

	// 	store.Set(iterator.Key(), m.keeper.cdc.MustMarshal(&stakeStorageV7))
	// }

	return nil
}

// Migrate7to8 transfers all the stake entries from the old stake storage to the new stake entries store
// StakeStorage is set to the stake entries store
// StakeStorageCurrent is set to the stake entries current store
// StakeStorageUnstake is deleted
func (m Migrator) Migrate7to8(ctx sdk.Context) error {
	utils.LavaFormatDebug("migrate: epochstorage to move stake entries from stake storage")
	k := m.keeper

	store := prefix.NewStore(ctx.KVStore(k.storeKey), v3.KeyPrefix(v3.StakeStorageKeyPrefix))
	iterator := sdk.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		key := string(iterator.Key())

		// identify stake storage type: regular, current or unstake
		if key == v3.StakeStorageKeyUnstakeConst {
			store.Delete(iterator.Key())
			continue
		}

		epoch, err := extractEpochFromStakeStorageKey(key)
		isCurrentStakeStorage := m.isCurrentStakeStorageKey(ctx, key)
		if err != nil && !isCurrentStakeStorage {
			panic(fmt.Errorf("stake storage with unidentified index %s", key))
		}

		var stakeStorage types.StakeStorage
		k.cdc.MustUnmarshal(iterator.Value(), &stakeStorage)

		for _, entry := range stakeStorage.StakeEntries {
			if isCurrentStakeStorage {
				k.SetStakeEntryCurrent(ctx, entry)
			} else {
				k.SetStakeEntry(ctx, epoch, entry)
			}
		}

		if !isCurrentStakeStorage {
			m.SetEpochHashForMigrator(ctx, epoch, stakeStorage.EpochBlockHash)
		}

		store.Delete(iterator.Key())
	}

	return nil
}

// the legacy StakeStorage store used keys that were built like this:
// strconv.FormatUint(block, 10) + chainID
func extractEpochFromStakeStorageKey(key string) (uint64, error) {
	number := ""
	for _, char := range key {
		if !unicode.IsDigit(char) {
			break
		}
		number += string(char)
	}
	// Attempt conversion to uint64, return 0 and false if fails
	parsedUint, err := strconv.ParseUint(number, 10, 64)
	if err != nil {
		return 0, err
	}
	return parsedUint, nil
}

func (m Migrator) isCurrentStakeStorageKey(ctx sdk.Context, key string) bool {
	// the legacy StakeStorage key (both regular and current) had a "/" which should be cut off
	key, found := strings.CutSuffix(key, "/")
	if !found {
		return false
	}
	_, found, _ = m.keeper.specKeeper.IsSpecFoundAndActive(ctx, key)
	return found
}

func (m Migrator) SetEpochHashForMigrator(ctx sdk.Context, epoch uint64, hash []byte) {
	store := prefix.NewStore(ctx.KVStore(m.keeper.storeKey), types.KeyPrefix(types.EpochHashPrefix))
	store.Set(utils.SerializeBigEndian(epoch), hash)
}
