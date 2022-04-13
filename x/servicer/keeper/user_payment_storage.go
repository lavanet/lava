package keeper

import (
	"fmt"
	"strconv"

	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/x/servicer/types"
)

// SetUserPaymentStorage set a specific userPaymentStorage in the store from its index
func (k Keeper) SetUserPaymentStorage(ctx sdk.Context, userPaymentStorage types.UserPaymentStorage) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.UserPaymentStorageKeyPrefix))
	b := k.cdc.MustMarshal(&userPaymentStorage)
	store.Set(types.UserPaymentStorageKey(
		userPaymentStorage.Index,
	), b)
}

// GetUserPaymentStorage returns a userPaymentStorage from its index
func (k Keeper) GetUserPaymentStorage(
	ctx sdk.Context,
	index string,

) (val types.UserPaymentStorage, found bool) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.UserPaymentStorageKeyPrefix))

	b := store.Get(types.UserPaymentStorageKey(
		index,
	))
	if b == nil {
		return val, false
	}

	k.cdc.MustUnmarshal(b, &val)
	return val, true
}

// RemoveUserPaymentStorage removes a userPaymentStorage from the store
func (k Keeper) RemoveUserPaymentStorage(
	ctx sdk.Context,
	index string,

) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.UserPaymentStorageKeyPrefix))
	store.Delete(types.UserPaymentStorageKey(
		index,
	))
}

// GetAllUserPaymentStorage returns all userPaymentStorage
func (k Keeper) GetAllUserPaymentStorage(ctx sdk.Context) (list []types.UserPaymentStorage) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.UserPaymentStorageKeyPrefix))
	iterator := sdk.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.UserPaymentStorage
		k.cdc.MustUnmarshal(iterator.Value(), &val)
		list = append(list, val)
	}

	return
}

func (k Keeper) AddUserPaymentInSession(ctx sdk.Context, session uint64, userAddress sdk.AccAddress, servicerAddress sdk.AccAddress, usedCU uint64, uniqueIdentifier string) (userPayment *types.UserPaymentStorage, err error) {
	//key is session+user
	key := strconv.FormatUint(session, 16) + userAddress.String()
	isUnique, uniquePaymentStorageUserServicerEntryAddr := k.AddUniquePaymentStorageUserServicer(ctx, session, userAddress, servicerAddress, uniqueIdentifier)
	if !isUnique {
		//tried to use an existing identifier!
		return nil, fmt.Errorf("failed to add user payment since uniqueIdentifier was already detected, and created on block %d", uniquePaymentStorageUserServicerEntryAddr.Block)
	}
	userPaymentStorageInSession, found := k.GetUserPaymentStorage(ctx, key)
	if !found {
		// is new entry
		userPaymentStorageInSession = types.UserPaymentStorage{Index: key, UniquePaymentStorageUserServicer: []*types.UniquePaymentStorageUserServicer{uniquePaymentStorageUserServicerEntryAddr}, TotalCU: usedCU, Session: session}
	} else {
		userPaymentStorageInSession.UniquePaymentStorageUserServicer = append(userPaymentStorageInSession.UniquePaymentStorageUserServicer, uniquePaymentStorageUserServicerEntryAddr)
		userPaymentStorageInSession.TotalCU += usedCU
	}
	k.SetUserPaymentStorage(ctx, userPaymentStorageInSession)
	return &userPaymentStorageInSession, nil
}
