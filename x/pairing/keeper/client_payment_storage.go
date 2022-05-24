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

func (k Keeper) GetClientPaymentStorageKey(ctx sdk.Context, chainID string, epoch uint64, clientAddr sdk.AccAddress) string {
	return chainID + "_" + strconv.FormatUint(epoch, 16) + "_" + clientAddr.String()
}

func (k Keeper) AddClientPaymentInEpoch(ctx sdk.Context, chainID string, epoch uint64, userAddress sdk.AccAddress, providerAddress sdk.AccAddress, usedCU uint64, uniqueIdentifier string) (userPayment *types.ClientPaymentStorage, usedCUProviderTotal uint64, err error) {
	//key is chainID+_+epoch+_+user
	key := k.GetClientPaymentStorageKey(ctx, chainID, epoch, userAddress)
	isUnique, uniquePaymentStorageClientProviderEntryAddr := k.AddUniquePaymentStorageClientProvider(ctx, chainID, epoch, userAddress, providerAddress, uniqueIdentifier, usedCU)
	if !isUnique {
		//tried to use an existing identifier!
		return nil, 0, fmt.Errorf("failed to add user payment since uniqueIdentifier was already detected, and created on block %d", uniquePaymentStorageClientProviderEntryAddr.Block)
	}
	userPaymentStorageInEpoch, found := k.GetClientPaymentStorage(ctx, key)
	if !found {
		// is new entry
		userPaymentStorageInEpoch = types.ClientPaymentStorage{Index: key, UniquePaymentStorageClientProvider: []*types.UniquePaymentStorageClientProvider{uniquePaymentStorageClientProviderEntryAddr}, Epoch: epoch}
		usedCUProviderTotal = usedCU
	} else {
		userPaymentStorageInEpoch.UniquePaymentStorageClientProvider = append(userPaymentStorageInEpoch.UniquePaymentStorageClientProvider, uniquePaymentStorageClientProviderEntryAddr)
		// sums up usedCU for this client and this provider over this epoch
		usedCUProviderTotal, err = k.GetTotalUsedCUForProviderEpoch(ctx, providerAddress, userPaymentStorageInEpoch)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to add user payment! could not GetTotalUsedCUForProviderEpoch client: %s provider: %s", userAddress.String(), providerAddress.String())
		}
		// #O uncomment the next line to see that relayValidateCU is working
		// k.Logger(ctx).Error("!!! usedCU " + strconv.FormatUint(usedCU, 10) + " ::: totalCU for serviser " + strconv.FormatUint(usedCUProviderTotal, 10))
	}
	k.SetClientPaymentStorage(ctx, userPaymentStorageInEpoch)
	return &userPaymentStorageInEpoch, usedCUProviderTotal, nil
}

func (k Keeper) GetTotalUsedCUForProviderEpoch(ctx sdk.Context, providerAddress sdk.AccAddress, userPaymentStorageInEpoch types.ClientPaymentStorage) (usedCUProviderTotal uint64, err error) {
	usedCUProviderTotal = 0
	usedCUMap, err := k.GetEpochClientProviderUsedCUMap(ctx, userPaymentStorageInEpoch)
	if err != nil {
		return 0, err
	}
	if usedProvider, ok := usedCUMap.Providers[providerAddress.String()]; ok {
		return usedProvider, nil
	}
	return 0, nil
}
