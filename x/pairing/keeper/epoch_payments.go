package keeper

import (
	"fmt"
	"strconv"

	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
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

func (k Keeper) RemoveOldEpochPayment(ctx sdk.Context) (err error) {
	if uint64(ctx.BlockHeight()) < k.epochStorageKeeper.BlocksToSave(ctx) {
		return nil
	}
	block := uint64(ctx.BlockHeight()) - k.epochStorageKeeper.BlocksToSave(ctx)
	earliestEpochBlock := k.epochStorageKeeper.GetEarliestEpochStart(ctx)
	if earliestEpochBlock > block {
		return nil
	}
	//we passed the distance to earliest epoch block, so remove the entries
	err = k.RemoveAllEpochPaymentsForBlock(ctx, earliestEpochBlock)
	return
}

func (k Keeper) GetEpochPaymentsFromBlock(ctx sdk.Context, epoch uint64) (epochPayment types.EpochPayments, found bool, key string) {
	key = strconv.FormatUint(epoch, 16)
	epochPayment, found = k.GetEpochPayments(ctx, key)
	return
}

func (k Keeper) AddEpochPayment(ctx sdk.Context, chainID string, epoch uint64, userAddress sdk.AccAddress, providerAddress sdk.AccAddress, usedCU uint64, uniqueIdentifier string) (uint64, error) {
	userPaymentProviderStorage, usedCUProviderTotal, err := k.AddClientPaymentInEpoch(ctx, chainID, epoch, userAddress, providerAddress, usedCU, uniqueIdentifier)
	if err != nil {
		return 0, fmt.Errorf("could not add epoch payment: %s,%s,%s,%d error: %s", userAddress, providerAddress, uniqueIdentifier, epoch, err)
	}

	epochPayments, found, key := k.GetEpochPaymentsFromBlock(ctx, epoch)
	if !found {
		epochPayments = types.EpochPayments{Index: key, ClientsPayments: []*types.ClientPaymentStorage{userPaymentProviderStorage}}
	} else {
		epochPayments.ClientsPayments = append(epochPayments.ClientsPayments, userPaymentProviderStorage)
	}
	k.SetEpochPayments(ctx, epochPayments)
	return usedCUProviderTotal, nil
}

func (k Keeper) RemoveAllEpochPaymentsForBlock(ctx sdk.Context, blockForDelete uint64) error {
	//remove the old epochs
	epochPayments, found, key := k.GetEpochPaymentsFromBlock(ctx, blockForDelete)
	if !found {
		// return fmt.Errorf("did not find any epochPayments for block %d", blockForDelete.Num)
		return nil
	}
	userPaymentsStorages := epochPayments.ClientsPayments
	for _, userPaymentStorage := range userPaymentsStorages {
		uniquePaymentStoragesCliPro := userPaymentStorage.UniquePaymentStorageClientProvider
		for _, uniquePaymentStorageCliPro := range uniquePaymentStoragesCliPro {
			//validate its an old entry, for sanity
			if uniquePaymentStorageCliPro.Block > blockForDelete {
				errMsg := "trying to delete a new entry in epoch payments for block"
				k.Logger(ctx).Error(errMsg)
				panic(errMsg)
			}
			//delete all payment storages
			k.RemoveUniquePaymentStorageClientProvider(ctx, uniquePaymentStorageCliPro.Index)
		}
		k.RemoveClientPaymentStorage(ctx, userPaymentStorage.Index)
	}
	k.RemoveEpochPayments(ctx, key)
	return nil
}
