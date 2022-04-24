package keeper

import (
	"fmt"
	"strconv"

	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/x/pairing/types"
)

// SetClientPaymentStorage set a specific clientPaymentStorage in the store from its index
func (k Keeper) SetClientPaymentStorage(ctx sdk.Context, clientPaymentStorage types.ClientPaymentStorage) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.ClientPaymentStorageKeyPrefix))
	b := k.cdc.MustMarshal(&clientPaymentStorage)
	store.Set(types.ClientPaymentStorageKey(
		clientPaymentStorage.Index,
	), b)
}

// GetClientPaymentStorage returns a clientPaymentStorage from its index
func (k Keeper) GetClientPaymentStorage(
	ctx sdk.Context,
	index string,

) (val types.ClientPaymentStorage, found bool) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.ClientPaymentStorageKeyPrefix))

	b := store.Get(types.ClientPaymentStorageKey(
		index,
	))
	if b == nil {
		return val, false
	}

	k.cdc.MustUnmarshal(b, &val)
	return val, true
}

// RemoveClientPaymentStorage removes a clientPaymentStorage from the store
func (k Keeper) RemoveClientPaymentStorage(
	ctx sdk.Context,
	index string,

) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.ClientPaymentStorageKeyPrefix))
	store.Delete(types.ClientPaymentStorageKey(
		index,
	))
}

// GetAllClientPaymentStorage returns all clientPaymentStorage
func (k Keeper) GetAllClientPaymentStorage(ctx sdk.Context) (list []types.ClientPaymentStorage) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.ClientPaymentStorageKeyPrefix))
	iterator := sdk.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.ClientPaymentStorage
		k.cdc.MustUnmarshal(iterator.Value(), &val)
		list = append(list, val)
	}

	return
}

func (k Keeper) AddClientPaymentInEpoch(ctx sdk.Context, epoch uint64, userAddress sdk.AccAddress, servicerAddress sdk.AccAddress, usedCU uint64, uniqueIdentifier string) (userPayment *types.ClientPaymentStorage, err error) {
	//key is epoch+user
	key := strconv.FormatUint(epoch, 16) + userAddress.String()
	isUnique, uniquePaymentStorageClientProviderEntryAddr := k.AddUniquePaymentStorageClientProvider(ctx, epoch, userAddress, servicerAddress, uniqueIdentifier)
	if !isUnique {
		//tried to use an existing identifier!
		return nil, fmt.Errorf("failed to add user payment since uniqueIdentifier was already detected, and created on block %d", uniquePaymentStorageClientProviderEntryAddr.Block)
	}
	userPaymentStorageInEpoch, found := k.GetClientPaymentStorage(ctx, key)
	if !found {
		// is new entry
		userPaymentStorageInEpoch = types.ClientPaymentStorage{Index: key, UniquePaymentStorageClientProvider: []*types.UniquePaymentStorageClientProvider{uniquePaymentStorageClientProviderEntryAddr}, TotalCU: usedCU, Epoch: epoch}
	} else {
		userPaymentStorageInEpoch.UniquePaymentStorageClientProvider = append(userPaymentStorageInEpoch.UniquePaymentStorageClientProvider, uniquePaymentStorageClientProviderEntryAddr)
		userPaymentStorageInEpoch.TotalCU += usedCU
	}
	k.SetClientPaymentStorage(ctx, userPaymentStorageInEpoch)
	return &userPaymentStorageInEpoch, nil
}
