package keeper

import (
	"fmt"
	"strconv"

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

// Function to get a providerPaymentStorage object's key (key is chainID_epoch_providerAddress, epoch in hex representation)
func (k Keeper) GetProviderPaymentStorageKey(ctx sdk.Context, chainID string, epoch uint64, providerAddress sdk.AccAddress) string {
	return chainID + "_" + strconv.FormatUint(epoch, 16) + "_" + providerAddress.String()
}

// Function to add a payment (which is represented by a uniquePaymentStorageClientProvider object) to a providerPaymentStorage object
func (k Keeper) AddProviderPaymentInEpoch(ctx sdk.Context, chainID string, epoch uint64, userAddress sdk.AccAddress, providerAddress sdk.AccAddress, usedCU uint64, uniqueIdentifier string) (userPayment *types.ProviderPaymentStorage, usedCUConsumerTotal uint64, err error) {
	// create an uniquePaymentStorageClientProvider object and set it in the KVStore
	isUnique, uniquePaymentStorageClientProviderEntryAddr := k.AddUniquePaymentStorageClientProvider(ctx, chainID, epoch, userAddress, providerAddress, uniqueIdentifier, usedCU)
	if !isUnique {
		// the uniquePaymentStorageClientProvider object is not unique -> tried to use an existing identifier!
		return nil, 0, fmt.Errorf("failed to add user payment since uniqueIdentifier was already detected, and created on block %d", uniquePaymentStorageClientProviderEntryAddr.Block)
	}

	// get the providerPaymentStorage object
	providerPaymentStorageKey := k.GetProviderPaymentStorageKey(ctx, chainID, epoch, providerAddress)
	userPaymentStorageInEpoch, found := k.GetProviderPaymentStorage(ctx, providerPaymentStorageKey)
	if !found {
		// is new entry -> create a new providerPaymentStorage object
		userPaymentStorageInEpoch = types.ProviderPaymentStorage{Index: providerPaymentStorageKey, UniquePaymentStorageClientProviderKeys: []string{uniquePaymentStorageClientProviderEntryAddr.GetIndex()}, Epoch: epoch}
		usedCUConsumerTotal = usedCU
	} else {
		// found the providerPaymentStorage object -> append the uniquePaymentStorageClientProvider object's key
		userPaymentStorageInEpoch.UniquePaymentStorageClientProviderKeys = append(userPaymentStorageInEpoch.UniquePaymentStorageClientProviderKeys, uniquePaymentStorageClientProviderEntryAddr.GetIndex())

		// sum up the used CU for this provider and this consumer over this epoch
		usedCUConsumerTotal, err = k.GetTotalUsedCUForConsumerPerEpoch(ctx, userAddress.String(), userPaymentStorageInEpoch.GetUniquePaymentStorageClientProviderKeys(), providerAddress.String())
		if err != nil {
			return nil, 0, err
		}
	}

	// set the providerPaymentStorage object in the KVStore
	k.SetProviderPaymentStorage(ctx, userPaymentStorageInEpoch)

	return &userPaymentStorageInEpoch, usedCUConsumerTotal, nil
}

// Function to get the total serviced CU by a provider in this epoch for a specific consumer
func (k Keeper) GetTotalUsedCUForConsumerPerEpoch(ctx sdk.Context, consumerAddress string, uniquePaymentStorageKeys []string, providerAddress string) (uint64, error) {
	usedCUProviderTotal := uint64(0)

	// go over the uniquePaymentStorageKeys
	for _, uniquePaymentKey := range uniquePaymentStorageKeys {
		// get the uniquePaymentStorageClientProvider object
		uniquePayment, found := k.GetUniquePaymentStorageClientProvider(ctx, uniquePaymentKey)
		if !found {
			return 0, utils.LavaFormatError("could not find uniquePaymentStorageClientProvider object", fmt.Errorf("unique payment object not found"),
				utils.Attribute{Key: "providerAddress", Value: providerAddress},
				utils.Attribute{Key: "consumerAddress", Value: consumerAddress},
			)
		}

		// if the uniquePaymentStorageClientProvider object is between the provider and the specific consumer, add the serviced CU
		if k.GetConsumerFromUniquePayment(&uniquePayment) == consumerAddress {
			usedCUProviderTotal += uniquePayment.UsedCU
		}
	}
	return usedCUProviderTotal, nil
}
