package v4 // migrations.go

import (
	"log"

	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/v4/x/pairing/types"
)

const (
	ConflictVoteKeyPrefix = "ConflictVote/value/"
)

func DeleteOpenConflicts(ctx sdk.Context, storeKey storetypes.StoreKey, cdc codec.BinaryCodec) error {
	store := ctx.KVStore(storeKey)
	log.Println("@@@ REMOVING OLD STORAGE KEYS @@@")
	oldStore := prefix.NewStore(store, types.KeyPrefix(ConflictVoteKeyPrefix))
	iterator := storetypes.KVStorePrefixIterator(oldStore, []byte{})
	defer iterator.Close()
	for ; iterator.Valid(); iterator.Next() {
		log.Printf("@@@ Key: %s @@@", string(iterator.Key()))
		oldStore.Delete(iterator.Key()) // Delete old key, value
	}
	return nil
}
