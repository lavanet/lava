package keeper

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/utils"
	"github.com/lavanet/lava/x/pairing/types"
)

// SetProviderPaymentStorage set a specific providerPaymentStorage in the store from its index
func (k Keeper) SetProviderPaymentStorage(ctx sdk.Context, providerPaymentStorage types.ProviderPaymentStorage) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.ProviderPaymentStorageKeyPrefix))
	b := k.cdc.MustMarshal(&providerPaymentStorage)
	store.Set(types.ProviderPaymentStorageKey(
		providerPaymentStorage.Index,
	), b)
}

// GetProviderPaymentStorage returns a providerPaymentStorage from its index
func (k Keeper) GetProviderPaymentStorage(
	ctx sdk.Context,
	index string,
) (val types.ProviderPaymentStorage, found bool) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.ProviderPaymentStorageKeyPrefix))

	b := store.Get(types.ProviderPaymentStorageKey(
		index,
	))
	if b == nil {
		return val, false
	}

	k.cdc.MustUnmarshal(b, &val)
	return val, true
}

func (k EpochPaymentHandler) SetProviderPaymentStorageCached(ctx sdk.Context, providerPaymentStorage types.ProviderPaymentStorage) {
	b := k.cdc.MustMarshal(&providerPaymentStorage)
	k.ProviderPaymentStorageCache.Set(types.ProviderPaymentStorageKey(providerPaymentStorage.Index), b)
}

func (k EpochPaymentHandler) GetProviderPaymentStorageCached(
	ctx sdk.Context,
	index string,
) (val types.ProviderPaymentStorage, found bool) {
	b := k.ProviderPaymentStorageCache.Get(types.ProviderPaymentStorageKey(index))
	if b == nil {
		return val, false
	}

	k.cdc.MustUnmarshal(b, &val)
	return val, true
}

// RemoveProviderPaymentStorage removes a providerPaymentStorage from the store
func (k Keeper) RemoveProviderPaymentStorage(
	ctx sdk.Context,
	index string,
) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.ProviderPaymentStorageKeyPrefix))
	store.Delete(types.ProviderPaymentStorageKey(
		index,
	))
}

// GetAllProviderPaymentStorage returns all providerPaymentStorage
func (k Keeper) GetAllProviderPaymentStorage(ctx sdk.Context) (list []types.ProviderPaymentStorage) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.ProviderPaymentStorageKeyPrefix))
	iterator := sdk.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.ProviderPaymentStorage
		k.cdc.MustUnmarshal(iterator.Value(), &val)
		list = append(list, val)
	}

	return
}

func (k Keeper) GetProviderFromProviderPaymentStorage(providerPaymentStorage *types.ProviderPaymentStorage) (string, error) {
	index := providerPaymentStorage.Index
	// index consists of chain_epoch_providerAddress
	lastIndex := strings.LastIndex(index, "_")
	if lastIndex != -1 {
		return index[lastIndex+1:], nil
	}
	return "", fmt.Errorf("invalid provider payment storage key %s", index)
}

// Function to get a providerPaymentStorage object's key (key is chainID_epoch_providerAddress, epoch in hex representation)
func (k Keeper) GetProviderPaymentStorageKey(ctx sdk.Context, chainID string, epoch uint64, providerAddress sdk.AccAddress) string {
	return chainID + "_" + strconv.FormatUint(epoch, 16) + "_" + providerAddress.String()
}

// Function to add a payment (which is represented by a uniquePaymentStorageClientProvider object) to a providerPaymentStorage object
func (k EpochPaymentHandler) AddProviderPaymentInEpoch(ctx sdk.Context, chainID string, epoch uint64, projectID string, providerAddress sdk.AccAddress, usedCU uint64, uniqueIdentifier string) (userPayment *types.ProviderPaymentStorage, usedCUConsumerTotal uint64) {
	// create an uniquePaymentStorageClientProvider object and set it in the KVStore
	uniquePaymentStorageClientProviderEntryAddr := k.AddUniquePaymentStorageClientProvider(ctx, chainID, epoch, projectID, providerAddress, uniqueIdentifier, usedCU)

	// get the providerPaymentStorage object
	providerPaymentStorageKey := k.GetProviderPaymentStorageKey(ctx, chainID, epoch, providerAddress)
	userPaymentStorageInEpoch, found := k.GetProviderPaymentStorageCached(ctx, providerPaymentStorageKey)
	if !found {
		// is new entry -> create a new providerPaymentStorage object
		userPaymentStorageInEpoch = types.ProviderPaymentStorage{Index: providerPaymentStorageKey, UniquePaymentStorageClientProviderKeys: []string{uniquePaymentStorageClientProviderEntryAddr.GetIndex()}, Epoch: epoch}
		usedCUConsumerTotal = usedCU
	} else {
		// found the providerPaymentStorage object -> append the uniquePaymentStorageClientProvider object's key
		userPaymentStorageInEpoch.UniquePaymentStorageClientProviderKeys = append(userPaymentStorageInEpoch.UniquePaymentStorageClientProviderKeys, uniquePaymentStorageClientProviderEntryAddr.GetIndex())

		// sum up the used CU for this provider and this consumer over this epoch
		usedCUConsumerTotal = k.GetTotalUsedCUForConsumerPerEpoch(ctx, projectID, userPaymentStorageInEpoch.GetUniquePaymentStorageClientProviderKeys(), providerAddress.String())
	}

	// set the providerPaymentStorage object in the KVStore
	k.SetProviderPaymentStorageCached(ctx, userPaymentStorageInEpoch)

	return &userPaymentStorageInEpoch, usedCUConsumerTotal
}

// Function to get the total serviced CU by a provider in this epoch for a specific consumer
func (k EpochPaymentHandler) GetTotalUsedCUForConsumerPerEpoch(ctx sdk.Context, projectID string, uniquePaymentStorageKeys []string, providerAddress string) uint64 {
	usedCUProviderTotal := uint64(0)

	// go over the uniquePaymentStorageKeys
	for _, uniquePaymentKey := range uniquePaymentStorageKeys {
		// if the uniquePaymentStorageClientProvider object is not between the provider and the specific consumer, continue
		if k.GetConsumerFromUniquePayment(uniquePaymentKey) != projectID {
			continue
		}
		// get the uniquePaymentStorageClientProvider object
		uniquePayment, found := k.GetUniquePaymentStorageClientProviderCached(ctx, uniquePaymentKey)
		if !found {
			uniquePayment, found = k.GetUniquePaymentStorageClientProvider(ctx, uniquePaymentKey)
			if found {
				k.SetUniquePaymentStorageClientProviderCached(ctx, uniquePayment)
			}
		}
		if !found {
			utils.LavaFormatError("could not find uniquePaymentStorageClientProvider object", fmt.Errorf("unique payment object not found"),
				utils.Attribute{Key: "providerAddress", Value: providerAddress},
				utils.Attribute{Key: "projectID", Value: projectID},
			)
			continue
		}
		usedCUProviderTotal += uniquePayment.UsedCU
	}
	return usedCUProviderTotal
}
