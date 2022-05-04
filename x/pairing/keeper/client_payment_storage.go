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

func (k Keeper) AddClientPaymentInEpoch(ctx sdk.Context, epoch uint64, userAddress sdk.AccAddress, servicerAddress sdk.AccAddress, usedCU uint64, uniqueIdentifier string) (userPayment *types.ClientPaymentStorage, usedCUProviderTotal uint64, err error) {
	k.Logger(ctx).Error("!!!! AddClientPaymentInEpoch !!!!")
	//key is epoch+user
	key := strconv.FormatUint(epoch, 16) + userAddress.String()
	// key := strconv.FormatUint(epoch, 10) + userAddress.String()
	isUnique, uniquePaymentStorageClientProviderEntryAddr := k.AddUniquePaymentStorageClientProvider(ctx, epoch, userAddress, servicerAddress, uniqueIdentifier, usedCU)
	if !isUnique {
		//tried to use an existing identifier!
		// uniquePaymentStorageClientProviderEntryAddr.Index = uniquePaymentStorageClientProviderEntryAddr.Index[:len(uniquePaymentStorageClientProviderEntryAddr.Index)-3] + "xxx" // this is to bypass this error
		return nil, 0, fmt.Errorf("failed to add user payment since uniqueIdentifier was already detected, and created on block %d", uniquePaymentStorageClientProviderEntryAddr.Block)
	}
	// k.Logger(ctx).Error("! 000 key " + key)
	userPaymentStorageInEpoch, found := k.GetClientPaymentStorage(ctx, key)
	// currentUser, currentServicer, currentUniqID := k.DecodeUniquePaymentKey(ctx, uniquePaymentStorageClientProviderEntryAddr.Index) // delete - used for logs
	if !found {
		// is new entry
		// k.Logger(ctx).Error("! 111 usedCU " + strconv.FormatUint(usedCU, 10) + " epoch: " + strconv.FormatUint(epoch, 10))
		// k.Logger(ctx).Error("! 111 ::: " + currentUser + " ::: " + currentServicer + " :::" + currentUniqID + " ::: ")
		userPaymentStorageInEpoch = types.ClientPaymentStorage{Index: key, UniquePaymentStorageClientProvider: []*types.UniquePaymentStorageClientProvider{uniquePaymentStorageClientProviderEntryAddr}, TotalCU: usedCU, Epoch: epoch}
		usedCUProviderTotal = usedCU
	} else {
		// k.Logger(ctx).Error("! 222 !@!@!@! usedCU " + strconv.FormatUint(usedCU, 10) + " epoch: " + strconv.FormatUint(epoch, 10))
		userPaymentStorageInEpoch.UniquePaymentStorageClientProvider = append(userPaymentStorageInEpoch.UniquePaymentStorageClientProvider, uniquePaymentStorageClientProviderEntryAddr)
		// TODO: Add used CU to specific provider : in unique or sum of unique
		for _, paymentInEpoch := range userPaymentStorageInEpoch.UniquePaymentStorageClientProvider {
			// k.Logger(ctx).Error("! ::: Servicer " + servicerAddress.String() + " usedCU: " + strconv.FormatUint(paymentInEpoch.UsedCU, 10))
			_, servicer, _ := k.DecodeUniquePaymentKey(ctx, paymentInEpoch.Index)
			if servicerAddress.String() == servicer {
				// k.Logger(ctx).Error("! ::: +++++++ " + paymentInEpoch.Index)
				usedCUProviderTotal += paymentInEpoch.UsedCU
			} else {
				// k.Logger(ctx).Error("! ::: ------- " + servicerAddress.String() + " --- " + servicer)
			}
		}
		//TODO: DELETE TotalCU ?
		userPaymentStorageInEpoch.TotalCU += usedCU
	}
	k.SetClientPaymentStorage(ctx, userPaymentStorageInEpoch)
	// k.Logger(ctx).Error("! ::: FINAL TOTAL! ::: \n usedCU: " + strconv.FormatUint(usedCU, 10) + " total: " + strconv.FormatUint((usedCUProviderTotal), 10) + " ::: !!!!!!! ")
	return &userPaymentStorageInEpoch, usedCUProviderTotal, nil
}
