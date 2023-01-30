package keeper

import (
	"strconv"

	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/utils"
	"github.com/lavanet/lava/x/pairing/types"
)

// SetEpochPayments set a specific epochPayments in the store from its index
func (k Keeper) SetEpochPayments(ctx sdk.Context, epochPayments types.EpochPayments) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.EpochPaymentsKeyPrefix))
	b := k.cdc.MustMarshal(&epochPayments)
	store.Set(types.EpochPaymentsKey(
		epochPayments.Index,
	), b)
}

// GetEpochPayments returns a epochPayments from its index
func (k Keeper) GetEpochPayments(
	ctx sdk.Context,
	index string,
) (val types.EpochPayments, found bool) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.EpochPaymentsKeyPrefix))

	b := store.Get(types.EpochPaymentsKey(
		index,
	))
	if b == nil {
		return val, false
	}

	k.cdc.MustUnmarshal(b, &val)
	return val, true
}

// RemoveEpochPayments removes a epochPayments from the store
func (k Keeper) RemoveEpochPayments(
	ctx sdk.Context,
	index string,
) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.EpochPaymentsKeyPrefix))
	store.Delete(types.EpochPaymentsKey(
		index,
	))
}

// GetAllEpochPayments returns all epochPayments
func (k Keeper) GetAllEpochPayments(ctx sdk.Context) (list []types.EpochPayments) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.EpochPaymentsKeyPrefix))
	iterator := sdk.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.EpochPayments
		k.cdc.MustUnmarshal(iterator.Value(), &val)
		list = append(list, val)
	}

	return
}

// Function to remove epochPayments objects from deleted epochs (older than the chain's memory)
func (k Keeper) RemoveOldEpochPayment(ctx sdk.Context) (err error) {
	for _, epoch := range k.epochStorageKeeper.GetDeletedEpochs(ctx) {
		err = k.RemoveAllEpochPaymentsForBlock(ctx, epoch)
	}
	return
}

// Function to get the epochPayments object from a specific epoch. Note that it also returns the epochPayments object's key which is the epoch in hex representation (base 16)
func (k Keeper) GetEpochPaymentsFromBlock(ctx sdk.Context, epoch uint64) (epochPayment types.EpochPayments, found bool, key string) {
	key = strconv.FormatUint(epoch, 16)
	epochPayment, found = k.GetEpochPayments(ctx, key)
	return
}

// Function to add an epoch payment to the epochPayments object
func (k Keeper) AddEpochPayment(ctx sdk.Context, chainID string, epoch uint64, userAddress sdk.AccAddress, providerAddress sdk.AccAddress, usedCU uint64, uniqueIdentifier string) (uint64, error) {
	// add a uniquePaymentStorageClientProvider object (the object that represent the actual payment) to this epoch's providerPaymentPayment object
	userPaymentProviderStorage, usedCUProviderTotal, err := k.AddProviderPaymentInEpoch(ctx, chainID, epoch, userAddress, providerAddress, usedCU, uniqueIdentifier)
	if err != nil {
		return 0, utils.LavaFormatError("could not add epoch payment", err, &map[string]string{"userAddress": userAddress.String(), "providerAddress": providerAddress.String(), "uniqueIdentifier": uniqueIdentifier, "epoch": strconv.FormatUint(epoch, 10), "chainID": chainID})
	}

	// get this epoch's epochPayments object
	epochPayments, found, key := k.GetEpochPaymentsFromBlock(ctx, epoch)
	if !found {
		// this epoch doesn't have a epochPayments object, create one with the providerPaymentStorage object from before
		epochPayments = types.EpochPayments{Index: key, ProviderPaymentStorageKeys: []string{userPaymentProviderStorage.GetIndex()}}
	} else {
		// this epoch has a epochPayments object -> make sure this payment is not already in this object
		providerPaymentStorageKeyFound := false
		for _, providerPaymentStorageKey := range epochPayments.GetProviderPaymentStorageKeys() {
			if providerPaymentStorageKey == userPaymentProviderStorage.GetIndex() {
				providerPaymentStorageKeyFound = true
				break
			}
		}

		// this epoch's epochPayments object doesn't contain this providerPaymentStorage key -> append the new key
		if !providerPaymentStorageKeyFound {
			epochPayments.ProviderPaymentStorageKeys = append(epochPayments.ProviderPaymentStorageKeys, userPaymentProviderStorage.GetIndex())
		}
	}

	// update the epochPayments object
	k.SetEpochPayments(ctx, epochPayments)

	return usedCUProviderTotal, nil
}

// Function to remove all epochPayments objects from a specific epoch
func (k Keeper) RemoveAllEpochPaymentsForBlock(ctx sdk.Context, blockForDelete uint64) error {
	// get the epochPayments object of blockForDelete
	epochPayments, found, key := k.GetEpochPaymentsFromBlock(ctx, blockForDelete)
	if !found {
		// return fmt.Errorf("did not find any epochPayments for block %d", blockForDelete.Num)
		// no epochPayments object -> do nothing
		return nil
	}

	// go over the epochPayments object's providerPaymentStorageKeys
	userPaymentsStorageKeys := epochPayments.GetProviderPaymentStorageKeys()
	for _, userPaymentStorageKey := range userPaymentsStorageKeys {
		// get the providerPaymentStorage object
		userPaymentStorage, found := k.GetProviderPaymentStorage(ctx, userPaymentStorageKey)
		if !found {
			return utils.LavaError(ctx, k.Logger(ctx), "get_provider_payment_storage", map[string]string{"providerPaymentStorageKey": userPaymentStorageKey}, "could not get userPaymentStorage")
		}

		// go over the providerPaymentStorage object's uniquePaymentStorageClientProviderKeys
		uniquePaymentStoragesCliProKeys := userPaymentStorage.GetUniquePaymentStorageClientProviderKeys()
		for _, uniquePaymentStorageKey := range uniquePaymentStoragesCliProKeys {
			// get the uniquePaymentStorageClientProvider object
			uniquePaymentStorage, found := k.GetUniquePaymentStorageClientProvider(ctx, uniquePaymentStorageKey)
			if !found {
				continue
			}

			// validate its an old entry, for sanity
			if uniquePaymentStorage.Block > blockForDelete {
				errMsg := "trying to delete a new entry in epoch payments for block"
				k.Logger(ctx).Error(errMsg)
				panic(errMsg)
			}

			// delete the uniquePaymentStorageClientProvider object
			k.RemoveUniquePaymentStorageClientProvider(ctx, uniquePaymentStorage.Index)
		}

		// after we're done deleting the uniquePaymentStorageClientProvider objects, delete the providerPaymentStorage object
		k.RemoveProviderPaymentStorage(ctx, userPaymentStorage.Index)
	}

	// after we're done deleting the providerPaymentStorage objects, delete the epochPayments object
	k.RemoveEpochPayments(ctx, key)
	return nil
}
