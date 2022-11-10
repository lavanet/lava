package v4 // migrations.go

import (
	"fmt"
	"log"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/store/prefix"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/x/pairing/types"
)

const (
	// ClientPaymentStorageKeyPrefix is the prefix to retrieve all ClientPaymentStorage
	ClientPaymentStorageKeyPrefix = "ClientPaymentStorage/value/"
)

func DeleteAllClientPaymentStorageEntries(ctx sdk.Context, storeKey storetypes.StoreKey, cdc codec.BinaryCodec) error {
	store := ctx.KVStore(storeKey)
	log.Println("@@@ REMOVING OLD STORAGE KEYS @@@")
	oldStore := prefix.NewStore(store, types.KeyPrefix(ClientPaymentStorageKeyPrefix))
	iterator := sdk.KVStorePrefixIterator(oldStore, []byte{})
	defer iterator.Close()
	for ; iterator.Valid(); iterator.Next() {
		log.Println(fmt.Sprintf("@@@ Key: %s @@@", string(iterator.Key())))
		oldStore.Delete(iterator.Key()) // Delete old key, value
	}
	return nil
}
