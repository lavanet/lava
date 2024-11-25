package v2

import (
	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/v4/x/pairing/types"
)

// RemoveAllEpochPayments removes all epochPayments
func RemoveAllEpochPayments(ctx sdk.Context, storeKey storetypes.StoreKey) {
	store := prefix.NewStore(ctx.KVStore(storeKey), types.KeyPrefix(EpochPaymentsKeyPrefix))
	iterator := sdk.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		store.Delete(iterator.Key())
	}
}

// RemoveAllProviderPaymentStorage removes all providerPaymentStorage
func RemoveAllProviderPaymentStorage(ctx sdk.Context, storeKey storetypes.StoreKey) {
	store := prefix.NewStore(ctx.KVStore(storeKey), types.KeyPrefix(ProviderPaymentStorageKeyPrefix))
	iterator := sdk.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		store.Delete(iterator.Key())
	}
}

// RemoveAllUniquePaymentStorageClientProvider removes all uniquePaymentStorageClientProvider
func RemoveAllUniquePaymentStorageClientProvider(ctx sdk.Context, storeKey storetypes.StoreKey) {
	store := prefix.NewStore(ctx.KVStore(storeKey), types.KeyPrefix(UniquePaymentStorageClientProviderKeyPrefix))
	iterator := sdk.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		store.Delete(iterator.Key())
	}
}
