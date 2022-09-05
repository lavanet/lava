package keeper

import (
	"fmt"
	"strconv"

	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
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

func (k Keeper) GetProviderPaymentStorageKey(ctx sdk.Context, chainID string, epoch uint64, providerAddress sdk.AccAddress) string {
	return chainID + "_" + strconv.FormatUint(epoch, 16) + "_" + providerAddress.String()
}

func (k Keeper) AddClientPaymentInEpoch(ctx sdk.Context, chainID string, epoch uint64, userAddress sdk.AccAddress, providerAddress sdk.AccAddress, usedCU uint64, uniqueIdentifier string) (userPayment *types.ProviderPaymentStorage, usedCUProviderTotal uint64, err error) {
	//key is chainID+_+epoch+_+user
	key := k.GetProviderPaymentStorageKey(ctx, chainID, epoch, providerAddress)
	isUnique, uniquePaymentStorageClientProviderEntryAddr := k.AddUniquePaymentStorageClientProvider(ctx, chainID, epoch, userAddress, providerAddress, uniqueIdentifier, usedCU)
	if !isUnique {
		//tried to use an existing identifier!
		return nil, 0, fmt.Errorf("failed to add user payment since uniqueIdentifier was already detected, and created on block %d", uniquePaymentStorageClientProviderEntryAddr.Block)
	}
	userPaymentStorageInEpoch, found := k.GetProviderPaymentStorage(ctx, key)
	if !found {
		// is new entry
		userPaymentStorageInEpoch = types.ProviderPaymentStorage{Index: key, UniquePaymentStorageClientProvider: []*types.UniquePaymentStorageClientProvider{uniquePaymentStorageClientProviderEntryAddr}, Epoch: epoch}
		usedCUProviderTotal = usedCU
	} else {
		userPaymentStorageInEpoch.UniquePaymentStorageClientProvider = append(userPaymentStorageInEpoch.UniquePaymentStorageClientProvider, uniquePaymentStorageClientProviderEntryAddr)
		// sums up usedCU for this client and this provider over this epoch
		usedCUProviderTotal = k.GetTotalUsedCUForProviderEpoch(ctx, providerAddress, userPaymentStorageInEpoch)
	}
	k.SetProviderPaymentStorage(ctx, userPaymentStorageInEpoch)
	return &userPaymentStorageInEpoch, usedCUProviderTotal, nil
}

func (k Keeper) GetTotalUsedCUForProviderEpoch(ctx sdk.Context, providerAddress sdk.AccAddress, userPaymentStorageInEpoch types.ProviderPaymentStorage) (usedCUProviderTotal uint64) {
	usedCUProviderTotal = 0
	usedCUMap := k.GetEpochClientProviderUsedCUMap(ctx, userPaymentStorageInEpoch)
	if usedProvider, ok := usedCUMap.Providers[providerAddress.String()]; ok {
		return usedProvider
	}
	return 0
}
