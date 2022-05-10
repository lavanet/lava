package keeper

import (
	"fmt"

	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/x/pairing/types"
)

// SetUniquePaymentStorageClientProvider set a specific uniquePaymentStorageClientProvider in the store from its index
func (k Keeper) SetUniquePaymentStorageClientProvider(ctx sdk.Context, uniquePaymentStorageClientProvider types.UniquePaymentStorageClientProvider) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.UniquePaymentStorageClientProviderKeyPrefix))
	b := k.cdc.MustMarshal(&uniquePaymentStorageClientProvider)
	store.Set(types.UniquePaymentStorageClientProviderKey(
		uniquePaymentStorageClientProvider.Index,
	), b)
}

// GetUniquePaymentStorageClientProvider returns a uniquePaymentStorageClientProvider from its index
func (k Keeper) GetUniquePaymentStorageClientProvider(
	ctx sdk.Context,
	index string,

) (val types.UniquePaymentStorageClientProvider, found bool) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.UniquePaymentStorageClientProviderKeyPrefix))

	b := store.Get(types.UniquePaymentStorageClientProviderKey(
		index,
	))
	if b == nil {
		return val, false
	}

	k.cdc.MustUnmarshal(b, &val)
	return val, true
}

// RemoveUniquePaymentStorageClientProvider removes a uniquePaymentStorageClientProvider from the store
func (k Keeper) RemoveUniquePaymentStorageClientProvider(
	ctx sdk.Context,
	index string,

) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.UniquePaymentStorageClientProviderKeyPrefix))
	store.Delete(types.UniquePaymentStorageClientProviderKey(
		index,
	))
}

// GetAllUniquePaymentStorageClientProvider returns all uniquePaymentStorageClientProvider
func (k Keeper) GetAllUniquePaymentStorageClientProvider(ctx sdk.Context) (list []types.UniquePaymentStorageClientProvider) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.UniquePaymentStorageClientProviderKeyPrefix))
	iterator := sdk.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.UniquePaymentStorageClientProvider
		k.cdc.MustUnmarshal(iterator.Value(), &val)
		list = append(list, val)
	}

	return
}

func (k Keeper) AddUniquePaymentStorageClientProvider(ctx sdk.Context,
	block uint64, userAddress sdk.AccAddress, providerAddress sdk.AccAddress, uniqueIdentifier string, usedCU uint64) (bool, *types.UniquePaymentStorageClientProvider) {
	key := k.EncodeUniquePaymentKey(ctx, userAddress, providerAddress, uniqueIdentifier)
	entry, found := k.GetUniquePaymentStorageClientProvider(ctx, key)
	if found {
		return false, &entry
	}
	entry = types.UniquePaymentStorageClientProvider{Index: key, Block: block, UsedCU: usedCU}
	k.SetUniquePaymentStorageClientProvider(ctx, entry)
	return true, &entry
}

func (k Keeper) GetProviderFromUniquePayment(ctx sdk.Context, uniquePaymentStorageClientProvider types.UniquePaymentStorageClientProvider) string {
	_, provider, _ := k.DecodeUniquePaymentKey(ctx, uniquePaymentStorageClientProvider.Index)
	return provider
}

func addressLengths() (int, int) {
	//TODO: Get these values from AccAddress somehow and remove AdrLengthUser and AdrLengthProvider from pairing/types/key_unique_payment_storage_client_provider
	adrLengthUser, adrLengthProvider := types.AdrLengthUser, types.AdrLengthProvider
	return adrLengthUser, adrLengthProvider
}

//TODO: refactor to xxx_provider_client_uid
func (k Keeper) DecodeUniquePaymentKey(ctx sdk.Context, key string) (string, string, string) {
	adrLengthUser, adrLengthProvider := addressLengths()

	userAddress := key[:adrLengthUser]
	providerAddress := key[adrLengthUser : adrLengthUser+adrLengthProvider]
	uniqueIdentifier := key[adrLengthUser+adrLengthProvider:]

	return userAddress, providerAddress, uniqueIdentifier
}

//TODO: refactor to xxx_provider_client_uid
func (k Keeper) EncodeUniquePaymentKey(ctx sdk.Context, userAddress sdk.AccAddress, providerAddress sdk.AccAddress, uniqueIdentifier string) string {
	adrLengthUser, adrLengthProvider := addressLengths()
	if len(userAddress.String()) != adrLengthUser {
		panic(fmt.Sprintf("invalid userAddress found! len(%s) != %d == %d", userAddress.String(), adrLengthUser, len(userAddress.String())))
	} else if len(providerAddress.String()) != adrLengthProvider {
		panic(fmt.Sprintf("invalid providerAddress found! len(%s) != %d == %d", providerAddress.String(), adrLengthProvider, len(providerAddress.String())))
	}
	key := userAddress.String() + providerAddress.String() + uniqueIdentifier
	return key
}
