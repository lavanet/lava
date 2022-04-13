package keeper

import (
	"fmt"
	"strconv"

	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/x/servicer/types"
)

// SetSessionPayments set a specific sessionPayments in the store from its index
func (k Keeper) SetSessionPayments(ctx sdk.Context, sessionPayments types.SessionPayments) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.SessionPaymentsKeyPrefix))
	b := k.cdc.MustMarshal(&sessionPayments)
	store.Set(types.SessionPaymentsKey(
		sessionPayments.Index,
	), b)
}

// GetSessionPayments returns a sessionPayments from its index
func (k Keeper) GetSessionPayments(
	ctx sdk.Context,
	index string,

) (val types.SessionPayments, found bool) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.SessionPaymentsKeyPrefix))

	b := store.Get(types.SessionPaymentsKey(
		index,
	))
	if b == nil {
		return val, false
	}

	k.cdc.MustUnmarshal(b, &val)
	return val, true
}

// RemoveSessionPayments removes a sessionPayments from the store
func (k Keeper) RemoveSessionPayments(
	ctx sdk.Context,
	index string,

) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.SessionPaymentsKeyPrefix))
	store.Delete(types.SessionPaymentsKey(
		index,
	))
}

// GetAllSessionPayments returns all sessionPayments
func (k Keeper) GetAllSessionPayments(ctx sdk.Context) (list []types.SessionPayments) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.SessionPaymentsKeyPrefix))
	iterator := sdk.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.SessionPayments
		k.cdc.MustUnmarshal(iterator.Value(), &val)
		list = append(list, val)
	}

	return
}

func (k Keeper) GetSessionPaymentsFromBlock(ctx sdk.Context, session uint64) (sessionPayment types.SessionPayments, found bool, key string) {
	key = strconv.FormatUint(session, 16)
	sessionPayment, found = k.GetSessionPayments(ctx, key)
	return
}

func (k Keeper) AddSessionPayment(ctx sdk.Context, session uint64, userAddress sdk.AccAddress, servicerAddress sdk.AccAddress, usedCU uint64, uniqueIdentifier string) (uint64, error) {
	userPaymentStorage, err := k.AddUserPaymentInSession(ctx, session, userAddress, servicerAddress, usedCU, uniqueIdentifier)
	if err != nil {
		return 0, fmt.Errorf("could not add session payment: %s,%s,%s,%d error: %s", userAddress, servicerAddress, uniqueIdentifier, session, err)
	}

	sessionPayments, found, key := k.GetSessionPaymentsFromBlock(ctx, session)
	if !found {
		sessionPayments = types.SessionPayments{Index: key, UsersPayments: []*types.UserPaymentStorage{userPaymentStorage}}
	} else {
		sessionPayments.UsersPayments = append(sessionPayments.UsersPayments, userPaymentStorage)
	}
	k.SetSessionPayments(ctx, sessionPayments)
	return userPaymentStorage.TotalCU, nil
}

func (k Keeper) RemoveAllSessionPaymentsForBlock(ctx sdk.Context, blockForDelete uint64) error {
	//remove the old sessions
	sessionPayments, found, key := k.GetSessionPaymentsFromBlock(ctx, blockForDelete)
	if !found {
		// return fmt.Errorf("did not find any sessionPayments for block %d", blockForDelete.Num)
		return nil
	}
	userPaymentsStorages := sessionPayments.UsersPayments
	for _, userPaymentStorage := range userPaymentsStorages {
		uniquePaymentStoragesCliSer := userPaymentStorage.UniquePaymentStorageUserServicer
		for _, uniquePaymentStorageCliSer := range uniquePaymentStoragesCliSer {
			//validate its an old entry, for sanity
			if uniquePaymentStorageCliSer.Block > blockForDelete {
				errMsg := "trying to delete a new entry in session payments for block"
				k.Logger(ctx).Error(errMsg)
				panic(errMsg)
			}
			//delete all payment storages
			k.RemoveUniquePaymentStorageUserServicer(ctx, uniquePaymentStorageCliSer.Index)
		}
		k.RemoveUserPaymentStorage(ctx, userPaymentStorage.Index)
	}
	k.RemoveSessionPayments(ctx, key)
	return nil
}
