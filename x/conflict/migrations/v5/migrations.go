package v4 // migrations.go

import (
	"log"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/store/prefix"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/v2/x/pairing/types"
)

const (
	ConflictVoteKeyPrefix = "ConflictVote/value/"
)

func DeleteOpenConflicts(ctx sdk.Context, storeKey storetypes.StoreKey, cdc codec.BinaryCodec) error {
	store := ctx.KVStore(storeKey)
	log.Println("@@@ REMOVING OLD STORAGE KEYS @@@")
	oldStore := prefix.NewStore(store, types.KeyPrefix(ConflictVoteKeyPrefix))
	iterator := sdk.KVStorePrefixIterator(oldStore, []byte{})
	defer iterator.Close()
	for ; iterator.Valid(); iterator.Next() {
		log.Printf("@@@ Key: %s @@@", string(iterator.Key()))
		oldStore.Delete(iterator.Key()) // Delete old key, value
	}
	return nil
}
